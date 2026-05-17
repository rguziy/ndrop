package cmd

import (
	"bytes"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// writeToClipboard writes text to the OS clipboard using platform-specific tools.
// This avoids the CGO dependency of golang.design/x/clipboard.
func writeToClipboard(text []byte) error {
	switch runtime.GOOS {
	case "darwin":
		return pipeToCmd(text, "pbcopy")
	case "linux":
		// Try xclip first, then xsel, then wl-copy (Wayland).
		var failures []string
		for _, args := range [][]string{
			{"xclip", "-selection", "clipboard"},
			{"xsel", "--clipboard", "--input"},
			{"wl-copy"},
		} {
			if path, err := exec.LookPath(args[0]); err == nil {
				if err := pipeToCmd(text, path, args[1:]...); err == nil {
					return nil
				} else {
					failures = append(failures, err.Error())
				}
			}
		}
		if len(failures) > 0 {
			return fmt.Errorf("clipboard write failed: %s", strings.Join(failures, "; "))
		}
		return fmt.Errorf("no clipboard tool found: install xclip, xsel, or wl-copy")
	case "windows":
		return pipeToCmd(text, "clip")
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
}

func readFromClipboard() ([]byte, error) {
	switch runtime.GOOS {
	case "darwin":
		return runClipboardOutput("pbpaste")
	case "linux":
		var failures []string
		for _, args := range [][]string{
			{"xclip", "-selection", "clipboard", "-o"},
			{"xsel", "--clipboard", "--output"},
			{"wl-paste"},
		} {
			if path, err := exec.LookPath(args[0]); err == nil {
				out, err := runClipboardOutput(path, args[1:]...)
				if err == nil {
					return out, nil
				}
				failures = append(failures, err.Error())
			}
		}
		if len(failures) > 0 {
			return nil, fmt.Errorf("clipboard read failed: %s", strings.Join(failures, "; "))
		}
		return nil, fmt.Errorf("no clipboard tool found: install xclip, xsel, or wl-paste")
	case "windows":
		return runClipboardOutput("powershell", "-NoProfile", "-Command", "Get-Clipboard -Raw")
	default:
		return nil, fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
}

// pipeToCmd runs a command with text piped to its stdin.
func pipeToCmd(text []byte, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("clipboard pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", name, err)
	}
	if _, err := stdin.Write(text); err != nil {
		return fmt.Errorf("write to %s: %w", name, err)
	}
	stdin.Close()
	if err := cmd.Wait(); err != nil {
		return commandError(name, err, stderr.String())
	}
	return nil
}

func runClipboardOutput(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, commandError(name, err, stderr.String())
	}
	return out, nil
}

func commandError(name string, err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return fmt.Errorf("%s: %w", name, err)
	}
	return fmt.Errorf("%s: %w: %s", name, err, stderr)
}
