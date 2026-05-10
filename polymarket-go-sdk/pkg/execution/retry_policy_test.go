package execution

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"
)

type timeoutNetErr struct{}

func (timeoutNetErr) Error() string   { return "timeout" }
func (timeoutNetErr) Timeout() bool   { return true }
func (timeoutNetErr) Temporary() bool { return true }

func TestDefaultRetryPolicy(t *testing.T) {
	p := DefaultRetryPolicy()
	if p.MaxAttempts < 2 {
		t.Fatalf("expected max attempts >=2, got %d", p.MaxAttempts)
	}
	if p.BaseBackoff <= 0 || p.MaxBackoff <= 0 {
		t.Fatalf("expected positive backoff values")
	}
}

func TestRetryPolicyDecideRetriesTimeout(t *testing.T) {
	p := DefaultRetryPolicy()
	decision := p.Decide(1, context.DeadlineExceeded, 0)
	if !decision.Retry {
		t.Fatalf("expected timeout to be retryable")
	}
	if decision.Reason == "" {
		t.Fatalf("expected retry reason")
	}
}

func TestRetryPolicyDecideRetriesNetworkError(t *testing.T) {
	p := DefaultRetryPolicy()
	var err error = timeoutNetErr{}
	decision := p.Decide(1, err, 0)
	if !decision.Retry {
		t.Fatalf("expected net timeout to be retryable")
	}
}

func TestRetryPolicyDecideRetriesHTTP5xx(t *testing.T) {
	p := DefaultRetryPolicy()
	decision := p.Decide(1, nil, 503)
	if !decision.Retry {
		t.Fatalf("expected 503 to be retryable")
	}
}

func TestRetryPolicyDecideDoesNotRetryHTTP4xx(t *testing.T) {
	p := DefaultRetryPolicy()
	decision := p.Decide(1, nil, 400)
	if decision.Retry {
		t.Fatalf("expected 400 to be non-retryable")
	}
}

func TestRetryPolicyDecideStopsAtMaxAttempts(t *testing.T) {
	p := RetryPolicy{
		MaxAttempts: 2,
		BaseBackoff: 10 * time.Millisecond,
		MaxBackoff:  1 * time.Second,
	}
	decision := p.Decide(2, errors.New("boom"), 503)
	if decision.Retry {
		t.Fatalf("expected no retry at max attempts")
	}
}

func TestComputeBackoffCapsAtMax(t *testing.T) {
	p := RetryPolicy{
		MaxAttempts: 5,
		BaseBackoff: 50 * time.Millisecond,
		MaxBackoff:  120 * time.Millisecond,
	}
	delay := p.ComputeBackoff(4)
	if delay != 120*time.Millisecond {
		t.Fatalf("expected capped backoff, got %s", delay)
	}
}

func TestRetryableErrorHelper(t *testing.T) {
	if !IsRetryableError(net.UnknownNetworkError("x")) {
		t.Fatalf("expected unknown network error retryable")
	}
	if IsRetryableError(errors.New("plain")) {
		t.Fatalf("expected plain error not retryable")
	}
}
