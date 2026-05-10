package transport

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// WebSocketDialer returns a WebSocket dialer that routes through proxyURL when non-empty.
// Empty proxyURL returns websocket.DefaultDialer.
func WebSocketDialer(proxyURL string) (*websocket.Dialer, error) {
	u, err := ParseHTTPProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return websocket.DefaultDialer, nil
	}
	d := *websocket.DefaultDialer
	d.Proxy = http.ProxyURL(u)
	return &d, nil
}
