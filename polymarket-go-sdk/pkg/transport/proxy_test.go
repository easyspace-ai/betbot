package transport

import (
	"testing"
)

func TestParseHTTPProxyURL(t *testing.T) {
	u, err := ParseHTTPProxyURL("")
	if err != nil || u != nil {
		t.Fatalf("empty: u=%v err=%v", u, err)
	}
	if _, err := ParseHTTPProxyURL("socks5://x"); err == nil {
		t.Fatal("expected scheme error")
	}
}

func TestNewHTTPClientWithProxy_Invalid(t *testing.T) {
	_, err := NewHTTPClientWithProxy("", 0)
	if err == nil {
		t.Fatal("expected error for empty proxy")
	}
	_, err = NewHTTPClientWithProxy("socks5://127.0.0.1:1080", 0)
	if err == nil {
		t.Fatal("expected error for unsupported scheme")
	}
}

func TestNewHTTPClientWithProxy_OK(t *testing.T) {
	c, err := NewHTTPClientWithProxy("http://127.0.0.1:15236", 30)
	if err != nil {
		t.Fatal(err)
	}
	if c == nil || c.Transport == nil {
		t.Fatal("expected client with transport")
	}
}
