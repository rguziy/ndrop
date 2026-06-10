package server

import (
	"sync"
	"time"
)

// EntryType distinguishes text, file, and folder payloads.
type EntryType string

const (
	EntryTypeText   EntryType = "text"
	EntryTypeFile   EntryType = "file"
	EntryTypeFolder EntryType = "folder"
)

// Entry is the value stored per bucket. All content fields come directly
// from the client request; the server never decrypts them.
type Entry struct {
	Device    string    `json:"device"`
	Type      EntryType `json:"type"`
	Name      string    `json:"name"` // original filename; empty for text
	Mime      string    `json:"mime"`
	Data      string    `json:"data"`  // base64(AES-256-GCM(payload))
	Nonce     string    `json:"nonce"` // base64(12-byte GCM nonce)
	ExpiresAt time.Time `json:"-"`
}

// Store is an in-memory, TTL-aware key-value store.
// One entry per bucket; Set always overwrites (last-write-wins).
type Store struct {
	mu      sync.RWMutex
	entries map[string]Entry
	ttl     time.Duration
}

// NewStore creates a Store with the given TTL and starts a background goroutine
// that purges expired entries every ttl/2 (minimum 30 seconds).
func NewStore(ttl time.Duration) *Store {
	s := &Store{
		entries: make(map[string]Entry),
		ttl:     ttl,
	}

	interval := ttl / 2
	if interval < 30*time.Second {
		interval = 30 * time.Second
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for range ticker.C {
			s.purge()
		}
	}()

	return s
}

// Set stores or replaces the entry for bucketID, resetting its TTL.
func (s *Store) Set(bucketID string, e Entry) {
	e.ExpiresAt = time.Now().Add(s.ttl)
	s.mu.Lock()
	s.entries[bucketID] = e
	s.mu.Unlock()
}

// Get retrieves the entry for bucketID.
// Returns (entry, true) if present and not expired; (Entry{}, false) otherwise.
// Expired entries are lazily deleted on access.
func (s *Store) Get(bucketID string) (Entry, bool) {
	s.mu.RLock()
	e, ok := s.entries[bucketID]
	s.mu.RUnlock()

	if !ok {
		return Entry{}, false
	}

	if time.Now().After(e.ExpiresAt) {
		s.mu.Lock()
		delete(s.entries, bucketID)
		s.mu.Unlock()
		return Entry{}, false
	}

	return e, true
}

// Delete removes the entry for bucketID if it exists.
func (s *Store) Delete(bucketID string) {
	s.mu.Lock()
	delete(s.entries, bucketID)
	s.mu.Unlock()
}

// purge removes all entries whose TTL has elapsed.
func (s *Store) purge() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, e := range s.entries {
		if now.After(e.ExpiresAt) {
			delete(s.entries, id)
		}
	}
}
