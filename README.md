# ndrop

🚀 `ndrop` is a self-hosted data transfer utility for text and files, built in Go.
It uses a self-hosted HTTP server accessible locally or remotely, with end-to-end encryption so the server stores only ciphertext.

Similar in spirit to AirDrop-style sharing, but uses HTTP and E2E encryption instead of OS-specific peer-to-peer discovery.

## Use cases

- Run `ndropd` on your laptop for local quick transfers.
- Deploy `ndropd` on a VPS or remote machine for internet-based drops.
- Use `ndrop` from any network-capable client to push/pull text and files over HTTPS.

## Features

- 🔐 End-to-end encrypted content transfer
- 📤 Push text, files, or clipboard contents from the CLI
- 📥 Pull shared content to stdout, system clipboard, or saved file
- 🧩 Token-based isolation per shared buffer
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
token = "your-token"

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

# Push clipboard contents
./ndrop push --clipboard

# Push command output
./ndrop push -c "date"

# Pull text to stdout (default)
./ndrop pull

# Pull text to system clipboard
./ndrop pull --clipboard

# Pull a file to a directory
./ndrop pull --save ./downloads/

# Pull raw bytes to stdout
./ndrop pull --stdout
```

## CLI Usage

### Client commands

- `ndrop init` — create default client config
- `ndrop push [text|file]` — push text or a file
- `ndrop pull` — pull the latest entry

### Push options

- `--clipboard` — push text from the system clipboard
- `-c, --cmd <command>` — execute a command and push its output

### Pull options

- `--clipboard` — write text to the system clipboard
- `--save <dir>` — save a file to a directory
- `--stdout` — write raw bytes to stdout

### Server commands

- `ndropd init` — create server config
- `ndropd start` — run server in the foreground
- `ndropd stop` — stop the running server
- `ndropd help` — show help

## Configuration

### Client config

File: `~/.config/ndrop/ndrop.toml`

```toml
[server]
url = "http://localhost:8080"
token = "my-secret-token"

[pull]
default_save_dir = "~/Downloads"
```

### Server config

File: `~/.config/ndrop/ndropd.toml`

```toml
port = "8080"
max_size_mb = 10
ttl_hours = 1
```

### Environment variables

Client config can be overridden with:

- `NDROP_URL`
- `NDROP_TOKEN`

Server config can be overridden with:

- `PORT`
- `MAX_SIZE_MB`
- `TTL_HOURS`

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

## Build

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

- `build/ndrop-linux-amd64-1.0.0.zip`
- `build/ndrop-windows-amd64-1.0.0.zip`
- `build/ndrop-darwin-arm64-1.0.0.zip`

## License

This project is licensed under the MIT License. See `LICENSE`.
