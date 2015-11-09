package backend

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/png"
	"time"

	"golang.org/x/net/context"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/logging"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
)

const authDelay = 2 * time.Second

func (s *session) ignoreState(cmd *proto.Packet) *response {
	switch cmd.Type {
	case proto.PingType, proto.PingReplyType:
		return s.joinedState(cmd)
	default:
		return &response{}
	}
}

func (s *session) unauthedState(cmd *proto.Packet) *response {
	payload, err := cmd.Payload()
	if err != nil {
		return &response{err: fmt.Errorf("payload: %s", err)}
	}

	switch msg := payload.(type) {
	case *proto.AuthCommand:
		return s.handleAuthCommand(msg)
	case *proto.StaffInvadeCommand:
		return s.handleStaffInvadeCommand(msg)
	default:
		if resp := s.handleCoreCommands(payload); resp != nil {
			return resp
		}
		return &response{err: fmt.Errorf("access denied, please authenticate")}
	}
}

func (s *session) joinedState(cmd *proto.Packet) *response {
	payload, err := cmd.Payload()
	if err != nil {
		return &response{err: fmt.Errorf("payload: %s", err)}
	}

	switch msg := payload.(type) {
	case *proto.AuthCommand:
		return &response{err: fmt.Errorf("already joined")}
	case *proto.SendCommand:
		return s.handleSendCommand(msg)
	case *proto.GetMessageCommand:
		ret, err := s.room.GetMessage(s.ctx, msg.ID)
		if err != nil {
			return &response{err: err}
		}
		packet, err := proto.DecryptPayload(proto.GetMessageReply(*ret), &s.client.Authorization, s.privilegeLevel())
		return &response{
			packet: packet,
			err:    err,
			cost:   1,
		}
	case *proto.LogCommand:
		msgs, err := s.room.Latest(s.ctx, msg.N, msg.Before)
		if err != nil {
			return &response{err: err}
		}
		packet, err := proto.DecryptPayload(
			proto.LogReply{Log: msgs, Before: msg.Before}, &s.client.Authorization, s.privilegeLevel())
		return &response{
			packet: packet,
			err:    err,
			cost:   1,
		}
	case *proto.NickCommand:
		nick, err := proto.NormalizeNick(msg.Name)
		if err != nil {
			return &response{err: err}
		}
		formerName := s.identity.Name()
		s.identity.name = nick
		event, err := s.room.RenameUser(s.ctx, s, formerName)
		if err != nil {
			return &response{err: err}
		}
		return &response{
			packet: proto.NickReply(*event),
			cost:   1,
		}
	case *proto.WhoCommand:
		listing, err := s.room.Listing(s.ctx, s.privilegeLevel())
		if err != nil {
			return &response{err: err}
		}
		return &response{packet: &proto.WhoReply{Listing: listing}}
	default:
		if resp := s.handleCoreCommands(payload); resp != nil {
			return resp
		}
		return &response{err: fmt.Errorf("command type %T not implemented", payload)}
	}
}

