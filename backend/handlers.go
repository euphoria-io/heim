package backend

import (
	"fmt"
	"net/http"
	"path"

	"encoding/hex"
	"encoding/json"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
)

func (s *Server) route() {
	s.r = mux.NewRouter().StrictSlash(true)
	s.r.NotFoundHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.serveErrorPage("page not found", http.StatusNotFound, w, r)
	})

	s.r.Path("/").Methods("OPTIONS").HandlerFunc(s.handleProbe)
	s.r.Path("/robots.txt").HandlerFunc(s.handleRobotsTxt)
	s.r.Path("/metrics").Handler(
		prometheus.InstrumentHandler("metrics", prometheus.UninstrumentedHandler()))

	s.r.PathPrefix("/static/").Handler(
		prometheus.InstrumentHandler("static", http.HandlerFunc(s.handleStatic)))

	s.r.Handle("/", prometheus.InstrumentHandlerFunc("home", s.handleHomeStatic))

	s.r.PathPrefix("/about").Handler(
		prometheus.InstrumentHandler("about", http.HandlerFunc(s.handleAboutStatic)))

	s.r.HandleFunc("/room/{prefix:(pm:)?}{room:[a-z0-9]+}/ws", instrumentSocketHandlerFunc("ws", s.handleRoom))
	s.r.Handle(
		"/room/{prefix:(pm:)?}{room:[a-z0-9]+}/", prometheus.InstrumentHandlerFunc("room_static", s.handleRoomStatic))

	s.r.Handle(
		"/prefs/reset-password",
		prometheus.InstrumentHandlerFunc("prefsResetPassword", s.handlePrefsResetPassword))
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
	roomName := mux.Vars(r)["room"]
	_, err := s.b.GetRoom(scope.New(), roomName)
	if err != nil {
		if err == proto.ErrRoomNotFound {
			if !s.allowRoomCreation {
				s.serveErrorPage("room not found", http.StatusNotFound, w, r)
				return
			}
		} else {
			s.serveErrorPage(err.Error(), http.StatusInternalServerError, w, r)
			return
		}
	}
	params := map[string]interface{}{"RoomName": roomName}
	s.servePage("room.html", params, w, r)
}

func (s *Server) handleHomeStatic(w http.ResponseWriter, r *http.Request) {
	s.serveGzippedFile(w, r, "/pages/home.html", false)
}

func (s *Server) handleAboutStatic(w http.ResponseWriter, r *http.Request) {
	if s.staticPath == "" || r.URL.Path == "" {
		s.servePage("about", nil, w, r)
		return
	}
	s.servePage(path.Clean(r.URL.Path)[1:]+".html", nil, w, r)
}

func (s *Server) handleRobotsTxt(w http.ResponseWriter, r *http.Request) {
	s.serveGzippedFile(w, r, "/static/robots.txt", false)
}

func (s *Server) resolveRoom(ctx scope.Context, prefix, roomName string, client *proto.Client) (room proto.Room, err error) {
	// TODO: support room creation?
	switch prefix {
	case "pm:":
		var (
			sf      snowflake.Snowflake
			roomKey *security.ManagedKey
		)
		if err := sf.FromString(roomName); err != nil {
			return nil, proto.ErrRoomNotFound
		}
		room, roomKey, err = s.b.PMTracker().Room(ctx, s.kms, sf, client)
		if err != nil {
			return nil, err
		}
		client.Authorization.AddMessageKey("pm:"+roomName, roomKey)
		return room, nil
	case "":
		room, err = s.b.GetRoom(ctx, roomName)
		if s.allowRoomCreation && err == proto.ErrRoomNotFound {
			room, err = s.b.CreateRoom(ctx, s.kms, false, roomName)
		}
		if err != nil {
			return nil, err
		}
		if err := client.RoomAuthorize(ctx, room); err != nil {
			return nil, err
		}
		return room, nil
	default:
		return nil, proto.ErrRoomNotFound
	}
}

