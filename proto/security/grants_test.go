// Test from external package to avoid import cycles.
package security_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"euphoria.io/heim/backend/mock"
	"euphoria.io/heim/proto"
	"euphoria.io/heim/proto/security"
	"euphoria.io/heim/proto/snowflake"
	"euphoria.io/scope"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGrants(t *testing.T) {
	Convey("Grant a capability on a room", t, func() {
		kms := security.LocalKMS()
		kms.SetMasterKey(make([]byte, security.AES256.KeySize()))
		ctx := scope.New()
		client := &proto.Client{Agent: &proto.Agent{}}
		client.FromRequest(ctx, &http.Request{})
		backend := &mock.TestBackend{}
		room, err := backend.CreateRoom(ctx, kms, true, "test")
		So(err, ShouldBeNil)

		rkey, err := room.MessageKey(ctx)
		So(err, ShouldBeNil)
		mkey := rkey.ManagedKey()
		So(kms.DecryptKey(&mkey), ShouldBeNil)

		// Sign in as alice and send an encrypted message with aliceSendTime
		// as the nonce.
		aliceSendTime := time.Now()
		msgNonce := []byte(snowflake.NewFromTime(aliceSendTime).String())

		aliceKey := &security.ManagedKey{
			KeyType:   security.AES256,
			Plaintext: make([]byte, security.AES256.KeySize()),
		}

		grant, err := security.GrantSharedSecretCapability(aliceKey, rkey.Nonce(), nil, mkey.Plaintext)
		So(err, ShouldBeNil)

		alice := mock.TestSession("Alice", "A1")
		So(room.Join(ctx, alice), ShouldBeNil)

		msg := proto.Message{
			ID:       snowflake.NewFromTime(aliceSendTime),
			UnixTime: proto.Time(aliceSendTime),
			Content:  "hello",
		}

		iv, err := base64.URLEncoding.DecodeString(grant.CapabilityID())
		So(err, ShouldBeNil)
		payload := grant.EncryptedPayload()
		So(aliceKey.BlockCrypt(iv, aliceKey.Plaintext, payload, false), ShouldBeNil)
		key := &security.ManagedKey{
			KeyType: security.AES128,
		}
		So(json.Unmarshal(aliceKey.Unpad(payload), &key.Plaintext), ShouldBeNil)

		digest, ciphertext, err := security.EncryptGCM(
			key, msgNonce, []byte(msg.Content), []byte("Alice"))
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
		grant, err = security.GrantSharedSecretCapability(bobKey, rkey.Nonce(), nil, mkey.Plaintext)
		So(err, ShouldBeNil)

		iv, err = base64.URLEncoding.DecodeString(grant.CapabilityID())
		So(err, ShouldBeNil)
		payload = grant.EncryptedPayload()
		So(bobKey.BlockCrypt(iv, bobKey.Plaintext, payload, false), ShouldBeNil)
		key = &security.ManagedKey{
			KeyType: security.AES128,
		}
		So(json.Unmarshal(bobKey.Unpad(payload), &key.Plaintext), ShouldBeNil)

		bob := mock.TestSession("Bob", "B1")
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

		plaintext, err := security.DecryptGCM(key, msgNonce, digest, ciphertext, []byte("Alice"))
		So(err, ShouldBeNil)
		So(string(plaintext), ShouldEqual, "hello")
	})
}
