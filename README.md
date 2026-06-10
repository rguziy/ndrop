# ndrop

🚀 `ndrop` is a self-hosted data transfer utility for text, files, and folders, built in Go.
It uses a self-hosted HTTP server accessible locally or remotely, with end-to-end encryption so the server stores only ciphertext.

Similar in spirit to AirDrop-style sharing, but uses HTTP and E2E encryption instead of OS-specific peer-to-peer discovery.

## Use cases

- Run `ndropd` on your laptop for local quick transfers.
- Deploy `ndropd` on a VPS or remote machine for internet-based drops.
- Use `ndrop` from any network-capable client to push/pull text, files, and folders over HTTPS.

## Features

- 🔐 End-to-end encrypted content transfer
- 📤 Push text, files, folders, or clipboard contents from the CLI
- 📥 Pull shared content to stdout, system clipboard, saved file, or extracted folder
- 🧩 API-key-based isolation per shared buffer
- ⚙️ Configurable with TOML, CLI flags, and environment variables
- 🖥️ `ndropd` CLI with `init`, `start`, `stop`, and `help`

## Quick Start

### 1. Initialize client config

```bash
./ndrop init
```

This creates `~/.config/ndrop/ndrop.toml` with:

```toml
[server]
url = "http://localhost:8080"
api_key = "your-api-key"

[pull]
default_save_dir = "~/Downloads"
```

### 2. Initialize server config

```bash
./ndropd init
```

This creates `~/.config/ndrop/ndropd.toml` with:

```toml
port = "8080"
max_size_mb = 10
ttl_hours = 1
allow_any_api_key = true
allowed_api_keys = []
```

### 3. Start the server

```bash
./ndropd start
```

### 4. Push and pull

```bash
# Push inline text
./ndrop push "Hello World!"

# Push a file
./ndrop push ./archive.tar.gz

# Push a folder
./ndrop push ./my-folder

# Push clipboard contents
./ndrop push --clipboard

# Push command output
./ndrop push -c "date"

# Pull text to stdout (default)
./ndrop pull

# Pull text to system clipboard
./ndrop pull --clipboard

# Pull a file or folder to a directory
./ndrop pull --save ./downloads/

# Pull raw bytes to stdout
./ndrop pull --stdout
```

## CLI Usage

### Client commands

- `ndrop init` — create default client config
- `ndrop push [text|file|folder]` — push text, a file, or a folder
- `ndrop pull` — pull the latest entry

### Push options

- `--clipboard` — push text from the system clipboard
- `-c, --cmd <command>` — execute a command and push its output

### Pull options

- `--clipboard` — write text to the system clipboard
- `--save <dir>` — save a file or extract a folder to a directory
- `--stdout` — write raw bytes to stdout

### Server commands

- `ndropd init` — create server config
- `ndropd start` — run server in the foreground
- `ndropd stop` — stop the running server
- `ndropd help` — show help

## How encryption works

`ndrop` uses the configured API key for two separate purposes:

- It derives a stable `bucket_id`, which selects the shared buffer on the server.
- It derives an AES-256-GCM encryption key, which encrypts and decrypts the payload.

When you run `ndrop push`, the client encrypts the text, file, or zipped folder before uploading it. The server stores only:

- the derived `bucket_id`
- encrypted `data`
- the `nonce` needed to decrypt that encrypted payload
- metadata such as device name, MIME type, and payload name

The `nonce` is a random 12-byte value generated for every push. It is not secret, but it must be unique for each encryption with the same API key. The pull client receives `data` and `nonce`, then decrypts locally using the same API key.

The raw API key is sent to the server in the `Authorization: Bearer <api-key>` header so the server can route the request and optionally enforce `allowed_api_keys`. The server does not store the raw API key, but deployments should still use HTTPS when the server is accessed over a network.

## Configuration

### Client config

File: `~/.config/ndrop/ndrop.toml`

```toml
[server]
url = "http://localhost:8080"
api_key = "my-secret-api-key"

[pull]
default_save_dir = "~/Downloads"
```

### Server config

File: `~/.config/ndrop/ndropd.toml`

```toml
port = "8080"
max_size_mb = 10
ttl_hours = 1
allow_any_api_key = true
allowed_api_keys = []
```

Set `allow_any_api_key = false` and list accepted keys to reject unknown clients:

```toml
allow_any_api_key = false
allowed_api_keys = ["laptop-key", "phone-key"]
```

### Environment variables

Client config can be overridden with:

- `NDROP_URL`
- `NDROP_API_KEY`

Server config can be overridden with:

- `PORT`
- `MAX_SIZE_MB`
- `TTL_HOURS`
- `ALLOW_ANY_API_KEY`
- `ALLOWED_API_KEYS`

## Docker

A Docker Compose setup is included under `docker/docker-compose.yml`.
Use `docker/Dockerfile` to build the image and `docker/build.sh` to run the workflow.

From the repository root:

```bash
./docker/build.sh build
```

This builds the `ndropd` image only.

```bash
./docker/build.sh up
```

This rebuilds the image and starts the `ndropd` service in detached mode.

The compose service publishes port `8080` and sets:

- `PORT=8080`
- `MAX_SIZE_MB=10`
- `TTL_HOURS=1`
- `ALLOW_ANY_API_KEY=true`

## Linux systemd

A sample systemd unit is available at `deploy/systemd/ndropd.service`.

It assumes:

- `ndropd` is installed at `/usr/local/bin/ndropd`
- a dedicated Linux user and group named `ndrop` exist
- server state is stored under `/var/lib/ndrop`

Example setup:

```bash
sudo useradd --system --home-dir /var/lib/ndrop --create-home --shell /usr/sbin/nologin ndrop
sudo install -m 0755 ndropd /usr/local/bin/ndropd
sudo install -m 0644 deploy/systemd/ndropd.service /etc/systemd/system/ndropd.service
sudo systemctl daemon-reload
sudo systemctl enable --now ndropd
```

Before starting the service on a public host, edit `/etc/systemd/system/ndropd.service` and replace:

```ini
Environment=ALLOWED_API_KEYS=change-me
```

For multiple API keys, use a comma-separated value:

```ini
Environment=ALLOWED_API_KEYS=laptop-key,phone-key
```

## Build

The canonical project version is defined in `internal/version/version.go`.
Plain `go build`, Docker builds, and Make-based release builds use that value.
Update it there before cutting a new release.

Both binaries can print their version:

```bash
ndrop --version
ndropd --version
```

### Local Go build

```bash
go build -o ndrop ./cmd/ndrop
go build -o ndropd ./cmd/ndropd
```

### Cross-platform build with Make

The repository includes a `Makefile` with platform targets.
Built artifacts are packaged as ZIP archives in the `build/` directory.

```bash
make linux-amd64
make windows-amd64
make darwin-amd64
make darwin-arm64
```

To build all supported targets:

```bash
make release
```

To remove generated build artifacts:

```bash
make clean
```

Supported targets:

- `linux-amd64`
- `linux-armv5`
- `linux-armv6`
- `linux-armv7`
- `windows-amd64`
- `darwin-amd64`
- `darwin-arm64`

Each target builds both `ndrop` and `ndropd`, then creates a zip package like:

- `build/ndrop-linux-amd64-1.1.0.zip`
- `build/ndrop-windows-amd64-1.1.0.zip`
- `build/ndrop-darwin-arm64-1.1.0.zip`

## License

This project is licensed under the MIT License. See `LICENSE`.