func (s *Server) handleRoom(w http.ResponseWriter, r *http.Request) {
	ctx := s.rootCtx.Fork()

	// Tag the agent. We use an authenticated but un-encrypted cookie.
	agent, cookie, agentKey, err := getAgent(ctx, s, r)
	if err != nil {
		logging.Logger(ctx).Printf("get agent error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	client := &proto.Client{
		Agent: agent,
		Authorization: proto.Authorization{
			ClientKey: agentKey,
		},
	}
	client.FromRequest(ctx, r)

	// Look up account associated with agent.
	var accountID snowflake.Snowflake
	if err := accountID.FromString(agent.AccountID); agent.AccountID != "" && err == nil {
		if err := client.AuthenticateWithAgent(ctx, s.b, agent, agentKey); err != nil {
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

	// Resolve the room.
	prefix := mux.Vars(r)["prefix"]
	roomName := mux.Vars(r)["room"]
	room, err := s.resolveRoom(ctx, prefix, roomName, client)
	if err != nil {
		switch err {
		case proto.ErrAccessDenied:
			http.Error(w, "401 unauthorized", http.StatusUnauthorized)
			return
		case proto.ErrRoomNotFound:
			http.Error(w, "404 page not found", http.StatusNotFound)
			return
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Serve the room websocket.
	s.serveRoomWebsocket(ctx, room, cookie, client, agentKey, w, r)
}

func (s *Server) serveRoomWebsocket(
	ctx scope.Context, room proto.Room,
	cookie *http.Cookie, client *proto.Client, agentKey *security.ManagedKey,
	w http.ResponseWriter, r *http.Request) {

	// Upgrade to a websocket and set cookie.
	headers := http.Header{}
	if cookie != nil {
		headers.Add("Set-Cookie", cookie.String())
	}
	conn, err := upgrader.Upgrade(w, r, headers)
	if err != nil {
		logging.Logger(ctx).Printf("upgrade error: %s", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	// Serve the session.
	session := newSession(ctx, s, conn, room, client, agentKey)
	if err = session.serve(); err != nil {
		// TODO: error handling
		logging.Logger(ctx).Printf("session serve error: %s", err)
		return
	}
}

func (s *Server) handlePrefsVerify(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorPage("bad request", http.StatusBadRequest, w, r)
		return
	}

	switch r.Method {
	case "GET":
		confirmation := r.Form.Get("token")
		email := r.Form.Get("email")
		params := map[string]interface{}{
			"confirmation": confirmation,
			"email":        email,
		}
		s.servePage(VerifyEmailPage, params, w, r)
	case "POST":
		s.handlePrefsVerifyPost(w, r)
	default:
		s.serveErrorPage("invalid method", http.StatusMethodNotAllowed, w, r)
	}
}

func (s *Server) handlePrefsVerifyPost(w http.ResponseWriter, r *http.Request) {
	reply := func(err error, status int) {
		data := struct {
			Error string `json:"error,omitempty"`
		}{}
		if err != nil {
			data.Error = err.Error()
		}
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}

	var req struct {
		Confirmation string `json:"confirmation"`
		Email        string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reply(err, http.StatusBadRequest)
		return
	}

	email := req.Email
	token, err := hex.DecodeString(req.Confirmation)
	if err != nil {
		reply(err, http.StatusBadRequest)
		return
	}
	if email == "" || len(token) == 0 {
		reply(fmt.Errorf("missing parameters"), http.StatusBadRequest)
		return
	}

	ctx := s.rootCtx.Fork()
	account, err := s.b.AccountManager().Resolve(ctx, "email", email)
	if err != nil {
		status := http.StatusInternalServerError
		if err == proto.ErrAccountNotFound {
			status = http.StatusNotFound
		}
		reply(err, status)
		return
	}

	if err := proto.CheckEmailVerificationToken(s.kms, account, email, token); err != nil {
		status := http.StatusInternalServerError
		if err == proto.ErrInvalidVerificationToken {
			status = http.StatusForbidden
		}
		reply(err, status)
		return
	}

	if err := s.b.AccountManager().VerifyPersonalIdentity(ctx, "email", email); err != nil {
		reply(err, http.StatusInternalServerError)
		return
	}

	reply(nil, http.StatusOK)
}

func (s *Server) handlePrefsResetPassword(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		s.serveErrorPage(err.Error(), http.StatusBadRequest, w, r)
		return
	}

	switch r.Method {
	case "GET":
		confirmation := r.Form.Get("confirmation")
		ctx := s.rootCtx.Fork()
		account, err := s.b.AccountManager().GetPasswordResetAccount(ctx, confirmation)
		switch err {
		case nil:
			params := map[string]interface{}{
				"confirmation": confirmation,
				"email":        "unknown",
			}
			for _, pid := range account.PersonalIdentities() {
				if pid.Namespace() == "email" {
					params["email"] = pid.ID()
					break
				}
			}
			s.servePage(ResetPasswordPage, params, w, r)
		case proto.ErrInvalidConfirmationCode:
			s.serveErrorPage("invalid/expired confirmation code", http.StatusBadRequest, w, r)
		default:
			s.serveErrorPage(err.Error(), http.StatusInternalServerError, w, r)
		}
	case "POST":
		s.handlePrefsResetPasswordPost(w, r)
	default:
		s.serveErrorPage("invalid method", http.StatusMethodNotAllowed, w, r)
	}
}

func (s *Server) handlePrefsResetPasswordPost(w http.ResponseWriter, r *http.Request) {
	reply := func(err error, status int) {
		data := struct {
			Error string `json:"error,omitempty"`
		}{}
		if err != nil {
			data.Error = err.Error()
		}
		w.WriteHeader(status)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	}

	var req struct {
		Password struct {
			Text     string `json:"text"`
			Strength string `json:"strength"`
		} `json:"password"`
		Confirmation string `json:"confirmation"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		reply(err, http.StatusBadRequest)
		return
	}

	ctx := s.rootCtx.Fork()
	if err := s.b.AccountManager().ConfirmPasswordReset(ctx, s.kms, req.Confirmation, req.Password.Text); err != nil {
		status := http.StatusInternalServerError
		if err == proto.ErrInvalidConfirmationCode {
			status = http.StatusBadRequest
		}
		reply(err, status)
	} else {
		reply(nil, http.StatusOK)
	}
}
