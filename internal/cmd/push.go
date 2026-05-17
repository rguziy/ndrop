package cmd

import (
	"fmt"
	"io"
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
		Use:   "push [text | file]",
		Short: "Push content to the shared buffer",
		Long: `Push content to the shared buffer.

Input sources are mutually exclusive.

Examples:
  echo "hello" | ndrop push          # stdin
  ndrop push "some text"             # inline text
  ndrop push --clipboard              # push clipboard text
  ndrop push -c "docker ps"          # command output
  ndrop push ./archive.tar.gz        # file`,
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

			// Argument: could be a file path or inline text
			case len(args) == 1:
				if info, statErr := os.Stat(args[0]); statErr == nil && !info.IsDir() {
					// It's a file.
					payload, err = os.ReadFile(args[0])
					if err != nil {
						return fmt.Errorf("read file: %w", err)
					}
					entryType = client.EntryTypeFile
					name = filepath.Base(args[0])
					mimeType = detectMIME(args[0])
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

			datab64, nonceb64, err := crypto.Encrypt(cfg.Server.Token, payload)
			if err != nil {
				return fmt.Errorf("encrypt: %w", err)
			}

			hostname, _ := os.Hostname()

			cl := client.New(cfg.Server.URL, cfg.Server.Token)
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
