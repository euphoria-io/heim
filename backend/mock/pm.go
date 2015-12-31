package mock

import (
	"fmt"
	"sync"
	"time"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type PMTracker struct {
	m   sync.Mutex
	b   *TestBackend
	pms map[snowflake.Snowflake]*PM
}

func (t *PMTracker) Initiate(
	ctx scope.Context, kms security.KMS, room proto.Room, client *proto.Client, recipient proto.UserID) (
	snowflake.Snowflake, error) {

	t.m.Lock()
	defer t.m.Unlock()

	// Look for reusable PM.
	for pmID, pm := range t.pms {
		if pm.pm.Initiator == client.Account.ID() && pm.pm.Receiver == recipient {
			return pmID, nil
		}
		if pm.pm.Receiver == client.UserID() {
			kind, id := pm.pm.Receiver.Parse()
			if kind == "account" && id == pm.pm.Initiator.String() {
				return pmID, nil
			}
		}
	}

	// Create new PM.
	initiatorNick, ok, err := room.ResolveNick(ctx, proto.UserID(fmt.Sprintf("account:%s", client.Account.ID())))
	if err != nil {
		return 0, err
	}
	if !ok {
		initiatorNick = fmt.Sprintf("account:%s", client.Account.ID())
	}

	recipientNick, ok, err := room.ResolveNick(ctx, recipient)
	if err != nil {
		return 0, err
	}
	if !ok {
		recipientNick = string(recipient)
	}

	pm, err := proto.InitiatePM(ctx, t.b, kms, client, initiatorNick, recipient, recipientNick)
	if err != nil {
		return 0, err
	}

	pmKey, _, otherName, err := pm.Access(ctx, t.b, kms, client)
	if err != nil {
		return 0, err
	}

	if t.pms == nil {
		t.pms = map[snowflake.Snowflake]*PM{}
	}
	t.pms[pm.ID] = &PM{
		RoomBase: RoomBase{
			name:    fmt.Sprintf("private chat with %s", otherName),
			version: t.b.version,
			log:     newMemLog(),
			messageKey: &roomMessageKey{
				id:        fmt.Sprintf("pm:%s", pm.ID),
				timestamp: time.Now(),
				key:       *pmKey,
			},
		},
		pm: pm,
	}
	return pm.ID, nil
}

func (t *PMTracker) Room(ctx scope.Context, kms security.KMS, pmID snowflake.Snowflake, client *proto.Client) (proto.Room, *security.ManagedKey, error) {
	pm, ok := t.pms[pmID]
	if !ok {
		return nil, nil, proto.ErrPMNotFound
	}

	pmKey, _, otherName, err := pm.pm.Access(ctx, t.b, kms, client)
	if err != nil {
		return nil, nil, err
	}

	pm.RoomBase.name = fmt.Sprintf("private chat with %s", otherName)
	return pm, pmKey, nil
}

type PM struct {
	RoomBase
	pm *proto.PM
}

func (pm *PM) ResolveNick(ctx scope.Context, userID proto.UserID) (string, bool, error) {
	if userID == proto.UserID(fmt.Sprintf("account:%s", pm.pm.Initiator)) {
		return pm.pm.InitiatorNick, true, nil
	}
	if userID == pm.pm.Receiver {
		return pm.pm.ReceiverNick, true, nil
	}
	return "", false, nil
}
