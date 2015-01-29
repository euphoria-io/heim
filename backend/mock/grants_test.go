package mock

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"heim/proto"
	"heim/proto/security"

	"golang.org/x/net/context"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGrants(t *testing.T) {
	Convey("Grant a capability on a room", t, func() {
		kms := security.LocalKMS()
		kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
		ctx := context.Background()
		room := newMemRoom("test", "testver")
		roomKey, err := room.GenerateMasterKey(ctx, kms)
		So(err, ShouldBeNil)

		// Sign in as alice and send an encrypted message with aliceSendTime
		// as the nonce.
		aliceSendTime := time.Now()
		nonce := []byte(proto.NewSnowflakeFromTime(aliceSendTime).String())

		aliceKey := &security.ManagedKey{
			KeyType:   security.AES256,
			Plaintext: make([]byte, security.AES256.KeySize()),
		}
		grant, err := GrantCapabilityOnRoom(ctx, kms, roomKey, aliceKey)
		So(err, ShouldBeNil)
		capability := security.Capability(grant)

		alice := newSession("Alice")
		So(room.Join(ctx, alice), ShouldBeNil)

		msg := proto.Message{
			ID:       proto.NewSnowflakeFromTime(aliceSendTime),
			UnixTime: aliceSendTime.Unix(),
			Content:  "hello",
		}

		iv, err := base64.URLEncoding.DecodeString(capability.ID())
		So(err, ShouldBeNil)
		payload := capability.EncryptedPayload()
		So(aliceKey.BlockCrypt(iv, aliceKey.Plaintext, payload, false), ShouldBeNil)
		key := &security.ManagedKey{
			KeyType: security.AES128,
		}
		So(json.Unmarshal(aliceKey.Unpad(payload), &key.Plaintext), ShouldBeNil)

		digest, ciphertext, err := security.EncryptGCM(
			key, nonce, []byte(msg.Content), []byte("Alice"))
		So(err, ShouldBeNil)

		digestStr := base64.URLEncoding.EncodeToString(digest)
		cipherStr := base64.URLEncoding.EncodeToString(ciphertext)
		msg.Content = digestStr + "/" + cipherStr
		_, err = room.Send(ctx, alice, msg)
		So(err, ShouldBeNil)

		// Now sign in as bob and decrypt the message.
		bobKey := &security.ManagedKey{
			KeyType:   security.AES256,
			Plaintext: make([]byte, security.AES256.KeySize()),
		}
		//bobKey.Plaintext[0] = 1
		grant, err = GrantCapabilityOnRoom(ctx, kms, roomKey, bobKey)
		So(err, ShouldBeNil)
		capability = security.Capability(grant)

		iv, err = base64.URLEncoding.DecodeString(capability.ID())
		So(err, ShouldBeNil)
		payload = capability.EncryptedPayload()
		So(bobKey.BlockCrypt(iv, bobKey.Plaintext, payload, false), ShouldBeNil)
		key = &security.ManagedKey{
			KeyType: security.AES128,
		}
		So(json.Unmarshal(bobKey.Unpad(payload), &key.Plaintext), ShouldBeNil)

		bob := newSession("Bob")
		So(room.Join(ctx, bob), ShouldBeNil)
		log, err := room.Latest(ctx, 1, 0)
		So(err, ShouldBeNil)
		So(len(log), ShouldEqual, 1)
		msg = log[0]

		parts := strings.Split(msg.Content, "/")
		So(len(parts), ShouldEqual, 2)
		digest, err = base64.URLEncoding.DecodeString(parts[0])
		So(err, ShouldBeNil)
		ciphertext, err = base64.URLEncoding.DecodeString(parts[1])
		So(err, ShouldBeNil)

		plaintext, err := security.DecryptGCM(key, nonce, digest, ciphertext, []byte("Alice"))
		So(err, ShouldBeNil)
		So(string(plaintext), ShouldEqual, "hello")
	})
}
