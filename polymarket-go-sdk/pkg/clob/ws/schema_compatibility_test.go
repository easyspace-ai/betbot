package ws

import (
	"testing"
	"time"
)

func TestProcessEvent_SchemaCompat_TypeAliasPrice(t *testing.T) {
	c := newTestClient()
	ch := make(chan PriceChangeEvent, 5)
	c.priceSubs["p1"] = &subscriptionEntry[PriceChangeEvent]{
		id: "p1", ch: ch, errCh: make(chan error, 5),
	}

	raw := map[string]interface{}{
		"type":       "price",
		"market":     "m1",
		"extra_meta": map[string]interface{}{"v": 2},
		"price_changes": []interface{}{
			map[string]interface{}{"asset_id": "tok-compat", "price": "0.51", "unexpected": "x"},
		},
	}
	c.processEvent(raw)

	select {
	case ev := <-ch:
		if ev.AssetID != "tok-compat" {
			t.Fatalf("expected tok-compat, got %s", ev.AssetID)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for type-alias price event")
	}
}

func TestProcessEvent_SchemaCompat_OrderbookBuysSells(t *testing.T) {
	c := newTestClient()
	obCh := make(chan OrderbookEvent, 5)
	c.orderbookSubs["ob1"] = &subscriptionEntry[OrderbookEvent]{
		id: "ob1", ch: obCh, errCh: make(chan error, 5),
	}

	raw := map[string]interface{}{
		"type":     "orderbook",
		"asset_id": "tok-ob",
		"market":   "m-ob",
		"buys":     []interface{}{map[string]interface{}{"price": "0.45", "size": "12"}},
		"sells":    []interface{}{map[string]interface{}{"price": "0.55", "size": "11"}},
	}
	c.processEvent(raw)

	select {
	case ev := <-obCh:
		if ev.AssetID != "tok-ob" {
			t.Fatalf("expected tok-ob, got %s", ev.AssetID)
		}
		if len(ev.Bids) != 1 || len(ev.Asks) != 1 {
			t.Fatalf("expected fallback buys/sells to bids/asks, got bids=%d asks=%d", len(ev.Bids), len(ev.Asks))
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for orderbook compatibility event")
	}
}

func TestProcessEvent_SchemaCompat_MarketResolvedTypeAlias(t *testing.T) {
	c := newTestClient()
	ch := make(chan MarketResolvedEvent, 5)
	c.marketResolvedSubs["mr1"] = &subscriptionEntry[MarketResolvedEvent]{
		id: "mr1", ch: ch, errCh: make(chan error, 5),
	}

	raw := map[string]interface{}{
		"type":             "market_resolved",
		"market":           "m1",
		"asset_ids":        []interface{}{"a1"},
		"winning_asset_id": "a1",
		"winning_outcome":  "Yes",
		"ignored_field":    true,
	}
	c.processEvent(raw)

	select {
	case ev := <-ch:
		if ev.Market != "m1" || ev.WinningAssetID != "a1" {
			t.Fatalf("unexpected resolved event: %+v", ev)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("timeout waiting for market_resolved compatibility event")
	}
}
