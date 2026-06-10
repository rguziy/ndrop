package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rguziy/ndrop/internal/client"
	"github.com/rguziy/ndrop/internal/crypto"
	"github.com/spf13/cobra"
)

func newPushCmd(globals *globalFlags) *cobra.Command {
	var cmdFlag string // -c flag: run a command and push its output
	var flagClipboard bool

	cmd := &cobra.Command{
		Use:   "push [text | file | folder]",
		Short: "Push content to the shared buffer",
		Long: `Push content to the shared buffer.

Input sources are mutually exclusive.

Examples:
  echo "hello" | ndrop push          # stdin
  ndrop push "some text"             # inline text
  ndrop push --clipboard              # push clipboard text
  ndrop push -c "docker ps"          # command output
  ndrop push ./archive.tar.gz        # file
  ndrop push ./my-folder             # folder`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(globals)
			if err != nil {
				return err
			}

			var payload []byte
			var entryType client.EntryType
			var name, mimeType string

			if flagClipboard && (cmdFlag != "" || len(args) > 0) {
				return fmt.Errorf("--clipboard cannot be combined with other input sources")
			}

			switch {
			case flagClipboard:
				payload, err = readFromClipboard()
				if err != nil {
					return err
				}
				entryType = client.EntryTypeText
				mimeType = "text/plain"

			// -c flag: execute command and capture output
			case cmdFlag != "":
				payload, err = runCommand(cmdFlag)
				if err != nil {
					return err
				}
				entryType = client.EntryTypeText
				mimeType = "text/plain"

			// Argument: could be a file path, folder path, or inline text
			case len(args) == 1:
				if info, statErr := os.Stat(args[0]); statErr == nil {
					if info.IsDir() {
						payload, err = zipFolder(args[0])
						if err != nil {
							return err
						}
						entryType = client.EntryTypeFolder
						name = filepath.Base(filepath.Clean(args[0]))
						mimeType = "application/zip"
					} else {
						payload, err = os.ReadFile(args[0])
						if err != nil {
							return fmt.Errorf("read file: %w", err)
						}
						entryType = client.EntryTypeFile
						name = filepath.Base(args[0])
						mimeType = detectMIME(args[0])
					}
				} else {
					// Treat as inline text.
					payload = []byte(args[0])
					entryType = client.EntryTypeText
					mimeType = "text/plain"
				}

			// No args and stdin is a pipe: read stdin
			case len(args) == 0:
				stat, _ := os.Stdin.Stat()
				if (stat.Mode() & os.ModeCharDevice) != 0 {
					return fmt.Errorf("no input: provide text, a file path, --clipboard, -c flag, or pipe from stdin")
				}
				payload, err = io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("read stdin: %w", err)
				}
				entryType = client.EntryTypeText
				mimeType = "text/plain"

			default:
				return fmt.Errorf("too many arguments")
			}

			if len(payload) == 0 {
				return fmt.Errorf("empty payload: nothing to push")
			}

			datab64, nonceb64, err := crypto.Encrypt(cfg.Server.APIKey, payload)
			if err != nil {
				return fmt.Errorf("encrypt: %w", err)
			}

			hostname, _ := os.Hostname()

			cl := client.New(cfg.Server.URL, cfg.Server.APIKey)
			if err := cl.Push(client.PushRequest{
				Device: hostname,
				Type:   entryType,
				Name:   name,
				Mime:   mimeType,
				Data:   datab64,
				Nonce:  nonceb64,
			}); err != nil {
				return err
			}

			// Human-readable feedback.
			switch entryType {
			case client.EntryTypeFolder:
				fmt.Fprintf(os.Stderr, "pushed folder %q (%s, %d bytes)\n",
					name, mimeType, len(payload))
			case client.EntryTypeFile:
				fmt.Fprintf(os.Stderr, "pushed file %q (%s, %d bytes)\n",
					name, mimeType, len(payload))
			case client.EntryTypeText:
				preview := strings.TrimSpace(string(payload))
				if len(preview) > 60 {
					preview = preview[:57] + "..."
				}
				fmt.Fprintf(os.Stderr, "pushed text (%d bytes): %s\n", len(payload), preview)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&cmdFlag, "cmd", "c", "", "execute command and push its output")
	cmd.Flags().BoolVar(&flagClipboard, "clipboard", false, "push text from the system clipboard")

	return cmd
}

// zipFolder returns a zip archive containing the contents of root.
func zipFolder(root string) ([]byte, error) {
	root = filepath.Clean(root)
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	skippedSymlinks := 0

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			skippedSymlinks++
			fmt.Fprintf(os.Stderr, "warning: skipping symlink %q\n", path)
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = rel
		if d.IsDir() {
			header.Name += "/"
		} else {
			header.Method = zip.Deflate
		}

		writer, err := zw.CreateHeader(header)
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if closeErr := zw.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return nil, fmt.Errorf("zip folder: %w", err)
	}
	if skippedSymlinks > 0 {
		fmt.Fprintf(os.Stderr, "warning: skipped %d symlink(s); symlinks are not included in folder transfers\n", skippedSymlinks)
	}

	return buf.Bytes(), nil
}

// runCommand executes a shell command and returns its combined stdout.
func runCommand(cmdStr string) ([]byte, error) {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "sh"
	}
	out, err := exec.Command(shell, "-c", cmdStr).Output()
	if err != nil {
		return nil, fmt.Errorf("command %q failed: %w", cmdStr, err)
	}
	return out, nil
}

// detectMIME returns a best-guess MIME type based on the file extension.
func detectMIME(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if t := mime.TypeByExtension(ext); t != "" {
		return t
	}
	return "application/octet-stream"
}
