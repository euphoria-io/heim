package backend

import (
	"net/http"
	"path"

	"heim/proto"

	gorillactx "github.com/gorilla/context"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"

	"golang.org/x/net/context"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"heim1"},
}

type Server struct {
	ID         string
	r          *mux.Router
	b          proto.Backend
	staticPath string
}

func NewServer(backend proto.Backend, id, staticPath string) *Server {
	s := &Server{
		ID:         id,
		b:          backend,
		staticPath: staticPath,
	}
	s.route()
	return s
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

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	logger := Logger(ctx)

	roomName := mux.Vars(r)["room"]
	room, err := s.b.GetRoom(roomName)
	if err != nil {
		logger.Printf("get room %s: %s", roomName, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	session := newMemSession(ctx, conn, s.ID, room)

	if err := session.sendSnapshot(); err != nil {
		logger.Printf("snapshot failed: %s", err)
		// TODO: send an error packet
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = room.Join(ctx, session)
	if err != nil {
		// TODO: error handling
		return
	}

	defer func() {
		if err := room.Part(ctx, session); err != nil {
			// TODO: error handling
			return
		}
	}()

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