func (s *session) handleCoreCommands(payload interface{}) *response {
	switch msg := payload.(type) {
	// pings
	case *proto.PingCommand:
		return &response{packet: &proto.PingReply{UnixTime: msg.UnixTime}}
	case *proto.PingReply:
		s.finishFastKeepAlive()
		if time.Time(msg.UnixTime).Unix() == s.expectedPingReply {
			s.outstandingPings = 0
		} else if s.outstandingPings > 1 {
			s.outstandingPings--
		}
		return &response{}

	// account management commands
	case *proto.ChangeEmailCommand:
		return s.handleChangeEmailCommand(msg)
	case *proto.ChangeNameCommand:
		return s.handleChangeNameCommand(msg)
	case *proto.ChangePasswordCommand:
		return s.handleChangePasswordCommand(msg)
	case *proto.LoginCommand:
		return s.handleLoginCommand(msg)
	case *proto.LogoutCommand:
		return s.handleLogoutCommand()
	case *proto.RegisterAccountCommand:
		return s.handleRegisterAccountCommand(msg)
	case *proto.ResendVerificationEmailCommand:
		return s.handleResendVerificationEmail(msg)
	case *proto.ResetPasswordCommand:
		return s.handleResetPasswordCommand(msg)

	// room manager commands
	case *proto.BanCommand:
		return s.handleBanCommand(msg)
	case *proto.UnbanCommand:
		return s.handleUnbanCommand(msg)
	case *proto.EditMessageCommand:
		return s.handleEditMessageCommand(msg)
	case *proto.GrantAccessCommand:
		return s.handleGrantAccessCommand(msg)
	case *proto.GrantManagerCommand:
		return s.handleGrantManagerCommand(msg)
	case *proto.RevokeManagerCommand:
		return s.handleRevokeManagerCommand(msg)
	case *proto.RevokeAccessCommand:
		return s.handleRevokeAccessCommand(msg)

	// staff commands
	case *proto.StaffCreateRoomCommand:
		return s.handleStaffCreateRoomCommand(msg)
	case *proto.StaffGrantManagerCommand:
		return s.handleStaffGrantManagerCommand(msg)
	case *proto.StaffRevokeManagerCommand:
		return s.handleStaffRevokeManagerCommand(msg)
	case *proto.StaffRevokeAccessCommand:
		return s.handleStaffRevokeAccessCommand(msg)
	case *proto.StaffLockRoomCommand:
		return s.handleStaffLockRoomCommand()
	case *proto.StaffEnrollOTPCommand:
		return s.handleStaffEnrollOTPCommand(msg)
	case *proto.StaffValidateOTPCommand:
		return s.handleStaffValidateOTPCommand(msg)
	case *proto.StaffInspectIPCommand:
		return s.handleStaffInspectIPCommand(msg)
	case *proto.StaffInvadeCommand:
		return s.handleStaffInvadeCommand(msg)
	case *proto.UnlockStaffCapabilityCommand:
		return s.handleUnlockStaffCapabilityCommand(msg)

	// fallthrough
	default:
		return nil
	}
}

func (s *session) handleSendCommand(cmd *proto.SendCommand) *response {
	if s.Identity().Name() == "" {
		return &response{err: fmt.Errorf("you must choose a name before you may begin chatting")}
	}

	if len(cmd.Content) > proto.MaxMessageLength {
		return &response{err: proto.ErrMessageTooLong}
	}

	msgID, err := snowflake.New()
	if err != nil {
		return &response{err: err}
	}

	isValidParent, err := s.room.IsValidParent(cmd.Parent)
	if err != nil {
		return &response{err: err}
	}
	if !isValidParent {
		return &response{err: proto.ErrInvalidParent}
	}
	msg := proto.Message{
		ID:      msgID,
		Content: cmd.Content,
		Parent:  cmd.Parent,
		Sender:  s.View(proto.Host),
	}

	if s.keyID != "" {
		key := s.client.Authorization.MessageKeys[s.keyID]
		if err := proto.EncryptMessage(&msg, s.keyID, key); err != nil {
			return &response{err: err}
		}
	}

	sent, err := s.room.Send(s.ctx, s, msg)
	if err != nil {
		return &response{err: err}
	}

	if s.privilegeLevel() == proto.General {
		sent.Sender.ClientAddress = ""
	}

	packet, err := proto.DecryptPayload(proto.SendReply(sent), &s.client.Authorization, s.privilegeLevel())
	return &response{
		packet: packet,
		err:    err,
		cost:   10,
	}
}

