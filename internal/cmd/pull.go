package cmd

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
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
  ndrop pull --save ./downloads/     # file/folder → save to directory
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

			cl := client.New(cfg.Server.URL, cfg.Server.APIKey, cfg.TimeoutSeconds)
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
				return savePayload(plaintext, resp, expandHome(flagSaveDir))

			default:
				// Default behaviour depends on type.
				if resp.Type == client.EntryTypeFile || resp.Type == client.EntryTypeFolder {
					// For files and folders default to saving in the configured dir.
					saveDir := cfg.Pull.DefaultSaveDir
					if saveDir == "" {
						saveDir = "."
					}
					return savePayload(plaintext, resp, expandHome(saveDir))
				}
				// Text → stdout.
				_, err = fmt.Fprint(os.Stdout, string(plaintext))
				return err
			}
		},
	}

	cmd.Flags().BoolVar(&flagClipboard, "clipboard", false, "write text to system clipboard")
	cmd.Flags().StringVar(&flagSaveDir, "save", "", "save file or folder to this directory")
	cmd.Flags().BoolVar(&flagStdout, "stdout", false, "write raw bytes to stdout")

	return cmd
}

func savePayload(plaintext []byte, resp *client.PullResponse, dir string) error {
	if resp.Type == client.EntryTypeFolder {
		return saveFolder(plaintext, resp, dir)
	}
	return saveFile(plaintext, resp, dir)
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
	name = safeOutputName(name, "ndrop-download")

	dest := filepath.Join(dir, name)
	if err := os.WriteFile(dest, plaintext, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	fmt.Fprintf(os.Stderr, "saved %q (%d bytes)\n", dest, len(plaintext))
	return nil
}

// saveFolder extracts a zip archive into a unique dir/<name> directory.
func saveFolder(plaintext []byte, resp *client.PullResponse, dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dir, err)
	}

	name := resp.Name
	if name == "" {
		name = "ndrop-folder"
	}
	name = safeOutputName(name, "ndrop-folder")

	dest, err := uniquePath(filepath.Join(dir, name))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return fmt.Errorf("create directory %q: %w", dest, err)
	}

	if err := unzipSecure(plaintext, dest); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "saved folder %q (%d bytes)\n", dest, len(plaintext))
	return nil
}

func uniquePath(base string) (string, error) {
	if _, err := os.Stat(base); err != nil {
		if os.IsNotExist(err) {
			return base, nil
		}
		return "", fmt.Errorf("inspect destination %q: %w", base, err)
	}

	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if _, err := os.Stat(candidate); err != nil {
			if os.IsNotExist(err) {
				return candidate, nil
			}
			return "", fmt.Errorf("inspect destination %q: %w", candidate, err)
		}
	}
}

func safeOutputName(name, fallback string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = filepath.Base(filepath.Clean(filepath.FromSlash(name)))
	if name == "." || name == string(os.PathSeparator) || name == "" {
		return fallback
	}
	return name
}

func unzipSecure(data []byte, dest string) error {
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return fmt.Errorf("read folder archive: %w", err)
	}

	destClean, err := filepath.Abs(filepath.Clean(dest))
	if err != nil {
		return fmt.Errorf("resolve destination: %w", err)
	}

	for _, file := range reader.File {
		target, err := secureZipTarget(destClean, file.Name)
		if err != nil {
			return err
		}

		mode := file.Mode()
		if mode&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe archive entry %q: symlinks are not allowed", file.Name)
		}

		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create directory %q: %w", target, err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directory %q: %w", filepath.Dir(target), err)
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("open archive entry %q: %w", file.Name, err)
		}

		perm := mode.Perm()
		if perm == 0 {
			perm = 0o644
		}
		dst, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
		if err != nil {
			src.Close()
			return fmt.Errorf("write archive entry %q: %w", file.Name, err)
		}

		_, copyErr := io.Copy(dst, src)
		closeSrcErr := src.Close()
		closeDstErr := dst.Close()
		if copyErr != nil {
			return fmt.Errorf("write archive entry %q: %w", file.Name, copyErr)
		}
		if closeSrcErr != nil {
			return fmt.Errorf("close archive entry %q: %w", file.Name, closeSrcErr)
		}
		if closeDstErr != nil {
			return fmt.Errorf("close file %q: %w", target, closeDstErr)
		}
	}

	return nil
}

func secureZipTarget(dest, name string) (string, error) {
	name = strings.ReplaceAll(name, "\\", "/")
	if name == "" || filepath.IsAbs(name) {
		return "", fmt.Errorf("unsafe archive entry %q", name)
	}

	cleanName := filepath.Clean(filepath.FromSlash(name))
	if cleanName == "." || cleanName == ".." || strings.HasPrefix(cleanName, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe archive entry %q", name)
	}

	target := filepath.Join(dest, cleanName)
	targetClean, err := filepath.Abs(filepath.Clean(target))
	if err != nil {
		return "", fmt.Errorf("resolve archive entry %q: %w", name, err)
	}

	if targetClean != dest && !strings.HasPrefix(targetClean, dest+string(os.PathSeparator)) {
		return "", fmt.Errorf("unsafe archive entry %q", name)
	}

	return targetClean, nil
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
