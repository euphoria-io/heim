package backend

import (
	"crypto/rand"
	"fmt"
	"mime"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"

	"encoding/json"

	"euphoria.io/heim/backend/cluster"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"

	gorillactx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	cookieKeySize = 32
	agentIDSize   = 8

	agentCookie = "a"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"heim1"},
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
	media      *MediaDispatcher

	allowRoomCreation bool

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

func (s *Server) AllowRoomCreation(allow bool) { s.allowRoomCreation = allow }

func (s *Server) SetMediaDispatcher(dispatcher *MediaDispatcher) { s.media = dispatcher }
func (s *Server) MediaDispatcher() *MediaDispatcher              { return s.media }

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
	s.r.HandleFunc(
		"/room/{room:[a-z0-9]+}/media", prometheus.InstrumentHandlerFunc("media", s.handleMedia)).
		Methods("POST")
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
		_, err := s.b.GetRoom(roomName, false)
		if err != nil {
			if err.Error() == "no such room" {
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

func (s *Server) generateAgentID() ([]byte, error) {
	if s.agentIDGenerator != nil {
		return s.agentIDGenerator()
	}

	agentID := make([]byte, agentIDSize)
	if _, err := rand.Read(agentID); err != nil {
		return nil, err
	}
	return agentID, nil
}

func (s *Server) setAgentID(w http.ResponseWriter) ([]byte, *http.Cookie, error) {
	agentID, err := s.generateAgentID()
	if err != nil {
		return nil, nil, err
	}

	encoded, err := s.sc.Encode(agentCookie, agentID)
	if err != nil {
		return nil, nil, err
	}

	cookie := &http.Cookie{
		Name:     agentCookie,
		Value:    encoded,
		Path:     "/",
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
	}
	return agentID, cookie, nil
}

func (s *Server) getAgentID(w http.ResponseWriter, r *http.Request) ([]byte, *http.Cookie, error) {
	agentID := []byte{}
	cookie, err := r.Cookie(agentCookie)
	if err != nil {
		return s.setAgentID(w)
	}
	if err := s.sc.Decode(agentCookie, cookie.Value, &agentID); err != nil {
		return s.setAgentID(w)
	}
	return agentID, nil, nil
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	ctx := s.rootCtx.Fork()
	logger := Logger(ctx)

	// Resolve the room.
	roomName := mux.Vars(r)["room"]
	room, err := s.b.GetRoom(roomName, s.allowRoomCreation)
	if err != nil {
		if err.Error() == "no such room" {
			http.Error(w, "404 page not found", http.StatusNotFound)
			return
		}
		logger.Printf("get room %s: %s", roomName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Tag the agent. We use an authenticated but un-encrypted cookie.
	agentID, cookie, err := s.getAgentID(w, r)
	if err != nil {
		logger.Printf("get agent id: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := &proto.Client{AgentID: fmt.Sprintf("%x", agentID)}
	client.FromRequest(ctx, r)

	// Upgrade to a websocket.
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
	session := newSession(ctx, conn, s.ID, s.Era, s.media, roomName, room, agentID)
	if err = session.serve(); err != nil {
		// TODO: error handling
		return
	}
}

func (s *Server) handleMedia(w http.ResponseWriter, r *http.Request) {
	ctx := scope.New()
	logger := Logger(ctx)

	// Parse the request.
	if r.Header.Get("Content-type") != "application/json" {
		http.Error(w, "content-type must be application/json", http.StatusBadRequest)
		return
	}
	t := proto.Transcoding{}
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Resolve the room.
	roomName := mux.Vars(r)["room"]
	room, err := s.b.GetRoom(roomName, false)
	if err != nil {
		logger.Printf("get room %s: %s", roomName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save the transcoding metadata.
	if err := room.AddMediaTranscoding(ctx, &t); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