func (s *session) handleGrantAccessCommand(cmd *proto.GrantAccessCommand) *response {
	mkp := s.client.Authorization.ManagerKeyPair
	if mkp == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	rmk, err := s.room.MessageKey(s.ctx)
	if err != nil {
		return &response{err: err}
	}
	if rmk == nil {
		return &response{err: fmt.Errorf("room is public")}
	}

	if _, ok := s.client.Authorization.MessageKeys[rmk.KeyID()]; !ok {
		return &response{err: fmt.Errorf("not holding message key")}
	}

	switch {
	case cmd.AccountID != 0:
		account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
		if err != nil {
			return &response{err: err}
		}

		err = rmk.GrantToAccount(
			s.ctx, s.kms, s.client.Account, s.client.Authorization.ClientKey, account)
		if err != nil {
			return &response{err: err}
		}
	case cmd.Passcode != "":
		err = rmk.GrantToPasscode(s.ctx, s.client.Account, s.client.Authorization.ClientKey, cmd.Passcode)
		if err != nil {
			return &response{err: err}
		}
	}

	return &response{packet: &proto.GrantAccessReply{}}
}

func (s *session) handleRevokeAccessCommand(cmd *proto.RevokeAccessCommand) *response {
	mkp := s.client.Authorization.ManagerKeyPair
	if s.client.Account == nil || mkp == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	mkey, err := s.room.MessageKey(s.ctx)
	if err != nil {
		return &response{err: err}
	}

	switch {
	case cmd.AccountID != 0:
		account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
		if err != nil {
			return &response{err: err}
		}
		if err := mkey.RevokeFromAccount(s.ctx, account); err != nil {
			return &response{err: err}
		}
	case cmd.Passcode != "":
		if err := mkey.RevokeFromPasscode(s.ctx, cmd.Passcode); err != nil {
			return &response{err: err}
		}
	}

	return &response{packet: &proto.RevokeAccessReply{}}
}

func (s *session) handleGrantManagerCommand(cmd *proto.GrantManagerCommand) *response {
	mkp := s.client.Authorization.ManagerKeyPair
	if s.managedRoom == nil || s.client.Account == nil || mkp == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
	if err != nil {
		return &response{err: err}
	}

	err = s.managedRoom.AddManager(s.ctx, s.kms, s.client.Account, s.client.Authorization.ClientKey, account)
	if err != nil {
		return &response{err: err}
	}

	return &response{packet: &proto.GrantAccessReply{}}
}

func (s *session) handleRevokeManagerCommand(cmd *proto.RevokeManagerCommand) *response {
	if s.managedRoom == nil || s.client.Account == nil || s.client.Authorization.ManagerKeyPair == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
	if err != nil {
		return &response{err: err}
	}

	err = s.managedRoom.RemoveManager(s.ctx, s.client.Account, s.client.Authorization.ClientKey, account)
	if err != nil {
		return &response{err: err}
	}

	return &response{packet: &proto.RevokeManagerReply{}}
}

func (s *session) handleStaffGrantManagerCommand(cmd *proto.StaffGrantManagerCommand) *response {
	if s.staffKMS == nil {
		return &response{err: fmt.Errorf("must unlock staff capability first")}
	}

	if s.managedRoom == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
	if err != nil {
		return &response{err: err}
	}

	mkey, err := s.managedRoom.ManagerKey(s.ctx)
	if err != nil {
		return &response{err: err}
	}

	msgkey, err := s.room.MessageKey(s.ctx)
	if err != nil {
		return &response{err: err}
	}

	if err := mkey.StaffGrantToAccount(s.ctx, s.staffKMS, account); err != nil {
		return &response{err: err}
	}

	if msgkey != nil {
		if err := msgkey.StaffGrantToAccount(s.ctx, s.staffKMS, account); err != nil {
			return &response{err: err}
		}
	}

	return &response{packet: &proto.StaffGrantManagerReply{}}
}

func (s *session) handleStaffRevokeManagerCommand(cmd *proto.StaffRevokeManagerCommand) *response {
	if s.staffKMS == nil {
		return &response{err: fmt.Errorf("must unlock staff capability first")}
	}

	if s.managedRoom == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
	if err != nil {
		return &response{err: err}
	}

	mkey, err := s.managedRoom.ManagerKey(s.ctx)
	if err != nil {
		return &response{err: err}
	}

	if err := mkey.RevokeFromAccount(s.ctx, account); err != nil {
		return &response{err: err}
	}

	return &response{packet: &proto.StaffRevokeManagerReply{}}
}

