package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/hkdf"
)

const (
	infoEncrypt = "ndrop-encrypt"
	infoBucket  = "ndrop-bucket"
	keyLen      = 32 // AES-256
)

// deriveKey runs HKDF-SHA256 over the API key with the given info label.
func deriveKey(apiKey, info string) ([]byte, error) {
	r := hkdf.New(sha256.New, []byte(apiKey), nil, []byte(info))
	key := make([]byte, keyLen)
	if _, err := io.ReadFull(r, key); err != nil {
		return nil, err
	}
	return key, nil
}

// BucketID derives a stable bucket identifier from an API key.
// The server uses this as a map key and does not store the raw API key.
func BucketID(apiKey string) (string, error) {
	key, err := deriveKey(apiKey, infoBucket)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(key), nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a key derived from an API key.
// Returns (base64(ciphertext), base64(nonce), error).
func Encrypt(apiKey string, plaintext []byte) (string, string, error) {
	encKey, err := deriveKey(apiKey, infoEncrypt)
	if err != nil {
		return "", "", err
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return "", "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", "", err
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", "", err
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	return base64.StdEncoding.EncodeToString(ciphertext),
		base64.StdEncoding.EncodeToString(nonce),
		nil
}

// Decrypt decrypts a base64-encoded ciphertext+nonce pair using the API key.
func Decrypt(apiKey, datab64, nonceb64 string) ([]byte, error) {
	encKey, err := deriveKey(apiKey, infoEncrypt)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(datab64)
	if err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(nonceb64)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(encKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid nonce size")
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: invalid API key or corrupted data")
	}

	return plaintext, nil
}
