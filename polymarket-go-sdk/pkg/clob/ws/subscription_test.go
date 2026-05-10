package ws

import (
	"encoding/json"
	"testing"
)

func TestSubscriptionRequestJSON(t *testing.T) {
	req := NewMarketSubscription([]string{"1", "2"}).WithCustomFeatures(true)

	raw, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded map[string]interface{}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if decoded["type"] != string(ChannelMarket) {
		t.Fatalf("type mismatch: got %v", decoded["type"])
	}
	if decoded["operation"] != string(OperationSubscribe) {
		t.Fatalf("operation mismatch: got %v", decoded["operation"])
	}
	if decoded["initial_dump"] != true {
		t.Fatalf("initial_dump mismatch: got %v", decoded["initial_dump"])
	}
	if decoded["custom_feature_enabled"] != true {
		t.Fatalf("custom_feature_enabled mismatch: got %v", decoded["custom_feature_enabled"])
	}
}
