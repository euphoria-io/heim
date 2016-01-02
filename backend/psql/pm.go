package psql

import (
	"database/sql"
	"fmt"

	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"
)

type PM struct {
	ID                    string
	Initiator             string
	InitiatorNick         string `db:"initiator_nick"`
	Receiver              string
	ReceiverNick          string `db:"receiver_nick"`
	ReceiverMAC           []byte `db:"receiver_mac"`
	IV                    []byte
	EncryptedSystemKey    []byte `db:"encrypted_system_key"`
	EncryptedInitiatorKey []byte `db:"encrypted_initiator_key"`
	EncryptedReceiverKey  []byte `db:"encrypted_receiver_key"`
}

func (pm *PM) ToBackend() *proto.PM {
	bpm := &proto.PM{
		InitiatorNick: pm.InitiatorNick,
		Receiver:      proto.UserID(pm.Receiver),
		ReceiverNick:  pm.ReceiverNick,
		ReceiverMAC:   pm.ReceiverMAC,
		IV:            pm.IV,
		EncryptedSystemKey: &security.ManagedKey{
			KeyType:      proto.RoomMessageKeyType,
			Ciphertext:   pm.EncryptedSystemKey,
			ContextKey:   "pm",
			ContextValue: pm.ID,
		},
		EncryptedInitiatorKey: &security.ManagedKey{
			KeyType:    proto.RoomMessageKeyType,
			IV:         pm.IV,
			Ciphertext: pm.EncryptedInitiatorKey,
		},
	}
	if len(pm.EncryptedReceiverKey) > 0 {
		bpm.EncryptedReceiverKey = &security.ManagedKey{
			KeyType:    proto.RoomMessageKeyType,
			IV:         pm.IV,
			Ciphertext: pm.EncryptedReceiverKey,
		}
	}
	// ignore id parsing errors
	_ = bpm.ID.FromString(pm.ID)
	_ = bpm.Initiator.FromString(pm.Initiator)
	return bpm
}

type PMRoomBinding struct {
	RoomBinding
	pm *proto.PM
}

func (pmrb *PMRoomBinding) MessageKeyID(ctx scope.Context) (string, bool, error) {
	return fmt.Sprintf("pm:%s", pmrb.pm.ID), true, nil
}

func (pmrb *PMRoomBinding) ResolveNick(ctx scope.Context, userID proto.UserID) (string, bool, error) {
	if userID == proto.UserID(fmt.Sprintf("account:%s", pmrb.pm.Initiator)) {
		return pmrb.pm.InitiatorNick, true, nil
	}
	if userID == pmrb.pm.Receiver {
		return pmrb.pm.ReceiverNick, true, nil
	}
	return "", false, nil
}

func (pmrb *PMRoomBinding) Snapshot(
	ctx scope.Context, session proto.Session, level proto.PrivilegeLevel, numMessages int) (*proto.SnapshotEvent, error) {

	snapshot, err := pmrb.RoomBinding.Snapshot(ctx, session, level, numMessages)
	if err != nil {
		return nil, err
	}

	if snapshot.Nick == pmrb.pm.InitiatorNick {
		snapshot.PMWithNick = pmrb.pm.ReceiverNick
		snapshot.PMWithUserID = pmrb.pm.Receiver
	} else {
		snapshot.PMWithNick = pmrb.pm.InitiatorNick
		snapshot.PMWithUserID = proto.UserID(fmt.Sprintf("account:%s", pmrb.pm.Initiator))
	}
	return snapshot, nil
}

type PMTracker struct {
	*Backend
}

func (t *PMTracker) Initiate(
	ctx scope.Context, kms security.KMS, room proto.Room, client *proto.Client, recipient proto.UserID) (
	snowflake.Snowflake, error) {

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

	pm, err := proto.InitiatePM(ctx, t.Backend, kms, client, initiatorNick, recipient, recipientNick)
	if err != nil {
		return 0, err
	}
	row := &PM{
		ID:                    pm.ID.String(),
		Initiator:             pm.Initiator.String(),
		InitiatorNick:         pm.InitiatorNick,
		Receiver:              string(pm.Receiver),
		ReceiverNick:          pm.ReceiverNick,
		ReceiverMAC:           pm.ReceiverMAC,
		IV:                    pm.IV,
		EncryptedSystemKey:    pm.EncryptedSystemKey.Ciphertext,
		EncryptedInitiatorKey: pm.EncryptedInitiatorKey.Ciphertext,
	}
	if pm.EncryptedReceiverKey != nil {
		row.EncryptedReceiverKey = pm.EncryptedReceiverKey.Ciphertext
	}

	// Look for existing PM to reuse.
	tx, err := t.DbMap.Begin()
	if err != nil {
		return 0, err
	}

	var existingRow PM
	err = tx.SelectOne(
		&existingRow,
		"SELECT id FROM pm WHERE initiator = $1 AND receiver = $2",
		client.Account.ID().String(), string(recipient))
	if err != nil && err != sql.ErrNoRows {
		rollback(ctx, tx)
		return 0, err
	}
	if err == nil {
		rollback(ctx, tx)
		var pmID snowflake.Snowflake
		if err := pmID.FromString(existingRow.ID); err != nil {
			return 0, err
		}
		return pmID, nil
	}

	kind, id := recipient.Parse()
	if kind == "account" {
		var existingRow PM
		err = tx.SelectOne(
			&existingRow,
			"SELECT id FROM pm WHERE initiator = $1 AND receiver = $2",
			id, string(client.UserID()))
		if err != nil && err != sql.ErrNoRows {
			rollback(ctx, tx)
			return 0, err
		}
		if err == nil {
			rollback(ctx, tx)
			var pmID snowflake.Snowflake
			if err := pmID.FromString(existingRow.ID); err != nil {
				return 0, err
			}
			return pmID, nil
		}
	}

	if err := tx.Insert(row); err != nil {
		rollback(ctx, tx)
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}

	return pm.ID, nil
}

func (t *PMTracker) Room(ctx scope.Context, kms security.KMS, pmID snowflake.Snowflake, client *proto.Client) (proto.Room, *security.ManagedKey, error) {
	row, err := t.Backend.Get(PM{}, pmID.String())
	if row == nil || err != nil {
		if row == nil || err == sql.ErrNoRows {
			return nil, nil, proto.ErrPMNotFound
		}
	}

	pm := row.(*PM).ToBackend()
	pmKey, modified, otherName, err := pm.Access(ctx, t.Backend, kms, client)
	if err != nil {
		return nil, nil, err
	}

	if modified {
		_, err := t.Backend.DbMap.Exec(
			"UPDATE pm SET receiver = $2, receiver_mac = $3, encrypted_receiver_key = $4 WHERE id = $1",
			pm.ID.String(), string(pm.Receiver), pm.ReceiverMAC, pm.EncryptedReceiverKey.Ciphertext)
		if err != nil {
			return nil, nil, err
		}
	}

	room := &PMRoomBinding{
		RoomBinding: RoomBinding{
			RoomName:  fmt.Sprintf("pm:%s", pm.ID),
			RoomTitle: fmt.Sprintf("%s (private chat)", otherName),
			Backend:   t.Backend,
		},
		pm: pm,
	}

	return room, pmKey, nil
}
