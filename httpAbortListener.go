package caddy_httpAbortListener

import (
	"bufio"
	"fmt"
	"net"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

func init() {
	caddy.RegisterModule(HTTPAbortListenerWrapper{})
}

// HTTPAbortListenerWrapper abort the
// connections that come on the TLS port as an HTTP request
// by detecting using the first few bytes that it's not a TLS
// handshake, but instead an HTTP request.
//
// This is especially useful when using a non-standard HTTPS port.
// A user may simply type the address in their browser without the
// https:// scheme, which would cause the browser to attempt the
// connection over HTTP, but this would cause a "Client sent an
// HTTP request to an HTTPS server" error response.
//
// This listener wrapper must be placed BEFORE the "tls" listener
// wrapper, for it to work properly.

// This plugin is inspired by the original HTTPRedirectListenerWrapper.

type HTTPAbortListenerWrapper struct {
}

func (HTTPAbortListenerWrapper) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "caddy.listeners.http_abort",
		New: func() caddy.Module { return new(HTTPAbortListenerWrapper) },
	}
}

func (h *HTTPAbortListenerWrapper) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	return nil
}

func (h *HTTPAbortListenerWrapper) WrapListener(l net.Listener) net.Listener {
	return &httpAbortListener{l}
}

// httpAbortListener is listener that checks the first few bytes
// of the request when the server is intended to accept HTTPS requests,
// to abort the HTTP request
type httpAbortListener struct {
	net.Listener
}

// Accept waits for and returns the next connection to the listener,
// wrapping it with a httpAbortConn.
func (l *httpAbortListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}

	return &httpAbortConn{
		Conn: c,
		r:    bufio.NewReader(c),
	}, nil
}

type httpAbortConn struct {
	net.Conn
	once bool
	r    *bufio.Reader
}

// Read tries to peek at the first few bytes of the request, and if we get
// an error reading the headers, and that error was due to the bytes looking
// like an HTTP request, then we abort the connection.
func (c *httpAbortConn) Read(p []byte) (int, error) {
	if c.once {
		return c.r.Read(p)
	}
	// no need to use sync.Once - net.Conn is not read from concurrently.
	c.once = true

	firstBytes, err := c.r.Peek(5)
	if err != nil {
		return 0, err
	}

	// If the request doesn't look like HTTP, then it's probably
	// TLS bytes, and we don't need to do anything.
	if !firstBytesLookLikeHTTP(firstBytes) {
		return c.r.Read(p)
	}

	// From now on, we can be almost certain the request is HTTP.
	// The returned error will be non nil and caller are expected to
	// close the connection.

	return 0, fmt.Errorf("connection aborted: non-TLS connection")
}

// firstBytesLookLikeHTTP reports whether a TLS record header
// looks like it might've been a misdirected plaintext HTTP request.
func firstBytesLookLikeHTTP(hdr []byte) bool {
	switch string(hdr[:5]) {
	case "GET /", "HEAD ", "POST ", "PUT /", "OPTIO":
		return true
	}
	return false
}

var (
	_ caddy.ListenerWrapper = (*HTTPAbortListenerWrapper)(nil)
	_ net.Conn              = (*httpAbortConn)(nil)
)