func (s *session) handleStaffRevokeAccessCommand(cmd *proto.StaffRevokeAccessCommand) *response {
	if s.staffKMS == nil {
		return &response{err: fmt.Errorf("must unlock staff capability first")}
	}

	if s.managedRoom == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	mkey, err := s.managedRoom.MessageKey(s.ctx)
	if err != nil {
		return &response{err: err}
	}

	switch {
	case cmd.AccountID != 0:
		account, err := s.backend.AccountManager().Get(s.ctx, cmd.AccountID)
		if err != nil {
			return &response{err: err}
		}
		if err := mkey.RevokeFromAccount(s.ctx, account); err != nil {
			return &response{err: err}
		}
	case cmd.Passcode != "":
		if err := mkey.RevokeFromPasscode(s.ctx, cmd.Passcode); err != nil {
			return &response{err: err}
		}
	}

	return &response{packet: &proto.RevokeAccessReply{}}
}

func (s *session) handleStaffLockRoomCommand() *response {
	if s.staffKMS == nil {
		return &response{err: fmt.Errorf("must unlock staff capability first")}
	}

	if s.managedRoom == nil {
		return &response{err: proto.ErrAccessDenied}
	}

	if _, err := s.managedRoom.GenerateMessageKey(s.ctx, s.staffKMS); err != nil {
		return &response{err: err}
	}

	return &response{packet: &proto.StaffLockRoomReply{}}
}

func (s *session) handleLoginCommand(cmd *proto.LoginCommand) *response {
	account, err := s.backend.AccountManager().Resolve(s.ctx, cmd.Namespace, cmd.ID)
	if err != nil {
		switch err {
		case proto.ErrAccountNotFound:
			return &response{packet: &proto.LoginReply{Reason: err.Error()}}
		default:
			return &response{err: err}
		}
	}

	clientKey := account.KeyFromPassword(cmd.Password)

	if _, err = account.Unlock(clientKey); err != nil {
		switch err {
		case proto.ErrAccessDenied:
			return &response{packet: &proto.LoginReply{Reason: err.Error()}}
		default:
			return &response{err: err}
		}
	}

	err = s.backend.AgentTracker().SetClientKey(
		s.ctx, s.client.Agent.IDString(), s.agentKey, account.ID(), clientKey)
	if err != nil {
		return &response{err: err}
	}

	err = s.backend.NotifyUser(s.ctx, s.Identity().ID(), proto.LoginEventType, proto.LoginEvent{AccountID: account.ID()}, s)
	if err != nil {
		return &response{err: err}
	}

	reply := &proto.LoginReply{
		Success:   true,
		AccountID: account.ID(),
	}
	return &response{packet: reply}
}

func (s *session) handleLogoutCommand() *response {
	if err := s.backend.AgentTracker().ClearClientKey(s.ctx, s.client.Agent.IDString()); err != nil {
		return &response{err: err}
	}
	err := s.backend.NotifyUser(s.ctx, proto.UserID("agent:"+s.AgentID()), proto.LogoutEventType, proto.LogoutEvent{}, s)
	if err != nil {
		return &response{err: err}
	}
	return &response{packet: &proto.LogoutReply{}}
}

func (s *session) handleChangeEmailCommand(msg *proto.ChangeEmailCommand) *response {
	if s.client.Account == nil {
		return &response{err: proto.ErrNotLoggedIn}
	}
	if _, err := s.client.Account.Unlock(s.client.Account.KeyFromPassword(msg.Password)); err != nil {
		if err == proto.ErrAccessDenied {
			return &response{packet: &proto.ChangeEmailReply{Reason: err.Error()}}
		}
		return &response{err: err}
	}
	verified, err := s.backend.AccountManager().ChangeEmail(s.ctx, s.client.Account.ID(), msg.Email)
	if err != nil {
		return &response{err: err}
	}
	err = s.heim.OnAccountEmailChanged(
		s.ctx, s.backend, s.client.Account, s.client.Authorization.ClientKey, msg.Email, verified)
	if err != nil {
		return &response{err: err}
	}
	return &response{packet: &proto.ChangeEmailReply{Success: true, VerificationNeeded: !verified}}
}

