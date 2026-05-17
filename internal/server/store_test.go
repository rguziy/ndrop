package server_test

import (
	"testing"
	"time"

	"github.com/rguziy/ndrop/internal/server"
)

func makeEntry(device string) server.Entry {
	return server.Entry{
		Device: device,
		Type:   server.EntryTypeText,
		Mime:   "text/plain",
		Data:   "encrypted-data",
		Nonce:  "nonce",
	}
}

func TestStoreSetAndGet(t *testing.T) {
	s := server.NewStore(1 * time.Minute)

	s.Set("bucket-1", makeEntry("mac"))

	e, ok := s.Get("bucket-1")
	if !ok {
		t.Fatal("expected entry, got nothing")
	}
	if e.Device != "mac" {
		t.Fatalf("got device %q, want %q", e.Device, "mac")
	}
}

func TestStoreLastWriteWins(t *testing.T) {
	s := server.NewStore(1 * time.Minute)

	s.Set("bucket-1", makeEntry("first"))
	s.Set("bucket-1", makeEntry("second"))

	e, ok := s.Get("bucket-1")
	if !ok {
		t.Fatal("expected entry")
	}
	if e.Device != "second" {
		t.Fatalf("expected last writer to win, got %q", e.Device)
	}
}

func TestStoreMissingKey(t *testing.T) {
	s := server.NewStore(1 * time.Minute)

	_, ok := s.Get("nonexistent")
	if ok {
		t.Fatal("expected not found for missing key")
	}
}

func TestStoreDelete(t *testing.T) {
	s := server.NewStore(1 * time.Minute)

	s.Set("bucket-1", makeEntry("mac"))
	s.Delete("bucket-1")

	_, ok := s.Get("bucket-1")
	if ok {
		t.Fatal("expected entry to be deleted")
	}
}

func TestStoreTTLExpiry(t *testing.T) {
	s := server.NewStore(50 * time.Millisecond)

	s.Set("bucket-1", makeEntry("mac"))

	_, ok := s.Get("bucket-1")
	if !ok {
		t.Fatal("entry should exist before TTL")
	}

	time.Sleep(100 * time.Millisecond)

	_, ok = s.Get("bucket-1")
	if ok {
		t.Fatal("entry should have expired")
	}
}
