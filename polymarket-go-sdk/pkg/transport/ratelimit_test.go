package transport

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewRateLimiter(t *testing.T) {
	tests := []struct {
		name               string
		requestsPerSecond  int
		expectedCapacity   int
		expectedAvailable  int
	}{
		{
			name:              "valid rate",
			requestsPerSecond: 10,
			expectedCapacity:  10,
			expectedAvailable: 10,
		},
		{
			name:              "zero rate defaults to 10",
			requestsPerSecond: 0,
			expectedCapacity:  10,
			expectedAvailable: 10,
		},
		{
			name:              "negative rate defaults to 10",
			requestsPerSecond: -5,
			expectedCapacity:  10,
			expectedAvailable: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rl := NewRateLimiter(tt.requestsPerSecond)
			if rl == nil {
				t.Fatal("NewRateLimiter returned nil")
			}
			if rl.Capacity() != tt.expectedCapacity {
				t.Errorf("Capacity() = %d, want %d", rl.Capacity(), tt.expectedCapacity)
			}
			if rl.Available() != tt.expectedAvailable {
				t.Errorf("Available() = %d, want %d", rl.Available(), tt.expectedAvailable)
			}
		})
	}
}

func TestRateLimiter_Wait(t *testing.T) {
	t.Run("successful wait", func(t *testing.T) {
		rl := NewRateLimiter(10)
		rl.Start()
		defer rl.Stop()

		ctx := context.Background()
		err := rl.Wait(ctx)
		if err != nil {
			t.Errorf("Wait() error = %v, want nil", err)
		}

		// Should have consumed one token
		if rl.Available() != 9 {
			t.Errorf("Available() = %d, want 9", rl.Available())
		}
	})

	t.Run("context cancelled", func(t *testing.T) {
		rl := NewRateLimiter(1)
		rl.Start()
		defer rl.Stop()

		// Consume the only token
		_ = rl.Wait(context.Background())

		// Now try to wait with a cancelled context
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		err := rl.Wait(ctx)
		if err != context.Canceled {
			t.Errorf("Wait() error = %v, want %v", err, context.Canceled)
		}
	})

	t.Run("context timeout", func(t *testing.T) {
		rl := NewRateLimiter(1)
		rl.Start()
		defer rl.Stop()

		// Consume the only token
		_ = rl.Wait(context.Background())

		// Try to wait with a short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		err := rl.Wait(ctx)
		if err != context.DeadlineExceeded {
			t.Errorf("Wait() error = %v, want %v", err, context.DeadlineExceeded)
		}
	})
}

func TestRateLimiter_TryAcquire(t *testing.T) {
	t.Run("successful acquire", func(t *testing.T) {
		rl := NewRateLimiter(10)
		rl.Start()
		defer rl.Stop()

		if !rl.TryAcquire() {
			t.Error("TryAcquire() = false, want true")
		}

		if rl.Available() != 9 {
			t.Errorf("Available() = %d, want 9", rl.Available())
		}
	})

	t.Run("failed acquire when empty", func(t *testing.T) {
		rl := NewRateLimiter(1)
		rl.Start()
		defer rl.Stop()

		// Consume the only token
		if !rl.TryAcquire() {
			t.Error("First TryAcquire() = false, want true")
		}

		// Second attempt should fail
		if rl.TryAcquire() {
			t.Error("Second TryAcquire() = true, want false")
		}
	})
}

func TestRateLimiter_Refill(t *testing.T) {
	t.Run("tokens refill over time", func(t *testing.T) {
		rl := NewRateLimiter(10) // 10 tokens per second = 100ms per token
		rl.Start()
		defer rl.Stop()

		// Consume all tokens
		for i := 0; i < 10; i++ {
			if !rl.TryAcquire() {
				t.Fatalf("Failed to acquire token %d", i)
			}
		}

		if rl.Available() != 0 {
			t.Errorf("Available() = %d, want 0", rl.Available())
		}

		// Wait for refill (at least 2 tokens should be refilled in 250ms)
		time.Sleep(250 * time.Millisecond)

		available := rl.Available()
		if available < 2 {
			t.Errorf("Available() = %d, want at least 2", available)
		}
	})
}

func TestRateLimiter_Concurrent(t *testing.T) {
	t.Run("concurrent access", func(t *testing.T) {
		rl := NewRateLimiter(100)
		rl.Start()
		defer rl.Stop()

		var wg sync.WaitGroup
		var successCount atomic.Int32
		goroutines := 50

		for i := 0; i < goroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()

				if err := rl.Wait(ctx); err == nil {
					successCount.Add(1)
				}
			}()
		}

		wg.Wait()

		// All goroutines should succeed since we have 100 tokens and only 50 goroutines
		if successCount.Load() != int32(goroutines) {
			t.Errorf("successCount = %d, want %d", successCount.Load(), goroutines)
		}
	})
}

func TestRateLimiter_StartStop(t *testing.T) {
	t.Run("start and stop", func(t *testing.T) {
		rl := NewRateLimiter(10)

		// Start should be idempotent
		rl.Start()
		rl.Start()

		// Consume all tokens
		for i := 0; i < 10; i++ {
			rl.TryAcquire()
		}

		// Stop the refill
		rl.Stop()

		// Wait a bit
		time.Sleep(200 * time.Millisecond)

		// Tokens should not have been refilled
		if rl.Available() != 0 {
			t.Errorf("Available() = %d, want 0 (refill should be stopped)", rl.Available())
		}
	})

	t.Run("stop without start", func(t *testing.T) {
		rl := NewRateLimiter(10)
		// Should not panic
		rl.Stop()
	})

	t.Run("stop twice", func(t *testing.T) {
		rl := NewRateLimiter(10)
		rl.Start()
		rl.Stop()

		done := make(chan struct{})
		go func() {
			rl.Stop()
			close(done)
		}()

		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatal("second Stop blocked")
		}
	})
}

func TestRateLimiter_RateEnforcement(t *testing.T) {
	t.Run("enforces rate limit", func(t *testing.T) {
		requestsPerSecond := 10
		rl := NewRateLimiter(requestsPerSecond)
		rl.Start()
		defer rl.Stop()

		// Consume all initial tokens
		for i := 0; i < requestsPerSecond; i++ {
			rl.TryAcquire()
		}

		// Measure how many requests we can make in 1 second
		start := time.Now()
		count := 0
		deadline := start.Add(1 * time.Second)

		for time.Now().Before(deadline) {
			ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
			if err := rl.Wait(ctx); err == nil {
				count++
			}
			cancel()
		}

		// We should get approximately requestsPerSecond tokens
		// Allow some tolerance (±20%)
		minExpected := int(float64(requestsPerSecond) * 0.8)
		maxExpected := int(float64(requestsPerSecond) * 1.2)

		if count < minExpected || count > maxExpected {
			t.Errorf("Got %d requests in 1 second, expected between %d and %d", count, minExpected, maxExpected)
		}
	})
}