func (s *session) handleResendVerificationEmail(msg *proto.ResendVerificationEmailCommand) *response {
	if s.client.Account == nil {
		return &response{err: proto.ErrNotLoggedIn}
	}

	// Refresh view of account.
	account, err := s.backend.AccountManager().Get(s.ctx, s.client.Account.ID())
	if err != nil {
		return &response{err: err}
	}
	sent := false
	for _, pid := range account.PersonalIdentities() {
		fmt.Printf("considering pid %s/%s/%t\n", pid.Namespace(), pid.ID(), pid.Verified())
		if pid.Namespace() == "email" && !pid.Verified() {
			err := s.heim.OnAccountEmailChanged(
				s.ctx, s.backend, account, s.client.Authorization.ClientKey, pid.ID(), false)
			if err != nil {
				return &response{err: err}
			}
			sent = true
		}
	}
	if !sent {
		return &response{err: proto.ErrPersonalIdentityAlreadyVerified}
	}
	return &response{packet: &proto.ResendVerificationEmailReply{}}
}

func (s *session) handleChangeNameCommand(msg *proto.ChangeNameCommand) *response {
	if s.client.Account == nil {
		return &response{err: proto.ErrNotLoggedIn}
	}
	if err := s.backend.AccountManager().ChangeName(s.ctx, s.client.Account.ID(), msg.Name); err != nil {
		return &response{err: err}
	}
	return &response{packet: &proto.ChangeNameReply{Name: msg.Name}}
}

func (s *session) handleChangePasswordCommand(msg *proto.ChangePasswordCommand) *response {
	if s.client.Account == nil {
		return &response{err: proto.ErrNotLoggedIn}
	}

	oldClientKey := s.client.Account.KeyFromPassword(msg.OldPassword)
	newClientKey := s.client.Account.KeyFromPassword(msg.NewPassword)

	// Change password, invalidating all agents.
	err := s.backend.AccountManager().ChangeClientKey(
		s.ctx, s.client.Account.ID(), oldClientKey, newClientKey)
	if err != nil {
		return &response{err: err}
	}

	// Log in current agent using new password.
	err = s.backend.AgentTracker().SetClientKey(
		s.ctx, s.client.Agent.IDString(), s.agentKey, s.client.Account.ID(), newClientKey)
	if err != nil {
		return &response{err: err}
	}

	// Log out all other agents on this account.
	err = s.backend.NotifyUser(s.ctx, s.Identity().ID(), proto.LogoutEventType, proto.LogoutEvent{}, s)
	if err != nil {
		return &response{err: err}
	}

	if err := s.heim.OnAccountPasswordChanged(s.ctx, s.backend, s.client.Account); err != nil {
		return &response{err: err}
	}

	return &response{packet: &proto.ChangePasswordReply{}}
}

func (s *session) handleResetPasswordCommand(msg *proto.ResetPasswordCommand) *response {
	acc, req, err := s.backend.AccountManager().RequestPasswordReset(s.ctx, s.kms, msg.Namespace, msg.ID)
	if err != nil {
		return &response{err: err}
	}

	if err := s.heim.OnAccountPasswordResetRequest(s.ctx, s.backend, acc, req); err != nil {
		return &response{err: err}
	}

	return &response{packet: &proto.ResetPasswordReply{}}
}

