package execution

import (
	"testing"
	"time"
)

func TestDefaultWSPolicy(t *testing.T) {
	p := DefaultWSPolicy()
	if !p.ReconnectEnabled {
		t.Fatalf("expected reconnect enabled by default")
	}
	if p.ReconnectBaseDelay <= 0 || p.ReconnectMaxDelay <= 0 {
		t.Fatalf("expected positive reconnect delays")
	}
	if p.HeartbeatInterval <= 0 || p.HeartbeatTimeout <= 0 {
		t.Fatalf("expected heartbeat values to be set")
	}
}

func TestWSPolicyNextReconnectDelay(t *testing.T) {
	p := WSPolicy{
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   100 * time.Millisecond,
		ReconnectMaxDelay:    1 * time.Second,
		ReconnectMultiplier:  2.0,
		ReconnectMaxAttempts: 5,
	}

	delay, ok := p.NextReconnectDelay(3)
	if !ok {
		t.Fatalf("expected reconnect allowed")
	}
	if delay != 400*time.Millisecond {
		t.Fatalf("unexpected delay: %s", delay)
	}
}

func TestWSPolicyNextReconnectDelayCapsAndStops(t *testing.T) {
	p := WSPolicy{
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   200 * time.Millisecond,
		ReconnectMaxDelay:    500 * time.Millisecond,
		ReconnectMultiplier:  3.0,
		ReconnectMaxAttempts: 2,
	}

	delay, ok := p.NextReconnectDelay(2)
	if !ok {
		t.Fatalf("expected attempt 2 to be allowed")
	}
	if delay != 500*time.Millisecond {
		t.Fatalf("expected capped delay, got %s", delay)
	}

	_, ok = p.NextReconnectDelay(3)
	if ok {
		t.Fatalf("expected attempt 3 to be blocked by max attempts")
	}
}

func TestWSPolicyHeartbeatExpired(t *testing.T) {
	p := WSPolicy{
		HeartbeatInterval: 10 * time.Second,
		HeartbeatTimeout:  30 * time.Second,
	}
	now := time.Unix(1710000000, 0).UTC()

	if p.IsHeartbeatExpired(now.Add(-20*time.Second), now) {
		t.Fatalf("expected heartbeat not expired")
	}
	if !p.IsHeartbeatExpired(now.Add(-31*time.Second), now) {
		t.Fatalf("expected heartbeat expired")
	}
}

func TestWSPolicyToCLOBConfig(t *testing.T) {
	p := WSPolicy{
		ReconnectEnabled:     true,
		ReconnectBaseDelay:   250 * time.Millisecond,
		ReconnectMaxDelay:    2 * time.Second,
		ReconnectMultiplier:  1.5,
		ReconnectMaxAttempts: 7,
		HeartbeatInterval:    5 * time.Second,
		HeartbeatTimeout:     20 * time.Second,
		DisablePing:          true,
	}
	cfg := p.ToCLOBConfig()
	if !cfg.DisablePing || cfg.ReconnectDelay != 250*time.Millisecond || cfg.ReconnectMax != 7 {
		t.Fatalf("unexpected ws config: %+v", cfg)
	}
}
