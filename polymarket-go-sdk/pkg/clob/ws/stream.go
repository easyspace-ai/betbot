package ws

import "fmt"

// Stream delivers messages and async errors for a subscription.
type Stream[T any] struct {
	C      <-chan T
	Err    <-chan error
	closeF func() error
}

// Close stops the subscription and closes the stream.
func (s *Stream[T]) Close() error {
	if s == nil || s.closeF == nil {
		return nil
	}
	return s.closeF()
}

// LaggedError indicates the subscriber missed messages due to backpressure.
type LaggedError struct {
	Count     int
	Channel   Channel
	EventType EventType
}

func (e LaggedError) Error() string {
	if e.EventType == "" {
		return fmt.Sprintf("clobws subscription lagged, missed %d messages", e.Count)
	}
	return fmt.Sprintf("clobws subscription lagged, missed %d messages (channel=%s type=%s)", e.Count, e.Channel, e.EventType)
}
