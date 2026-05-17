package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/rguziy/ndrop/internal/crypto"
)

// pushRequest is the JSON body accepted by POST /push.
type pushRequest struct {
	Device string    `json:"device"`
	Type   EntryType `json:"type"`
	Name   string    `json:"name"`
	Mime   string    `json:"mime"`
	Data   string    `json:"data"`
	Nonce  string    `json:"nonce"`
}

// pullResponse is the JSON body returned by GET /pull.
type pullResponse struct {
	Device string    `json:"device"`
	Type   EntryType `json:"type"`
	Name   string    `json:"name"`
	Mime   string    `json:"mime"`
	Data   string    `json:"data"`
	Nonce  string    `json:"nonce"`
}

// Handler holds the dependencies shared across HTTP handlers.
type Handler struct {
	store          *Store
	maxBytes       int64
	allowAnyAPIKey bool
	allowedAPIKeys map[string]struct{}
}

// AuthConfig controls which API keys may use the server.
type AuthConfig struct {
	AllowAnyAPIKey bool
	AllowedAPIKeys []string
}

// NewHandler creates a Handler and registers routes on mux.
func NewHandler(store *Store, maxBytes int64, auth AuthConfig) http.Handler {
	allowedAPIKeys := make(map[string]struct{}, len(auth.AllowedAPIKeys))
	for _, apiKey := range auth.AllowedAPIKeys {
		apiKey = strings.TrimSpace(apiKey)
		if apiKey != "" {
			allowedAPIKeys[apiKey] = struct{}{}
		}
	}

	h := &Handler{
		store:          store,
		maxBytes:       maxBytes,
		allowAnyAPIKey: auth.AllowAnyAPIKey,
		allowedAPIKeys: allowedAPIKeys,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/push", h.handlePush)
	mux.HandleFunc("/pull", h.handlePull)

	return mux
}

// extractAPIKey parses the Bearer API key from the Authorization header.
func extractAPIKey(r *http.Request) (string, bool) {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return "", false
	}
	apiKey := strings.TrimPrefix(header, "Bearer ")
	apiKey = strings.TrimSpace(apiKey)
	if apiKey == "" {
		return "", false
	}
	return apiKey, true
}

func (h *Handler) isAPIKeyAllowed(apiKey string) bool {
	if h.allowAnyAPIKey {
		return true
	}
	_, ok := h.allowedAPIKeys[apiKey]
	return ok
}

// handlePush handles POST /push.
func (h *Handler) handlePush(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey, ok := extractAPIKey(r)
	if !ok || !h.isAPIKeyAllowed(apiKey) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Enforce size limit before reading the body.
	r.Body = http.MaxBytesReader(w, r.Body, h.maxBytes)

	var req pushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		if err.Error() == "http: request body too large" {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if err := validatePushRequest(req); err != nil {
		http.Error(w, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	bucketID, err := crypto.BucketID(apiKey)
	if err != nil {
		log.Printf("bucket derivation error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	h.store.Set(bucketID, Entry{
		Device: req.Device,
		Type:   req.Type,
		Name:   req.Name,
		Mime:   req.Mime,
		Data:   req.Data,
		Nonce:  req.Nonce,
	})

	log.Printf("push  bucket=%.8s... device=%q type=%s", bucketID, req.Device, req.Type)
	w.WriteHeader(http.StatusOK)
}

// handlePull handles GET /pull.
func (h *Handler) handlePull(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	apiKey, ok := extractAPIKey(r)
	if !ok || !h.isAPIKeyAllowed(apiKey) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	bucketID, err := crypto.BucketID(apiKey)
	if err != nil {
		log.Printf("bucket derivation error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	entry, found := h.store.Get(bucketID)
	if !found {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	log.Printf("pull  bucket=%.8s... device=%q type=%s", bucketID, entry.Device, entry.Type)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(pullResponse{
		Device: entry.Device,
		Type:   entry.Type,
		Name:   entry.Name,
		Mime:   entry.Mime,
		Data:   entry.Data,
		Nonce:  entry.Nonce,
	})
}

// validatePushRequest returns an error if required fields are missing or invalid.
func validatePushRequest(req pushRequest) error {
	if req.Type != EntryTypeText && req.Type != EntryTypeFile {
		return errorf("type must be 'text' or 'file', got %q", req.Type)
	}
	if req.Data == "" {
		return errorf("data is required")
	}
	if req.Nonce == "" {
		return errorf("nonce is required")
	}
	if req.Type == EntryTypeFile && req.Name == "" {
		return errorf("name is required for type 'file'")
	}
	return nil
}

func errorf(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}
