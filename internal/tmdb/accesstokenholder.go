package tmdb

import "sync"

type AccessTokenHolder struct {
	mu  sync.RWMutex
	key string
}

func NewAccessTokenHolder(initialToken string) *AccessTokenHolder {
	return &AccessTokenHolder{key: initialToken}
}

func (h *AccessTokenHolder) Get() string {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.key
}

func (h *AccessTokenHolder) Set(key string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.key = key
}
