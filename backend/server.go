package backend

import (
	"crypto/rand"
	"fmt"
	"net/http"
	"path"
	"sync"
	"time"

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

	m sync.Mutex

	agentIDGenerator func() ([]byte, error)
}

func NewServer(
	backend proto.Backend, cluster cluster.Cluster, kms security.KMS, id, era, staticPath string) (
	*Server, error) {

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
	}
	s.route()
	return s, nil
}

func (s *Server) route() {
	s.r = mux.NewRouter().StrictSlash(true)
	s.r.Path("/").Methods("OPTIONS").HandlerFunc(s.handleProbe)
	s.r.Path("/robots.txt").HandlerFunc(s.handleRobotsTxt)
	s.r.Path("/metrics").Handler(
		prometheus.InstrumentHandler("metrics", prometheus.UninstrumentedHandler()))

	s.r.PathPrefix("/static/").Handler(prometheus.InstrumentHandlerFunc("static", s.handleStatic))

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
	if s.staticPath == "" || r.URL.Path == "/static/" {
		http.NotFound(w, r)
		return
	}

	handler := http.StripPrefix("/static/", http.FileServer(http.Dir(s.staticPath)))
	handler.ServeHTTP(w, r)
}

func (s *Server) handleRoomStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join(s.staticPath, "index.html"))
}

func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, path.Join(s.staticPath, "robots.txt"))
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
	ctx := scope.New()
	logger := Logger(ctx)

	// Resolve the room.
	roomName := mux.Vars(r)["room"]
	room, err := s.b.GetRoom(roomName)
	if err != nil {
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
	session := newSession(ctx, conn, s.ID, s.Era, room, agentID)
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
