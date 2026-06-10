# v1.2.1 - Modern UI & Dropzone UX

## Highlights

- **Complete Web UI Redesign**: Switched from a basic dark theme to a premium light cyberpunk/minimalist style featuring smooth gradients, glassmorphic panel blending (`backdrop-filter`), and refined typography.
- **Interactive Drag & Drop**: Implemented a fully functional dropzone for files and folders, eliminating the raw native browser file picker in favor of a modern drag-and-drop container.
- **On-Page File Verification**: Fixed a core JS array-buffer parsing issue. Selected file names and sizes are now accurately rendered right on the screen inside the dropzone layout, rather than hidden in browser tooltips.
- **Enhanced UX States**:
  - Added a dedicated "Cancel" action directly on the page to unstage a selected file before pushing.
  - Implemented dynamic field resetting to automatically wipe the active inputs and memory buffers upon successful text or file pushes.
  - Replaced text-based password toggles with dynamic, context-aware SVG icons (`eye` / `eye-off`) that match the unified icon sprite system.
- **Responsive Layout**: Re-engineered the actions grid to feature an adaptive split (`1fr 54px 54px`) on desktop, seamlessly collapsing into full-width mobile stacks for single-hand use.

# v1.2.0 - Web UI

## Highlights

- Added an embedded `ndropd` web UI served at `/`
- Web UI uses API key authentication and browser WebCrypto for local encryption/decryption
- Web UI can pull text, files, and folders; folders download as zip files
- Web UI can push text and files
- Native mobile apps are intentionally out of scope for now

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
