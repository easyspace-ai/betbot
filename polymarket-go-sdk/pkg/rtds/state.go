package rtds

import (
	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	stateTopic   = "connection_state"
	stateMsgType = "state"
)

type stateSubscription struct {
	id        string
	ch        chan ConnectionStateEvent
	errCh     chan error
	closed    atomic.Bool
	closeOnce sync.Once
}

func (s *stateSubscription) trySend(event ConnectionStateEvent) {
	if s.closed.Load() {
		return
	}
	defer func() {
		_ = recover()
	}()
	select {
	case s.ch <- event:
		return
	default:
		s.notifyLag(1)
	}
}

func (s *stateSubscription) notifyLag(count int) {
	if count <= 0 {
		return
	}
	select {
	case s.errCh <- LaggedError{Count: count, Topic: stateTopic, MsgType: stateMsgType}:
	default:
	}
}

func (s *stateSubscription) close() bool {
	if s.closed.Swap(true) {
		return false
	}
	s.closeOnce.Do(func() {
		close(s.ch)
		close(s.errCh)
	})
	return true
}

func (c *clientImpl) ConnectionStateStream(ctx context.Context) (*Stream[ConnectionStateEvent], error) {
	id := atomic.AddUint64(&c.nextStateID, 1)
	entry := &stateSubscription{
		id:    strconv.FormatUint(id, 10),
		ch:    make(chan ConnectionStateEvent, defaultStreamBuffer),
		errCh: make(chan error, defaultErrBuffer),
	}

	c.stateMu.Lock()
	c.stateSubs[entry.id] = entry
	c.stateMu.Unlock()

	stream := &Stream[ConnectionStateEvent]{
		C:   entry.ch,
		Err: entry.errCh,
		closeF: func() error {
			if entry.close() {
				c.stateMu.Lock()
				delete(c.stateSubs, entry.id)
				c.stateMu.Unlock()
			}
			return nil
		},
	}

	if ctx != nil {
		if done := ctx.Done(); done != nil {
			go func() {
				<-done
				_ = stream.Close()
			}()
		}
	}

	entry.trySend(ConnectionStateEvent{
		State:    c.ConnectionState(),
		Recorded: time.Now().UnixMilli(),
	})
	return stream, nil
}

func (c *clientImpl) setState(state ConnectionState) {
	switch state {
	case ConnectionConnected:
		atomic.StoreInt32(&c.state, connConnected)
	default:
		atomic.StoreInt32(&c.state, connDisconnected)
	}

	event := ConnectionStateEvent{
		State:    state,
		Recorded: time.Now().UnixMilli(),
	}
	c.stateMu.Lock()
	subs := make([]*stateSubscription, 0, len(c.stateSubs))
	for _, sub := range c.stateSubs {
		subs = append(subs, sub)
	}
	c.stateMu.Unlock()
	for _, sub := range subs {
		sub.trySend(event)
	}
}

func (c *clientImpl) closeStateSubscriptions() {
	c.stateMu.Lock()
	subs := make([]*stateSubscription, 0, len(c.stateSubs))
	for _, sub := range c.stateSubs {
		subs = append(subs, sub)
	}
	for key := range c.stateSubs {
		delete(c.stateSubs, key)
	}
	c.stateMu.Unlock()
	for _, sub := range subs {
		sub.close()
	}
}
