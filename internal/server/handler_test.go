package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/rguziy/ndrop/internal/server"
)

// newTestServer creates a test HTTP server with a short TTL and 1 MB limit.
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	store := server.NewStore(1 * time.Minute)
	handler := server.NewHandler(store, 1<<20) // 1 MB
	return httptest.NewServer(handler)
}

func pushPayload(t *testing.T, srv *httptest.Server, token string, body map[string]any) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", srv.URL+"/push", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("push request: %v", err)
	}
	return resp
}

func pullPayload(t *testing.T, srv *httptest.Server, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", srv.URL+"/pull", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("pull request: %v", err)
	}
	return resp
}

func validBody() map[string]any {
	return map[string]any{
		"device": "test-device",
		"type":   "text",
		"name":   "",
		"mime":   "text/plain",
		"data":   "dGVzdA==", // base64("test")
		"nonce":  "AAAAAAAAAAAAAAAA", // 12 bytes base64
	}
}

// --- push tests ---

func TestPushOK(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := pushPayload(t, srv, "token-a", validBody())
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
}

func TestPushNoAuth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	b, _ := json.Marshal(validBody())
	req, _ := http.NewRequest("POST", srv.URL+"/push", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestPushMissingData(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body := validBody()
	delete(body, "data")
	resp := pushPayload(t, srv, "token-a", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPushInvalidType(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body := validBody()
	body["type"] = "image" // not allowed
	resp := pushPayload(t, srv, "token-a", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", resp.StatusCode)
	}
}

func TestPushFileRequiresName(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	body := validBody()
	body["type"] = "file"
	body["name"] = ""
	resp := pushPayload(t, srv, "token-a", body)
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 when file has no name, got %d", resp.StatusCode)
	}
}

// --- pull tests ---

func TestPullEmpty(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	resp := pullPayload(t, srv, "token-empty")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", resp.StatusCode)
	}
}

func TestPullAfterPush(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	pushPayload(t, srv, "token-b", validBody())

	resp := pullPayload(t, srv, "token-b")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	var got map[string]any
	json.NewDecoder(resp.Body).Decode(&got)

	if got["device"] != "test-device" {
		t.Fatalf("unexpected device: %v", got["device"])
	}
	if got["data"] != validBody()["data"] {
		t.Fatalf("data mismatch: %v", got["data"])
	}
}

func TestPullNoAuth(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	req, _ := http.NewRequest("GET", srv.URL+"/pull", nil)
	resp, _ := http.DefaultClient.Do(req)

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

// --- isolation tests ---

func TestTokenIsolation(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	// Push with token-x.
	body := validBody()
	body["data"] = "dGVzdC14" // "test-x"
	pushPayload(t, srv, "token-x", body)

	// Pull with token-y must get 204.
	resp := pullPayload(t, srv, "token-y")
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("token-y should not see token-x data, got %d", resp.StatusCode)
	}
}

func TestLastWriteWinsHTTP(t *testing.T) {
	srv := newTestServer(t)
	defer srv.Close()

	first := validBody()
	first["device"] = "first-device"
	pushPayload(t, srv, "token-c", first)

	second := validBody()
	second["device"] = "second-device"
	pushPayload(t, srv, "token-c", second)

	resp := pullPayload(t, srv, "token-c")
	var got map[string]any
	json.NewDecoder(resp.Body).Decode(&got)

	if got["device"] != "second-device" {
		t.Fatalf("expected last-write-wins, got device %v", got["device"])
	}
}
