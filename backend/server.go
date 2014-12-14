package backend

import (
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"golang.org/x/net/context"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type Server struct {
	sync.Mutex
	r          *mux.Router
	staticPath string
	rooms      map[string]Room
}

func NewServer(staticPath string) *Server {
	s := &Server{
		staticPath: staticPath,
		rooms:      map[string]Room{},
	}
	s.route()
	return s
}

func (s *Server) route() {
	s.r = mux.NewRouter()
	s.r.HandleFunc("/room/{room:[a-z0-9]+}/ws", s.handleRoom)
	if s.staticPath != "" {
		s.r.PathPrefix("/room/{room:[a-z0-9]+}/").Handler(http.FileServer(http.Dir(s.staticPath)))
	}
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.r.ServeHTTP(w, r)
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	roomName := mux.Vars(r)["room"]

	s.Lock()
	room, ok := s.rooms[roomName]
	if !ok {
		room = newMemRoom(roomName)
		s.rooms[roomName] = room
	}
	s.Unlock()

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer conn.Close()

	ctx := context.Background()
	session := newMemSession(ctx, conn, room)
	err = room.Join(ctx, session)
	if err != nil {
		// TODO: error handling
		return
	}

	session.serve()
	err = room.Part(ctx, session)
	if err != nil {
		// TODO: error handling
		return
	}
}
