package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultTimeout = 30 * time.Second

// EntryType mirrors the server-side type.
type EntryType string

const (
	EntryTypeText EntryType = "text"
	EntryTypeFile EntryType = "file"
)

// PushRequest is the JSON body sent to POST /push.
type PushRequest struct {
	Device string    `json:"device"`
	Type   EntryType `json:"type"`
	Name   string    `json:"name"`
	Mime   string    `json:"mime"`
	Data   string    `json:"data"`
	Nonce  string    `json:"nonce"`
}

// PullResponse is the JSON body received from GET /pull.
type PullResponse struct {
	Device string    `json:"device"`
	Type   EntryType `json:"type"`
	Name   string    `json:"name"`
	Mime   string    `json:"mime"`
	Data   string    `json:"data"`
	Nonce  string    `json:"nonce"`
}

// Client handles HTTP communication with the ndrop server.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New creates a Client for the given server URL and auth token.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// Push sends a PushRequest to the server.
func (c *Client) Push(req PushRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.baseURL+"/push", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return fmt.Errorf("push: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		return nil
	case http.StatusUnauthorized:
		return fmt.Errorf("unauthorized: check your token")
	case http.StatusRequestEntityTooLarge:
		return fmt.Errorf("payload too large: exceeds server limit")
	default:
		msg, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}
}

// Pull retrieves the current buffer from the server.
// Returns (nil, nil) when the buffer is empty or expired (204 No Content).
func (c *Client) Pull() (*PullResponse, error) {
	httpReq, err := http.NewRequest("GET", c.baseURL+"/pull", nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.token)

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("pull: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil, nil
	case http.StatusUnauthorized:
		return nil, fmt.Errorf("unauthorized: check your token")
	case http.StatusOK:
		// continue below
	default:
		msg, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("server returned %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var pr PullResponse
	if err := json.NewDecoder(resp.Body).Decode(&pr); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &pr, nil
}
