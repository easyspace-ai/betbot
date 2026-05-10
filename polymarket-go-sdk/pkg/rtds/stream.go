package rtds

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
	Count   int
	Topic   string
	MsgType string
}

func (e LaggedError) Error() string {
	if e.Topic == "" {
		return fmt.Sprintf("rtds subscription lagged, missed %d messages", e.Count)
	}
	return fmt.Sprintf("rtds subscription lagged, missed %d messages (topic=%s type=%s)", e.Count, e.Topic, e.MsgType)
}
