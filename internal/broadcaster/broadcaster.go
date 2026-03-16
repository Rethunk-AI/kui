package broadcaster

import (
	"context"
	"sync"
)

// Event represents an SSE event. Type is the event name (e.g. vm.state_changed).
// Data is the JSON-serializable payload.
type Event struct {
	Type string
	Data any
}

// Subscription is a handle to a stream of events. Call Done when finished.
type Subscription struct {
	C    <-chan Event
	done func()
}

// Done unsubscribes and closes the channel. Safe to call multiple times.
func (s *Subscription) Done() {
	if s.done != nil {
		s.done()
		s.done = nil
	}
}

// Broadcaster distributes events to subscribers. For MVP it uses an in-memory
// fan-out. No libvirt integration yet.
type Broadcaster struct {
	mu          sync.RWMutex
	subscribers map[chan Event]struct{}
}

// NewBroadcaster creates a new broadcaster.
func NewBroadcaster() *Broadcaster {
	return &Broadcaster{
		subscribers: make(map[chan Event]struct{}),
	}
}

// Subscribe returns a subscription that receives events. Call Done when done.
// For MVP, a placeholder host.online event is sent immediately so clients can
// verify the stream works.
func (b *Broadcaster) Subscribe(ctx context.Context) *Subscription {
	ch := make(chan Event, 8)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()

	done := func() {
		b.mu.Lock()
		delete(b.subscribers, ch)
		b.mu.Unlock()
		close(ch)
	}

	// MVP: emit placeholder event on connect so clients can verify the stream.
	select {
	case ch <- Event{Type: "host.online", Data: map[string]string{"host_id": "kui"}}:
	case <-ctx.Done():
		done()
		return &Subscription{C: ch, done: nil}
	}

	return &Subscription{C: ch, done: done}
}

// Broadcast sends the event to all subscribers. Non-blocking; drops if a
// subscriber's channel is full.
func (b *Broadcaster) Broadcast(ev Event) {
	b.mu.RLock()
	subs := make([]chan Event, 0, len(b.subscribers))
	for ch := range b.subscribers {
		subs = append(subs, ch)
	}
	b.mu.RUnlock()

	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
			// Subscriber slow; skip to avoid blocking
		}
	}
}
