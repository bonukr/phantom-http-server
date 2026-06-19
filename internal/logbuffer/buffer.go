// Package logbuffer stores recent API request logs in memory for the GUI.
package logbuffer

import (
	"strings"
	"sync"
	"time"
)

const defaultCapacity = 500

// Entry is one recorded API request.
type Entry struct {
	ID        int64             `json:"id"`
	Time      time.Time         `json:"time"`
	Path      string            `json:"path"`
	Method    string            `json:"method"`
	ClientIP  string            `json:"clientIp"`
	Query     string            `json:"query,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Body      string            `json:"body,omitempty"`
	BodySize  int               `json:"bodySize"`
	Status    int               `json:"status"`
}

// Buffer is a thread-safe ring buffer with pub/sub for live updates.
type Buffer struct {
	mu          sync.RWMutex
	entries     []Entry
	capacity    int
	nextID      int64
	subscribers map[chan Entry]struct{}
}

// New creates a ring buffer with the given capacity.
func New(capacity int) *Buffer {
	if capacity < 1 {
		capacity = defaultCapacity
	}
	return &Buffer{
		entries:     make([]Entry, 0, capacity),
		capacity:    capacity,
		subscribers: make(map[chan Entry]struct{}),
	}
}

// Add stores a new log entry and notifies subscribers.
func (b *Buffer) Add(e Entry) Entry {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.nextID++
	e.ID = b.nextID
	if e.Time.IsZero() {
		e.Time = time.Now().UTC()
	}

	if len(b.entries) >= b.capacity {
		copy(b.entries, b.entries[1:])
		b.entries[len(b.entries)-1] = e
	} else {
		b.entries = append(b.entries, e)
	}

	for ch := range b.subscribers {
		select {
		case ch <- e:
		default:
		}
	}
	return e
}

// List returns entries matching optional filters, newest first.
func (b *Buffer) List(path, method, text string, limit int) []Entry {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if limit <= 0 {
		limit = 100
	}

	path = normalizeFilter(path)
	method = normalizeFilter(method)
	text = normalizeFilter(text)

	out := make([]Entry, 0, limit)
	for i := len(b.entries) - 1; i >= 0 && len(out) < limit; i-- {
		e := b.entries[i]
		if path != "" && e.Path != path {
			continue
		}
		if method != "" && e.Method != method {
			continue
		}
		if text != "" && !containsText(e, text) {
			continue
		}
		out = append(out, e)
	}
	return out
}

// Clear removes all stored log entries.
func (b *Buffer) Clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = make([]Entry, 0, b.capacity)
}

// Paths returns distinct API paths seen in the buffer.
func (b *Buffer) Paths() []string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	seen := make(map[string]struct{})
	for _, e := range b.entries {
		seen[e.Path] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	return out
}

// Subscribe registers a channel for live entry notifications.
func (b *Buffer) Subscribe() chan Entry {
	ch := make(chan Entry, 16)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (b *Buffer) Unsubscribe(ch chan Entry) {
	b.mu.Lock()
	delete(b.subscribers, ch)
	b.mu.Unlock()
	close(ch)
}

func normalizeFilter(s string) string {
	return strings.TrimSpace(s)
}

func containsText(e Entry, text string) bool {
	text = strings.ToLower(text)
	if strings.Contains(strings.ToLower(e.Body), text) {
		return true
	}
	if strings.Contains(strings.ToLower(e.Query), text) {
		return true
	}
	if strings.Contains(strings.ToLower(e.ClientIP), text) {
		return true
	}
	for k, v := range e.Headers {
		if strings.Contains(strings.ToLower(k), text) || strings.Contains(strings.ToLower(v), text) {
			return true
		}
	}
	return false
}
