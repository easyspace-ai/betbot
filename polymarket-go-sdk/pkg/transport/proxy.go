package transport

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ProxyURLFromEnv returns POLYMARKET_PROXY_URL when set, for use with DefaultConfig.
func ProxyURLFromEnv() string {
	return strings.TrimSpace(os.Getenv("POLYMARKET_PROXY_URL"))
}

// ParseHTTPProxyURL parses an HTTP or HTTPS proxy URL. Empty input returns (nil, nil).
func ParseHTTPProxyURL(raw string) (*url.URL, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return nil, nil
	}
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("parse proxy URL: %w", err)
	}
	switch u.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("proxy URL scheme must be http or https, got %q", u.Scheme)
	}
	return u, nil
}

// NewHTTPClientWithProxy returns an *http.Client that sends all requests through the
// given HTTP or HTTPS proxy (RFC 7231 CONNECT for TLS targets). proxyURL must be a
// full URL such as http://127.0.0.1:15236.
func NewHTTPClientWithProxy(proxyURL string, timeout time.Duration) (*http.Client, error) {
	u, err := ParseHTTPProxyURL(proxyURL)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, fmt.Errorf("proxy URL is empty")
	}

	defaultTr, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		tr := &http.Transport{Proxy: http.ProxyURL(u)}
		return &http.Client{Transport: tr, Timeout: timeout}, nil
	}
	tr := defaultTr.Clone()
	tr.Proxy = http.ProxyURL(u)
	return &http.Client{Transport: tr, Timeout: timeout}, nil
}
