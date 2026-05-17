# v1.0.0 - Initial Release

First stable release of `ndrop`, a self-hosted CLI tool for encrypted text and file drops between devices.

## Highlights

- End-to-end encrypted text and file transfer
- Self-hosted `ndropd` HTTP server
- CLI client with `push`, `pull`, and `init`
- API-key-based shared buffer isolation
- Optional server-side API key allowlist
- In-memory server storage with TTL-based expiry
- Clipboard support for text
- Docker Compose deployment support
- Sample Linux systemd unit
- Cross-platform release builds via Makefile

## Binaries

This release includes builds for:

- Linux amd64
- Linux ARM v5/v6/v7
- Windows amd64
- macOS amd64
- macOS arm64

## Basic Usage

```bash
ndrop init
ndropd init
ndropd start

ndrop push "hello from one device"
ndrop pull
```

## Server Access Control

By default, `ndropd` can accept any API key:

```toml
allow_any_api_key = true
allowed_api_keys = []
```

To restrict access:

```toml
allow_any_api_key = false
allowed_api_keys = ["laptop-key", "phone-key"]
```

## Version

```bash
ndrop --version
ndropd --version
```
