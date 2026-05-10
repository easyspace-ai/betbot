package execution

import "testing"

func TestBuildIdempotencyKeyRequiresFields(t *testing.T) {
	_, err := BuildIdempotencyKey(IdempotencyKeyInput{
		Tenant:        "tenant-a",
		Strategy:      "strategy-a",
		ClientOrderID: "",
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestBuildIdempotencyKeyCanonicalAndDeterministic(t *testing.T) {
	in := IdempotencyKeyInput{
		Tenant:        " Tenant-A ",
		Strategy:      " Strategy-A ",
		ClientOrderID: "order-123",
	}

	k1, err := BuildIdempotencyKey(in)
	if err != nil {
		t.Fatalf("build key: %v", err)
	}
	k2, err := BuildIdempotencyKey(in)
	if err != nil {
		t.Fatalf("build key second: %v", err)
	}

	if k1.Value == "" {
		t.Fatalf("expected key value")
	}
	if k1.Value != k2.Value {
		t.Fatalf("expected deterministic value, got %q vs %q", k1.Value, k2.Value)
	}
	if k1.Canonical != "tenant=tenant-a;strategy=strategy-a;client_order_id=order-123" {
		t.Fatalf("unexpected canonical form: %q", k1.Canonical)
	}
	if k1.Version != IdempotencyKeyVersionV1 {
		t.Fatalf("unexpected version: %q", k1.Version)
	}
}

func TestParseIdempotencyCanonical(t *testing.T) {
	parts, err := ParseIdempotencyCanonical("tenant=t1;strategy=s1;client_order_id=o1")
	if err != nil {
		t.Fatalf("parse canonical: %v", err)
	}
	if parts.Tenant != "t1" || parts.Strategy != "s1" || parts.ClientOrderID != "o1" {
		t.Fatalf("unexpected parsed parts: %+v", parts)
	}
}

func TestBuildIdempotencyKeyChangesWhenInputChanges(t *testing.T) {
	base, err := BuildIdempotencyKey(IdempotencyKeyInput{
		Tenant:        "t1",
		Strategy:      "s1",
		ClientOrderID: "o1",
	})
	if err != nil {
		t.Fatalf("build base key: %v", err)
	}
	changed, err := BuildIdempotencyKey(IdempotencyKeyInput{
		Tenant:        "t1",
		Strategy:      "s2",
		ClientOrderID: "o1",
	})
	if err != nil {
		t.Fatalf("build changed key: %v", err)
	}
	if base.Value == changed.Value {
		t.Fatalf("expected different key values for different inputs")
	}
}
