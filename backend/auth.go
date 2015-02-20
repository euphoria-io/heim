package backend

import (
	"fmt"

	"heim/proto"
	"heim/proto/security"

	"golang.org/x/net/context"
)

func Authenticate(ctx context.Context, room proto.Room, cmd *proto.AuthCommand) (
	*proto.AuthReply, *security.ManagedKey, security.Capability, error) {

	switch cmd.Type {
	case proto.AuthPasscode:
		return authenticateWithPasscode(ctx, room, cmd.Passcode)
	default:
		reply := &proto.AuthReply{
			Reason: fmt.Sprintf("auth type not supported: %s", cmd.Type),
		}
		return reply, nil, nil, nil
	}
}

func authenticateWithPasscode(ctx context.Context, room proto.Room, passcode string) (
	*proto.AuthReply, *security.ManagedKey, security.Capability, error) {

	mkey, err := room.MasterKey(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	if mkey == nil {
		return &proto.AuthReply{Success: true}, nil, nil, nil
	}

	capabilityID, err := security.GetCapabilityIDForPasscode(mkey.Nonce(), []byte(passcode))
	if err != nil {
		return nil, nil, nil, err
	}

	capability, err := room.GetCapability(ctx, capabilityID)
	if err != nil {
		return nil, nil, nil, err
	}

	if capability == nil {
		return &proto.AuthReply{Reason: "passcode incorrect"}, nil, nil, nil
	}

	clientKey := security.KeyFromPasscode([]byte(passcode), mkey.Nonce(), security.AES128.KeySize())
	roomKey, err := decryptRoomKey(clientKey, capability)
	if err != nil {
		return nil, nil, nil, err
	}

	return &proto.AuthReply{Success: true}, roomKey, capability, nil
}
