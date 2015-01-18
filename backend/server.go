package backend

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"golang.org/x/net/context"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	Subprotocols:    []string{"heim1"},
}

type Server struct {
	r          *mux.Router
	b          Backend
	staticPath string
}

func NewServer(backend Backend, staticPath string) *Server {
	s := &Server{
		b:          backend,
		staticPath: staticPath,
	}
	s.route()
	return s
}

func (s *Server) route() {
	s.r = mux.NewRouter()
	s.r.Path("/").Methods("OPTIONS").HandlerFunc(s.handleProbe)
	s.r.HandleFunc("/room/{room:[a-z0-9]+}/ws", s.handleRoom)
	s.r.PathPrefix("/room/{room:[a-z0-9]+}/").HandlerFunc(s.handleStatic)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) handleProbe(w http.ResponseWriter, r *http.Request) {
	// TODO: determine if we're really healthy
	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.staticPath == "" {
		http.NotFound(w, r)
		return
	}

	roomName := mux.Vars(r)["room"]
	handler := http.StripPrefix(
		"/room/"+roomName+"/", http.FileServer(http.Dir(s.staticPath)))
	handler.ServeHTTP(w, r)
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

	session := newMemSession(ctx, conn, room)

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
