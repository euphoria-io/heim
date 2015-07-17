package backend

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"time"

	"euphoria.io/heim/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	gorillactx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
)

const cookieKeySize = 32

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"heim1"},
	CheckOrigin:     checkOrigin,
}

type Server struct {
	ID         string
	Era        string
	r          *mux.Router
	b          proto.Backend
	kms        security.KMS
	staticPath string
	sc         *securecookie.SecureCookie
	rootCtx    scope.Context

	allowRoomCreation     bool
	newAccountMinAgentAge time.Duration

	m sync.Mutex

	agentIDGenerator func() ([]byte, error)
}

func NewServer(
	ctx scope.Context, backend proto.Backend, cluster cluster.Cluster, kms security.KMS,
	id, era, staticPath string) (*Server, error) {

	mime.AddExtensionType(".map", "application/json")

	cookieSecret, err := cluster.GetSecret(kms, "cookie", cookieKeySize)
	if err != nil {
		return nil, fmt.Errorf("error coordinating shared cookie secret: %s", err)
	}

	s := &Server{
		ID:         id,
		Era:        era,
		b:          backend,
		kms:        kms,
		staticPath: staticPath,
		sc:         securecookie.New(cookieSecret, nil),
		rootCtx:    ctx,
	}
	s.route()
	return s, nil
}

func (s *Server) AllowRoomCreation(allow bool)            { s.allowRoomCreation = allow }
func (s *Server) NewAccountMinAgentAge(age time.Duration) { s.newAccountMinAgentAge = age }

func (s *Server) route() {
	s.r = mux.NewRouter().StrictSlash(true)
	s.r.Path("/").Methods("OPTIONS").HandlerFunc(s.handleProbe)
	s.r.Path("/robots.txt").HandlerFunc(s.handleRobotsTxt)
	s.r.Path("/metrics").Handler(
		prometheus.InstrumentHandler("metrics", prometheus.UninstrumentedHandler()))

	s.r.PathPrefix("/static/").Handler(
		prometheus.InstrumentHandler("static", http.StripPrefix("/static", http.HandlerFunc(s.handleStatic))))

	s.r.Handle("/", prometheus.InstrumentHandlerFunc("home", s.handleHomeStatic))

	s.r.HandleFunc("/room/{room:[a-z0-9]+}/ws", instrumentSocketHandlerFunc("ws", s.handleRoom))
	s.r.Handle(
		"/room/{room:[a-z0-9]+}/", prometheus.InstrumentHandlerFunc("room_static", s.handleRoomStatic))
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	// TODO: determine if we're really healthy
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.staticPath == "" || r.URL.Path == "" {
		http.NotFound(w, r)
		return
	}
	s.serveGzippedFile(w, r, path.Clean(r.URL.Path))
}

func (s *Server) handleRoomStatic(w http.ResponseWriter, r *http.Request) {
	if !s.allowRoomCreation {
		roomName := mux.Vars(r)["room"]
		_, err := s.b.GetRoom(scope.New(), roomName)
		if err != nil {
			if err == proto.ErrRoomNotFound {
				http.Error(w, "404 page not found", http.StatusNotFound)
				return
			}
		}
	}
	s.serveGzippedFile(w, r, "index.html")
}

func (s *Server) handleHomeStatic(w http.ResponseWriter, r *http.Request) {
	s.serveGzippedFile(w, r, "home.html")
}

func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	s.serveGzippedFile(w, r, "robots.txt")
}

type gzipResponseWriter struct {
	http.ResponseWriter
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	header := w.Header()
	header.Set("Content-Encoding", "gzip")
	header.Add("Vary", "Accept-Encoding")
	w.ResponseWriter.WriteHeader(code)
}

func (s *Server) serveGzippedFile(w http.ResponseWriter, r *http.Request, filename string) {
	dir := http.Dir(s.staticPath)
	var err error
	var f http.File
	gzipped := false

	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		f, err = dir.Open(filename + ".gz")
		if err != nil {
			f = nil
		} else {
			gzipped = true
		}
	}

	if f == nil {
		f, err = dir.Open(filename)
		if err != nil {
			http.NotFound(w, r)
			return
		}
	}

	defer f.Close()

	d, err := f.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}

	name := d.Name()
	if gzipped {
		name = strings.TrimSuffix(name, ".gz")
		w = &gzipResponseWriter{ResponseWriter: w}
	}

	http.ServeContent(w, r, name, d.ModTime(), f)
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	ctx := s.rootCtx.Fork()

	// Resolve the room.
	// TODO: support room creation?
	roomName := mux.Vars(r)["room"]
	room, err := s.b.GetRoom(ctx, roomName)
	if err != nil {
		if err == proto.ErrRoomNotFound {
			http.Error(w, "404 page not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Tag the agent. We use an authenticated but un-encrypted cookie.
	agent, cookie, agentKey, err := getAgent(ctx, s, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := &proto.Client{Agent: agent}
	client.FromRequest(ctx, r)

	// Look up account associated with agent.
	var accountID snowflake.Snowflake
	if err := accountID.FromString(agent.AccountID); agent.AccountID != "" && err == nil {
		if err := client.AuthenticateWithAgent(ctx, s.b, room, agent, agentKey); err != nil {
			fmt.Printf("agent auth failed: %s\n", err)
			switch err {
			case proto.ErrAccessDenied:
				http.Error(w, err.Error(), http.StatusForbidden)
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}
	}

	// Upgrade to a websocket and set cookie.
	headers := http.Header{}
	if cookie != nil {
		headers.Add("Set-Cookie", cookie.String())
	}
	conn, err := upgrader.Upgrade(w, r, headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Serve the session.
	session := newSession(ctx, s, conn, roomName, room, client, agentKey)
	if err = session.serve(); err != nil {
		// TODO: error handling
		return
	}
}

type hijackResponseWriter struct {
	http.ResponseWriter
	http.Hijacker
}

func instrumentSocketHandlerFunc(name string, handler http.HandlerFunc) http.HandlerFunc {
	type hijackerKey int
	var k hijackerKey

	loadHijacker := func(w http.ResponseWriter, r *http.Request) {
		if hj, ok := gorillactx.GetOk(r, k); ok {
			w = &hijackResponseWriter{ResponseWriter: w, Hijacker: hj.(http.Hijacker)}
		}
		handler(w, r)
	}

	promHandler := prometheus.InstrumentHandlerFunc(name, loadHijacker)

	saveHijacker := func(w http.ResponseWriter, r *http.Request) {
		if hj, ok := w.(http.Hijacker); ok {
			gorillactx.Set(r, k, hj)
		}
		promHandler(w, r)
	}

	return saveHijacker
}

func checkOrigin(r *http.Request) bool {
	origin := r.Header["Origin"]

	// If no Origin header was given, accept.
	if len(origin) == 0 {
		return true
	}

	// Try to parse Origin, and reject if there's an error.
	u, err := url.Parse(origin[0])
	if err != nil {
		return false
	}

	// If Origin matches any of these prefix/requested-host combinations, accept.
	for _, prefix := range []string{"", "www."} {
		if u.Host == prefix+r.Host {
			return true
		}
	}

	// If we reach this point, reject the Origin.
	return false
}