func (s *session) handleRegisterAccountCommand(cmd *proto.RegisterAccountCommand) *response {
	// Session must not be logged in.
	if s.client.Account != nil {
		return &response{packet: &proto.RegisterAccountReply{Reason: "already logged in"}}
	}

	// Agent must be of sufficient age.
	if time.Now().Sub(s.client.Agent.Created) < s.server.newAccountMinAgentAge {
		return &response{packet: &proto.RegisterAccountReply{Reason: "not familiar yet, try again later"}}
	}

	// Validate givens.
	if ok, reason := proto.ValidatePersonalIdentity(cmd.Namespace, cmd.ID); !ok {
		return &response{packet: &proto.RegisterAccountReply{Reason: reason}}
	}

	if ok, reason := proto.ValidateAccountPassword(cmd.Password); !ok {
		return &response{packet: &proto.RegisterAccountReply{Reason: reason}}
	}

	// Register the account.
	account, clientKey, err := s.backend.AccountManager().Register(
		s.ctx, s.kms, cmd.Namespace, cmd.ID, cmd.Password, s.client.Agent.IDString(), s.agentKey)
	if err != nil {
		switch err {
		case proto.ErrPersonalIdentityInUse:
			return &response{packet: &proto.RegisterAccountReply{Reason: err.Error()}}
		default:
			return &response{err: err}
		}
	}

	// Kick off on-registration tasks.
	if err := s.heim.OnAccountRegistration(s.ctx, s.backend, account, clientKey); err != nil {
		// Log this error only.
		logging.Logger(s.ctx).Printf("error on account registration: %s", err)
	}

	// Authorize session's agent to unlock account.
	err = s.backend.AgentTracker().SetClientKey(
		s.ctx, s.client.Agent.IDString(), s.agentKey, account.ID(), clientKey)
	if err != nil {
		return &response{err: err}
	}

	// Return successful response.
	reply := &proto.RegisterAccountReply{
		Success:   true,
		AccountID: account.ID(),
	}
	return &response{packet: reply}
}

func (s *session) handleAuthCommand(msg *proto.AuthCommand) *response {
	if s.joined {
		return &response{packet: &proto.AuthReply{Success: true}}
	}

	if s.authFailCount > 0 {
		buf := []byte{0}
		if _, err := rand.Read(buf); err != nil {
			return &response{err: err}
		}
		jitter := 4 * time.Duration(int(buf[0])-128) * time.Millisecond
		delay := authDelay + jitter
		if security.TestMode {
			delay = 0
		}
		time.Sleep(delay)
	}

	authAttempts.WithLabelValues(s.roomName).Inc()

	var (
		failureReason string
		err           error
	)
	switch msg.Type {
	case proto.AuthPasscode:
		failureReason, err = s.client.AuthenticateWithPasscode(s.ctx, s.room, msg.Passcode)
	default:
		failureReason = fmt.Sprintf("auth type not supported: %s", msg.Type)
	}
	if err != nil {
		return &response{err: err}
	}
	if failureReason != "" {
		authFailures.WithLabelValues(s.roomName).Inc()
		s.authFailCount++
		if s.authFailCount >= MaxAuthFailures {
			logging.Logger(s.ctx).Printf(
				"max authentication failures on room %s by %s", s.roomName, s.Identity().ID())
			authTerminations.WithLabelValues(s.roomName).Inc()
			s.state = s.ignoreState
		}
		return &response{packet: &proto.AuthReply{Reason: failureReason}}
	}

	s.keyID = s.client.Authorization.CurrentMessageKeyID
	s.state = s.joinedState
	if err := s.join(); err != nil {
		s.keyID = ""
		s.state = s.unauthedState
		return &response{err: err}
	}
	return &response{packet: &proto.AuthReply{Success: true}}
}

func (s *session) handleStaffEnrollOTPCommand(cmd *proto.StaffEnrollOTPCommand) *response {
	failure := func(err error) *response { return &response{err: err} }

	if s.client.Account == nil || !s.client.Account.IsStaff() {
		return failure(proto.ErrAccessDenied)
	}

	// TODO: use staff's kms
	otp, err := s.backend.AccountManager().GenerateOTP(s.ctx, s.heim, s.kms, s.client.Account)
	if err != nil {
		return failure(err)
	}

	img, err := otp.QRImage(200, 200)
	if err != nil {
		return failure(err)
	}
	encodedImg := &bytes.Buffer{}
	if err := png.Encode(encodedImg, img); err != nil {
		return failure(err)
	}

	reply := &proto.StaffEnrollOTPReply{
		URI:     otp.URI,
		QRImage: fmt.Sprintf("data:image/png;base64,%s", base64.StdEncoding.EncodeToString(encodedImg.Bytes())),
	}
	return &response{packet: reply}
}

