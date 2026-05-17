package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rguziy/ndrop/internal/client"
	"github.com/rguziy/ndrop/internal/crypto"
	"github.com/spf13/cobra"
)

func newPullCmd(globals *globalFlags) *cobra.Command {
	var (
		flagClipboard bool
		flagSaveDir   string
		flagStdout    bool
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Pull content from the shared buffer",
		Long: `Pull content from the shared buffer.

Output options are mutually exclusive.

Examples:
  ndrop pull                         # text → stdout (default)
  ndrop pull --clipboard             # text → system clipboard
  ndrop pull --save ./downloads/     # file → save to directory
  ndrop pull --stdout                # raw bytes → stdout (pipe-friendly)`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(globals)
			if err != nil {
				return err
			}

			// Validate flag combinations.
			flagCount := 0
			for _, f := range []bool{flagClipboard, flagSaveDir != "", flagStdout} {
				if f {
					flagCount++
				}
			}
			if flagCount > 1 {
				return fmt.Errorf("--clipboard, --save, and --stdout are mutually exclusive")
			}

			cl := client.New(cfg.Server.URL, cfg.Server.APIKey)
			resp, err := cl.Pull()
			if err != nil {
				return err
			}

			if resp == nil {
				fmt.Fprintln(os.Stderr, "buffer is empty or expired")
				return nil
			}

			// Decrypt.
			plaintext, err := crypto.Decrypt(cfg.Server.APIKey, resp.Data, resp.Nonce)
			if err != nil {
				return fmt.Errorf("decrypt: %w", err)
			}

			// Route output based on flags.
			switch {
			case flagStdout:
				_, err = os.Stdout.Write(plaintext)
				return err

			case flagClipboard:
				return writeToClipboard(plaintext)

			case flagSaveDir != "":
				return saveFile(plaintext, resp, expandHome(flagSaveDir))

			default:
				// Default behaviour depends on type.
				if resp.Type == client.EntryTypeFile {
					// For files default to saving in the configured dir.
					saveDir := cfg.Pull.DefaultSaveDir
					if saveDir == "" {
						saveDir = "."
					}
					return saveFile(plaintext, resp, expandHome(saveDir))
				}
				// Text → stdout.
				_, err = fmt.Fprint(os.Stdout, string(plaintext))
				return err
			}
		},
	}

	cmd.Flags().BoolVar(&flagClipboard, "clipboard", false, "write text to system clipboard")
	cmd.Flags().StringVar(&flagSaveDir, "save", "", "save file to this directory")
	cmd.Flags().BoolVar(&flagStdout, "stdout", false, "write raw bytes to stdout")

	return cmd
}

// saveFile writes plaintext to dir/<name>, creating the directory if needed.
func saveFile(plaintext []byte, resp *client.PullResponse, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}

	name := resp.Name
	if name == "" {
		name = "ndrop-download"
	}

	dest := filepath.Join(dir, name)
	if err := os.WriteFile(dest, plaintext, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "saved %q (%d bytes)\n", dest, len(plaintext))
	return nil
}

// expandHome replaces a leading ~ with the user home directory.
func expandHome(path string) string {
	if !strings.HasPrefix(path, "~") {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[1:])
}
