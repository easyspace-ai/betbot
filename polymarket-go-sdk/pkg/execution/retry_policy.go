package execution

import (
	"context"
	"errors"
	"io"
	"net"
	"syscall"
	"time"
)

const (
	defaultRetryMaxAttempts = 3
	defaultRetryBaseBackoff = 150 * time.Millisecond
	defaultRetryMaxBackoff  = 2 * time.Second
)

// RetryPolicy defines retry strategy for network/timeout/5xx failure classes.
type RetryPolicy struct {
	MaxAttempts int
	BaseBackoff time.Duration
	MaxBackoff  time.Duration
}

// RetryDecision is the evaluated action for a failed attempt.
type RetryDecision struct {
	Retry  bool
	Delay  time.Duration
	Reason string
}

// DefaultRetryPolicy returns the SDK default retry policy.
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxAttempts: defaultRetryMaxAttempts,
		BaseBackoff: defaultRetryBaseBackoff,
		MaxBackoff:  defaultRetryMaxBackoff,
	}
}

// Decide evaluates whether the current failure should be retried.
//
// attempt is 1-based and represents the just-failed attempt number.
func (p RetryPolicy) Decide(attempt int, err error, statusCode int) RetryDecision {
	normalized := p.withDefaults()
	if attempt <= 0 {
		attempt = 1
	}
	if attempt >= normalized.MaxAttempts {
		return RetryDecision{Retry: false, Reason: "max_attempts_reached"}
	}

	switch {
	case IsRetryableError(err):
		return RetryDecision{
			Retry:  true,
			Delay:  normalized.ComputeBackoff(attempt),
			Reason: "retryable_error",
		}
	case IsRetryableStatusCode(statusCode):
		return RetryDecision{
			Retry:  true,
			Delay:  normalized.ComputeBackoff(attempt),
			Reason: "retryable_status",
		}
	default:
		return RetryDecision{Retry: false, Reason: "non_retryable"}
	}
}

// ComputeBackoff returns exponential backoff for the given attempt, capped by MaxBackoff.
//
// attempt is 1-based and corresponds to the just-failed attempt.
func (p RetryPolicy) ComputeBackoff(attempt int) time.Duration {
	normalized := p.withDefaults()
	if attempt <= 0 {
		attempt = 1
	}

	delay := normalized.BaseBackoff
	for i := 1; i < attempt; i++ {
		if delay >= normalized.MaxBackoff {
			return normalized.MaxBackoff
		}
		delay *= 2
		if delay > normalized.MaxBackoff {
			return normalized.MaxBackoff
		}
	}
	return delay
}

// IsRetryableStatusCode classifies retryable HTTP status codes.
func IsRetryableStatusCode(statusCode int) bool {
	return statusCode == 408 || (statusCode >= 500 && statusCode <= 599)
}

// IsRetryableError classifies retryable transport errors.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, io.EOF) {
		return true
	}
	if errors.Is(err, syscall.ECONNRESET) || errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.EPIPE) {
		return true
	}
	var unknownNetErr net.UnknownNetworkError
	if errors.As(err, &unknownNetErr) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true
		}
	}

	var opErr *net.OpError
	return errors.As(err, &opErr)
}

func (p RetryPolicy) withDefaults() RetryPolicy {
	out := p
	if out.MaxAttempts < 2 {
		out.MaxAttempts = defaultRetryMaxAttempts
	}
	if out.BaseBackoff <= 0 {
		out.BaseBackoff = defaultRetryBaseBackoff
	}
	if out.MaxBackoff <= 0 {
		out.MaxBackoff = defaultRetryMaxBackoff
	}
	if out.BaseBackoff > out.MaxBackoff {
		out.MaxBackoff = out.BaseBackoff
	}
	return out
}