func (s *session) handleStaffValidateOTPCommand(cmd *proto.StaffValidateOTPCommand) *response {
	failure := func(err error) *response { return &response{err: err} }

	if s.client.Account == nil || !s.client.Account.IsStaff() {
		return failure(proto.ErrAccessDenied)
	}

	// TODO: use staff's kms
	if err := s.backend.AccountManager().ValidateOTP(s.ctx, s.kms, s.client.Account.ID(), cmd.Password); err != nil {
		return failure(err)
	}

	return &response{packet: &proto.StaffValidateOTPReply{}}
}

func (s *session) handleStaffInspectIPCommand(cmd *proto.StaffInspectIPCommand) *response {
	if s.privilegeLevel() != proto.Staff {
		return &response{err: proto.ErrAccessDenied}
	}

	if s.heim.GeoIP == nil {
		return &response{err: fmt.Errorf("geoip support not configured")}
	}

	addr, err := s.room.ResolveClientAddress(s.ctx, cmd.IP)
	if err != nil {
		return &response{err: err}
	}

	// geoip uses google's context package
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	resp, err := s.heim.GeoIP.Insights(ctx, addr.String())
	if err != nil {
		return &response{err: err}
	}

	serialized, err := json.Marshal(resp)
	if err != nil {
		return &response{err: err}
	}

	reply := &proto.StaffInspectIPReply{
		IP:      addr.String(),
		Details: json.RawMessage(serialized),
	}
	return &response{packet: reply}
}

func (s *session) handleStaffInvadeCommand(cmd *proto.StaffInvadeCommand) *response {
	failure := func(err error) *response { return &response{err: err} }

	if s.managedRoom == nil || s.client.Account == nil || !s.client.Account.IsStaff() {
		return failure(proto.ErrAccessDenied)
	}

	// TODO: use staff's kms
	if err := s.backend.AccountManager().ValidateOTP(s.ctx, s.kms, s.client.Account.ID(), cmd.Password); err != nil {
		return failure(err)
	}

	// Everything checks out. Acquire the host key.
	managerKey, err := s.managedRoom.ManagerKey(s.ctx)
	if err != nil {
		return failure(err)
	}
	managerKeyPair, err := managerKey.StaffUnlock(s.kms)
	if err != nil {
		return failure(err)
	}
	s.client.Authorization.ManagerKeyPair = managerKeyPair

	// Now acquire the message key and join the room, if necessary.
	mkey, err := s.managedRoom.MessageKey(s.ctx)
	if err != nil {
		return failure(err)
	}
	if mkey != nil && !s.joined {
		k := mkey.ManagedKey()
		if err := s.kms.DecryptKey(&k); err != nil {
			return failure(err)
		}
		s.client.Authorization.AddMessageKey(mkey.KeyID(), &k)
		s.keyID = s.client.Authorization.CurrentMessageKeyID
		s.state = s.joinedState
		if err := s.join(); err != nil {
			s.keyID = ""
			s.state = s.unauthedState
			return &response{err: err}
		}
	}

	return &response{packet: &proto.StaffInvadeReply{}}
}

func (s *session) handleUnlockStaffCapabilityCommand(cmd *proto.UnlockStaffCapabilityCommand) *response {
	rejection := func(reason string) *response {
		return &response{packet: &proto.UnlockStaffCapabilityReply{FailureReason: reason}}
	}

	failure := func(err error) *response { return &response{err: err} }

	if s.client.Account == nil || !s.client.Account.IsStaff() {
		return rejection("access denied")
	}

	kms, err := s.client.Account.UnlockStaffKMS(s.client.Account.KeyFromPassword(cmd.Password))
	if err != nil {
		// TODO: return specific failure reason for incorrect password
		return failure(err)
	}

	s.staffKMS = kms
	return &response{packet: &proto.UnlockStaffCapabilityReply{Success: true}}
}

