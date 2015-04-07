package proto

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"euphoria.io/scope"
)

type clientKey int

type Client struct {
	IP        string
	UserAgent string
	AgentID   string
	Connected time.Time
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
