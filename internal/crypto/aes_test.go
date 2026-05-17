package crypto_test

import (
	"testing"

	"github.com/rguziy/ndrop/internal/crypto"
)

func TestBucketIDDeterministic(t *testing.T) {
	id1, err := crypto.BucketID("my-token")
	if err != nil {
		t.Fatal(err)
	}
	id2, err := crypto.BucketID("my-token")
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Fatalf("BucketID not deterministic: %q != %q", id1, id2)
	}
}

func TestBucketIDDiffersPerToken(t *testing.T) {
	id1, _ := crypto.BucketID("token-a")
	id2, _ := crypto.BucketID("token-b")
	if id1 == id2 {
		t.Fatal("different tokens must produce different bucket IDs")
	}
}

func TestEncryptDecryptRoundtrip(t *testing.T) {
	token := "secret-token-123"
	original := []byte("hello, ndrop!")

	datab64, nonceb64, err := crypto.Encrypt(token, original)
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}

	got, err := crypto.Decrypt(token, datab64, nonceb64)
	if err != nil {
		t.Fatalf("Decrypt: %v", err)
	}

	if string(got) != string(original) {
		t.Fatalf("roundtrip mismatch: got %q, want %q", got, original)
	}
}

func TestDecryptWrongToken(t *testing.T) {
	datab64, nonceb64, err := crypto.Encrypt("correct-token", []byte("secret"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = crypto.Decrypt("wrong-token", datab64, nonceb64)
	if err == nil {
		t.Fatal("expected error decrypting with wrong token, got nil")
	}
}

func TestEncryptNonceUnique(t *testing.T) {
	token := "same-token"
	payload := []byte("same payload")

	_, n1, _ := crypto.Encrypt(token, payload)
	_, n2, _ := crypto.Encrypt(token, payload)

	if n1 == n2 {
		t.Fatal("two Encrypt calls must produce different nonces")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	_, err := crypto.Decrypt("token", "not-valid-base64!!!", "also-bad")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}
