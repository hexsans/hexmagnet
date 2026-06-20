package indexer

import "sync"

type ConfigNotifier struct {
	mu     sync.Mutex
	subs   map[int]chan struct{}
	nextID int
}

func NewConfigNotifier() *ConfigNotifier {
	return &ConfigNotifier{
		subs: make(map[int]chan struct{}),
	}
}

func (n *ConfigNotifier) Subscribe() (<-chan struct{}, func()) {
	n.mu.Lock()
	defer n.mu.Unlock()

	ch := make(chan struct{}, 1)
	id := n.nextID
	n.nextID++
	n.subs[id] = ch

	return ch, func() {
		n.mu.Lock()
		defer n.mu.Unlock()

		delete(n.subs, id)
		close(ch)
	}
}

func (n *ConfigNotifier) Notify() {
	n.mu.Lock()
	defer n.mu.Unlock()

	for _, ch := range n.subs {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
