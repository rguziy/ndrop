# Technical Requirements Document: ndrop

**Version:** 0.1.1  
**Status:** Draft  
**Date:** 2026-05-16

---

## 1. Overview

`ndrop` is a cross-platform CLI utility for transferring text and files between devices via a self-hosted HTTP server. It is designed to provide a network-transparent solution that works across firewalls, networks, and operating systems, whether the server runs locally or on a remote VPS.

---

## 2. Goals

- Transfer text and files between arbitrary devices over HTTPS
- No dependency on cloud providers (Dropbox, Google Drive, etc.)
- Self-hosted server deployable via Docker on local machines, home labs, or VPS instances
- Simple token-based isolation: one token = one shared buffer
- End-to-end encryption: server never sees plaintext
- Minimal dependencies, pure Go where possible

## 3. Non-Goals

- Real-time sync / watch mode
- Clipboard history
- Multi-user ACL or per-device tokens
- Browser or GUI clients
- OS clipboard integration for files (text only for `--clipboard`)

---

## 4. Architecture

```
[Device A]                [ndropd]              [Device B]
  CLI client  -- HTTPS -->  in-memory store  -- HTTPS -->  CLI client
  push text                 bucket per token               pull text
  push file                 TTL-based expiry               pull file → save
```

### 4.1 Server

- Single stateless HTTP server written in Go
- In-memory key-value store: `map[bucket_id] → Entry`
- One entry per bucket (last-write-wins)
- Background goroutine purges expired entries
- No database, no external dependencies

### 4.2 Client

- Single binary CLI (`ndrop`)
- Reads config from `~/.config/ndrop/ndrop.toml`
- Config values overridable via CLI flags and environment variables
- Includes `init`, `push`, and `pull` commands

### 4.3 Isolation Model

Each token maps to an isolated buffer on the server:

```
bucket_id = HKDF(token, "ndrop-bucket")   → used as map key
enc_key   = HKDF(token, "ndrop-encrypt")  → used for AES-256-GCM
```

The server stores only `bucket_id` (a derived value) and the encrypted ciphertext. The plaintext token and decrypted payload are never visible to the server.

---

## 5. API Specification

### 5.1 Authentication

All endpoints require:

```
Authorization: Bearer <token>
```

Returns `401 Unauthorized` if the header is missing or malformed.

---

### 5.2 `POST /push`

Upload content to the shared buffer.

**Request**

```
Content-Type: application/json
```

```json
{
  "device": "synology",
  "type":   "text",
  "name":   "",
  "mime":   "text/plain",
  "data":   "<base64(AES-256-GCM(payload))>",
  "nonce":  "<base64(12-byte random nonce)>"
}
```

| Field    | Required | Description                                      |
|----------|----------|--------------------------------------------------|
| `device` | yes      | Human-readable source device name                |
| `type`   | yes      | `text` or `file`                                 |
| `name`   | no       | Original filename; required when `type = file`   |
| `mime`   | yes      | MIME type of the original payload                |
| `data`   | yes      | Base64-encoded AES-256-GCM ciphertext            |
| `nonce`  | yes      | Base64-encoded 12-byte GCM nonce                 |

**Responses**

| Code | Meaning                                          |
|------|--------------------------------------------------|
| 200  | OK — entry stored                                |
| 400  | Bad Request — malformed JSON or missing fields   |
| 401  | Unauthorized                                     |
| 413  | Payload Too Large — exceeds `max_size_mb`        |

---

### 5.3 `GET /pull`

Retrieve the current buffer contents.

**Request**

No body. Authorization header required.

**Responses**

| Code | Meaning                                          |
|------|--------------------------------------------------|
| 200  | OK — returns entry JSON (same schema as push)    |
| 204  | No Content — buffer is empty or TTL expired      |
| 401  | Unauthorized                                     |

---

## 6. Encryption

### 6.1 Key Derivation

Both the bucket identifier and encryption key are derived from the token using HKDF-SHA256 (RFC 5869):

```
bucket_id = HKDF-SHA256(ikm=token, salt=nil, info="ndrop-bucket",  len=32)
enc_key   = HKDF-SHA256(ikm=token, salt=nil, info="ndrop-encrypt", len=32)
```

### 6.2 Encryption Scheme

- Algorithm: **AES-256-GCM**
- Nonce: 12 bytes, randomly generated per push operation
- Tag size: 16 bytes (GCM default)

```
ciphertext = AES-256-GCM.Seal(key=enc_key, nonce=nonce, plaintext=payload)
data_field = base64(ciphertext)
nonce_field = base64(nonce)
```

### 6.3 Security Properties

- The server cannot decrypt stored data (no access to token or enc_key)
- The server stores only `bucket_id` as a map key, not the raw token
- Replay protection is not in scope for v0.1 (TTL provides partial mitigation)

---

## 7. Data Model

### 7.1 Server Entry

```go
type Entry struct {
    Device    string
    Type      string    // "text" | "file"
    Name      string    // original filename, empty for text
    Mime      string
    Data      string    // base64 ciphertext
    Nonce     string    // base64 nonce
    ExpiresAt time.Time
}
```

### 7.2 Storage

```go
type Store interface {
    Set(bucketID string, entry Entry)
    Get(bucketID string) (Entry, bool)
    Delete(bucketID string)
    Purge()  // remove all expired entries
}
```

Initial implementation: `MemoryStore` backed by `sync.RWMutex` + `map[string]Entry`.

---

## 8. CLI Reference

### 8.1 Push

```bash
# Text from stdin
echo "hello world" | ndrop push

# Text from argument
ndrop push "some text"

# Text from system clipboard
ndrop push --clipboard

# Text from command output
ndrop push -c "docker ps"

# File
ndrop push ./archive.tar.gz
```

### 8.2 Pull

```bash
# Print text to stdout (default)
ndrop pull

# Write text to system clipboard
ndrop pull --clipboard

# Save file to directory
ndrop pull --save ./downloads/

# Write raw bytes to stdout (pipe-friendly)
ndrop pull --stdout
```

### 8.3 Client Init

```bash
ndrop init
```

Creates `~/.config/ndrop/ndrop.toml` with default local server settings:

```toml
[server]
url = "http://localhost:8080"
token = "your-token"

[pull]
default_save_dir = "~/Downloads"
```

### 8.4 Server CLI

```bash
ndropd init    # create ~/.config/ndrop/ndropd.toml
ndropd start   # run server in foreground
ndropd stop    # stop running server
ndropd help    # show usage
```

If `ndropd` is run with no arguments, it prints help.

### 8.5 Global Client Flags

```bash
--config   path to config file (default: ~/.config/ndrop/ndrop.toml)
--server   server URL (overrides config and NDROP_URL)
--token    auth token (overrides config and NDROP_TOKEN)
```

---

## 9. Configuration

### 9.1 Client — `~/.config/ndrop/ndrop.toml`

```toml
[server]
url   = "http://localhost:8080"
token = "my-secret-token"

[pull]
default_save_dir = "~/Downloads"
```

### 9.2 Server — `~/.config/ndrop/ndropd.toml`

```toml
port = "8080"
max_size_mb = 10
ttl_hours = 1
```

### 9.3 Server Environment Overrides

Environment variables take priority over `ndropd.toml`:

- `PORT`
- `MAX_SIZE_MB`
- `TTL_HOURS`
