package backend

import (
	"fmt"
	"net/http"
	"path"

	"encoding/hex"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

func (s *Server) route() {
	s.r = mux.NewRouter().StrictSlash(true)
	s.r.Path("/").Methods("OPTIONS").HandlerFunc(s.handleProbe)
	s.r.Path("/robots.txt").HandlerFunc(s.handleRobotsTxt)
	s.r.Path("/metrics").Handler(
		prometheus.InstrumentHandler("metrics", prometheus.UninstrumentedHandler()))

	s.r.PathPrefix("/static/").Handler(
		prometheus.InstrumentHandler("static", http.StripPrefix("/static", http.HandlerFunc(s.handleStatic))))

	s.r.Handle("/", prometheus.InstrumentHandlerFunc("home", s.handleHomeStatic))

	s.r.PathPrefix("/about/").Handler(
		prometheus.InstrumentHandler("about", http.HandlerFunc(s.handleAboutStatic)))

	s.r.HandleFunc("/room/{room:[a-z0-9]+}/ws", instrumentSocketHandlerFunc("ws", s.handleRoom))
	s.r.Handle(
		"/room/{room:[a-z0-9]+}/", prometheus.InstrumentHandlerFunc("room_static", s.handleRoomStatic))

	s.r.Handle(
		"/prefs/reset-password",
		prometheus.InstrumentHandlerFunc("prefsResetPassword", s.handleResetPassword))
	s.r.Handle(
		"/prefs/verify", prometheus.InstrumentHandlerFunc("prefsVerify", s.handlePrefsVerify))
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
	s.serveGzippedFile(w, r, path.Clean(r.URL.Path), true)
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
	s.serveGzippedFile(w, r, "room.html", false)
}

func (s *Server) handleHomeStatic(w http.ResponseWriter, r *http.Request) {
	s.serveGzippedFile(w, r, "home.html", false)
}

func (s *Server) handleAboutStatic(w http.ResponseWriter, r *http.Request) {
	if s.staticPath == "" || r.URL.Path == "" {
		http.NotFound(w, r)
		return
	}
	s.serveGzippedFile(w, r, path.Clean(r.URL.Path) + ".html", false)
}

func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	s.serveGzippedFile(w, r, "robots.txt", false)
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	ctx := s.rootCtx.Fork()

	// Resolve the room.
	// TODO: support room creation?
	roomName := mux.Vars(r)["room"]
	room, err := s.b.GetRoom(ctx, roomName)
	if s.allowRoomCreation && err == proto.ErrRoomNotFound {
		room, err = s.b.CreateRoom(ctx, s.kms, false, roomName)
	}
	if err != nil {
		if err == proto.ErrRoomNotFound {
			http.Error(w, "404 page not found", http.StatusNotFound)
			return
		}
		Logger(ctx).Printf("room error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Tag the agent. We use an authenticated but un-encrypted cookie.
	agent, cookie, agentKey, err := getAgent(ctx, s, r)
	if err != nil {
		Logger(ctx).Printf("get agent error: %s", err)
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
				// allow session to proceed, but agent will not be logged into account
				agent.AccountID = ""
			default:
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}

	// Upgrade to a websocket and set cookie.
	headers := http.Header{}
	if cookie != nil {
		headers.Add("Set-Cookie", cookie.String())
	}
	conn, err := upgrader.Upgrade(w, r, headers)
	if err != nil {
		Logger(ctx).Printf("upgrade error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Serve the session.
	session := newSession(ctx, s, conn, roomName, room, client, agentKey)
	if err = session.serve(); err != nil {
		// TODO: error handling
		Logger(ctx).Printf("session serve error: %s", err)
		return
	}
}

func (s *Server) handlePrefsVerify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	email := r.Form.Get("email")
	token, err := hex.DecodeString(r.Form.Get("token"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if email == "" || len(token) == 0 {
		http.Error(w, "missing parameters", http.StatusBadRequest)
		return
	}

	ctx := s.rootCtx.Fork()
	account, err := s.b.AccountManager().Resolve(ctx, "email", email)
	if err != nil {
		status := http.StatusInternalServerError
		if err == proto.ErrAccountNotFound {
			status = http.StatusNotFound
		}
		http.Error(w, err.Error(), status)
		return
	}

	if err := proto.CheckEmailVerificationToken(s.kms, account, email, token); err != nil {
		status := http.StatusInternalServerError
		if err == proto.ErrInvalidVerificationToken {
			status = http.StatusForbidden
		}
		http.Error(w, err.Error(), status)
		return
	}

	if err := s.b.AccountManager().VerifyPersonalIdentity(ctx, "email", email); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// TODO: serve success template
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch r.Method {
	case "GET":
		// TODO: serve password reset template
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<form method="POST"><input type="hidden" name="confirmation" value="%s">`,
			r.Form.Get("confirmation"))
		fmt.Fprintf(w, `<input type="password" name="password"></form>`)
	case "POST":
		s.handleResetPasswordPost(w, r)
	default:
		http.Error(w, "invalid method", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	confirmation := r.PostForm.Get("confirmation")
	password := r.PostForm.Get("password")

	ctx := s.rootCtx.Fork()
	if err := s.b.AccountManager().ConfirmPasswordReset(ctx, s.kms, confirmation, password); err != nil {
		status := http.StatusInternalServerError
		if err == proto.ErrInvalidConfirmationCode {
			status = http.StatusBadRequest
		}
		http.Error(w, err.Error(), status)
		return
	}

	// TODO: serve success template
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("ok"))
}
