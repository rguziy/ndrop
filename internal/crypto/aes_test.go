package crypto_test

import (
	"testing"

	"github.com/rguziy/ndrop/internal/crypto"
)

func TestBucketIDDeterministic(t *testing.T) {
	id1, err := crypto.BucketID("my-api-key")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := crypto.BucketID("my-api-key")
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("BucketID not deterministic: %q != %q", id1, id2)
	}
}

func TestBucketIDDiffersPerAPIKey(t *testing.T) {
	id1, _ := crypto.BucketID("api-key-a")
	id2, _ := crypto.BucketID("api-key-b")
	if id1 == id2 {
		t.Fatal("different API keys must produce different bucket IDs")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	apiKey := "secret-api-key-123"
	original := []byte("hello, ndrop!")

	datab64, nonceb64, err := crypto.Encrypt(apiKey, original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := crypto.Decrypt(apiKey, datab64, nonceb64)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(got) != string(original) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", got, original)
	}
}

func TestDecryptWrongAPIKey(t *testing.T) {
	datab64, nonceb64, err := crypto.Encrypt("correct-api-key", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = crypto.Decrypt("wrong-api-key", datab64, nonceb64)
	if err == nil {
		t.Fatal("expected error decrypting with wrong API key, got nil")
	}
}

func TestEncryptNonceUnique(t *testing.T) {
	apiKey := "same-api-key"
	payload := []byte("same payload")

	_, n1, _ := crypto.Encrypt(apiKey, payload)
	_, n2, _ := crypto.Encrypt(apiKey, payload)

	if n1 == n2 {
		t.Fatal("two Encrypt calls must produce different nonces")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	_, err := crypto.Decrypt("api-key", "not-valid-base64!!!", "also-bad")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}
