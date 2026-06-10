# v1.1.0 - Folder Transfer

Adds encrypted folder transfer support while keeping the server storage model simple: folders are zipped by the client, encrypted, pushed as a `folder` entry, then safely extracted by the receiving client.

## Highlights

- Added `ndrop push ./folder` support
- Added `folder` as a first-class transfer type
- Pulling a folder extracts it into the save directory
- Existing destination folders are preserved by writing to `name-1`, `name-2`, and so on
- Folder archives are protected against Zip Slip path traversal during extraction
- Symlinks are skipped during folder pushes, with warnings printed to stderr
- Updated CLI help and documentation for folder transfers

## Basic Usage

```bash
ndrop push ./project-notes
ndrop pull --save ./downloads/
```

If `./downloads/project-notes` already exists, the pulled folder is extracted to `./downloads/project-notes-1`.

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
