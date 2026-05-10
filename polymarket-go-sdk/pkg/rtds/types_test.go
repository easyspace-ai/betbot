package rtds

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSubscriptionFiltersSerialization(t *testing.T) {
	sub := Subscription{
		Topic:   string(CryptoPrice),
		MsgType: "update",
		Filters: `["btcusdt","ethusdt"]`,
	}
	data, err := json.Marshal(sub)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"filters":["btcusdt","ethusdt"]`) {
		t.Fatalf("expected filters array, got %s", string(data))
	}

	chainlink := Subscription{
		Topic:   string(ChainlinkPrice),
		MsgType: "*",
		Filters: `{"symbol":"eth/usd"}`,
	}
	data, err = json.Marshal(chainlink)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"filters":"{\"symbol\":\"eth/usd\"}"`) {
		t.Fatalf("expected chainlink filters as string, got %s", string(data))
	}

	raw := Subscription{
		Topic:   string(CryptoPrice),
		MsgType: "update",
		Filters: "not-json",
	}
	data, err = json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(data), `"filters":"not-json"`) {
		t.Fatalf("expected raw filters string, got %s", string(data))
	}
}