func (s *session) handleStaffCreateRoomCommand(cmd *proto.StaffCreateRoomCommand) *response {
	rejection := func(reason string) *response {
		return &response{packet: &proto.StaffCreateRoomReply{FailureReason: reason}}
	}

	failure := func(err error) *response { return &response{err: err} }

	if s.client.Account == nil || !s.client.Account.IsStaff() {
		return rejection("access denied")
	}

	if s.staffKMS == nil {
		return rejection("must unlock staff capability first")
	}

	if len(cmd.Managers) == 0 {
		return rejection("at least one manager is required")
	}

	managers := make([]proto.Account, len(cmd.Managers))
	for i, accountID := range cmd.Managers {
		account, err := s.backend.AccountManager().Get(s.ctx, accountID)
		if err != nil {
			switch err {
			case proto.ErrAccountNotFound:
				return rejection(err.Error())
			default:
				return failure(err)
			}
		}
		managers[i] = account
	}

	// TODO: validate room name
	// TODO: support unnamed rooms

	_, err := s.backend.CreateRoom(s.ctx, s.staffKMS, cmd.Private, cmd.Name, managers...)
	if err != nil {
		return failure(err)
	}

	return &response{packet: &proto.StaffCreateRoomReply{Success: true}}
}

func (s *session) handleEditMessageCommand(msg *proto.EditMessageCommand) *response {
	if s.client.Account == nil || s.client.Authorization.ManagerKeyPair == nil {
		return &response{err: proto.ErrAccessDenied}
	}
	reply, err := s.room.EditMessage(s.ctx, s, *msg)
	if err != nil {
		return &response{err: err}
	}
	return &response{packet: reply}
}

func (s *session) handleBanCommand(msg *proto.BanCommand) *response {
	// Copy input into reply before processing, so we don't leak addresses.
	reply := &proto.BanReply{
		Ban:     msg.Ban,
		Seconds: msg.Seconds,
	}
	if s.managedRoom == nil || s.privilegeLevel() == proto.General {
		return &response{err: proto.ErrAccessDenied}
	}
	if msg.Ban.Global && s.privilegeLevel() != proto.Staff {
		return &response{err: proto.ErrAccessDenied}
	}
	var until time.Time
	if msg.Seconds != 0 {
		until = time.Now().Add(time.Duration(msg.Seconds) * time.Second)
	}
	if msg.Ban.IP != "" {
		addr, err := s.room.ResolveClientAddress(s.ctx, msg.Ban.IP)
		if err != nil {
			return &response{err: err}
		}
		msg.Ban.IP = addr.String()
	}
	if msg.Ban.Global {
		if err := s.backend.Ban(s.ctx, msg.Ban, until); err != nil {
			return &response{err: err}
		}
	} else {
		if err := s.managedRoom.Ban(s.ctx, msg.Ban, until); err != nil {
			return &response{err: err}
		}
	}
	return &response{packet: reply}
}

func (s *session) handleUnbanCommand(msg *proto.UnbanCommand) *response {
	// Copy input into reply before processing, so we don't leak addresses.
	reply := &proto.UnbanReply{
		Ban: msg.Ban,
	}
	if s.managedRoom == nil || s.privilegeLevel() == proto.General {
		return &response{err: proto.ErrAccessDenied}
	}
	if s.privilegeLevel() != proto.Staff && msg.Global {
		return &response{err: proto.ErrAccessDenied}
	}
	if msg.Ban.IP != "" {
		addr, err := s.room.ResolveClientAddress(s.ctx, msg.Ban.IP)
		if err != nil {
			return &response{err: err}
		}
		msg.Ban.IP = addr.String()
	}
	switch msg.Global {
	case false:
		if err := s.managedRoom.Unban(s.ctx, msg.Ban); err != nil {
			return &response{err: err}
		}
	case true:
		if err := s.backend.Unban(s.ctx, msg.Ban); err != nil {
			return &response{err: err}
		}
	}
	return &response{packet: reply}
}
