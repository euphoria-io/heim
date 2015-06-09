package proto

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"euphoria.io/heim/proto/security"
	"euphoria.io/scope"
)

type clientKey int

type Client struct {
	IP            string
	UserAgent     string
	Connected     time.Time
	Agent         *Agent
	Account       Account
	Authorization Authorization
}

func (c *Client) FromRequest(ctx scope.Context, r *http.Request) {
	c.UserAgent = r.Header.Get("User-Agent")
	c.Connected = time.Now()
	c.IP = getIP(r)

	var k clientKey
	ctx.Set(k, c)
}

func (c *Client) FromContext(ctx scope.Context) bool {
	var k clientKey
	src, ok := ctx.Get(k).(*Client)
	if !ok || src == nil {
		return false
	}
	*c = *src
	return true
}

func (c *Client) UserID() string {
	if c.Account != nil {
		fmt.Printf("c.UserID(): returning account:%s\n", c.Account.ID().String())
		return fmt.Sprintf("account:%s", c.Account.ID().String())
	}
	fmt.Printf("c.UserID(): returning agent:%s\n", c.Agent.IDString())
	return fmt.Sprintf("agent:%s", c.Agent.IDString())
}

func (c *Client) AuthenticateWithPasscode(ctx scope.Context, room Room, passcode string) (string, error) {
	mkey, err := room.MessageKey(ctx)
	if err != nil {
		return "", err
	}

	if mkey == nil {
		return "", nil
	}

	holderKey := security.KeyFromPasscode([]byte(passcode), mkey.Nonce(), security.AES128)

	capabilityID, err := security.SharedSecretCapabilityID(holderKey, mkey.Nonce())
	if err != nil {
		return "", err
	}

	capability, err := room.GetCapability(ctx, capabilityID)
	if err != nil {
		return "", err
	}

	if capability == nil {
		return "passcode incorrect", nil
	}

	roomKey, err := decryptRoomKey(holderKey, capability)
	if err != nil {
		return "", err
	}

	// TODO: convert to account grant if signed in
	// TODO: load and return all historic keys

	c.Authorization.AddMessageKey(mkey.KeyID(), roomKey)
	return "", nil
}

func getIP(r *http.Request) string {
	addr := r.RemoteAddr
	if ffs := r.Header["X-Forwarded-For"]; len(ffs) > 0 {
		fmt.Printf("X-Forwarded-For: %#v\n", ffs)
		parts := strings.Split(ffs[len(ffs)-1], ",")
		addr = strings.TrimSpace(parts[len(parts)-1])
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		fmt.Printf("err: %s\n", err)
		return addr
	}
	return host
}
