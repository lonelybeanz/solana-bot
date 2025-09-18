package pubsub

import (
	"sync"
)

type Subscriber chan interface{}

type PubSub struct {
	mu          sync.RWMutex
	subscribers map[Subscriber]struct{}
}

func NewPubSub() *PubSub {
	return &PubSub{
		subscribers: make(map[Subscriber]struct{}),
	}
}

// Subscribe returns a new channel subscribed to broadcasts.
func (ps *PubSub) Subscribe() Subscriber {
	ch := make(Subscriber)
	ps.mu.Lock()
	ps.subscribers[ch] = struct{}{}
	ps.mu.Unlock()
	return ch
}

// Unsubscribe removes the channel from subscribers.
func (ps *PubSub) Unsubscribe(ch Subscriber) {
	ps.mu.Lock()
	delete(ps.subscribers, ch)
	close(ch)
	ps.mu.Unlock()
}

// Publish sends a message to all subscribers.
func (ps *PubSub) Publish(msg interface{}) {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	for ch := range ps.subscribers {
		select {
		case ch <- msg:
		default:
			// drop message if channel is full to avoid blocking
		}
	}
}
