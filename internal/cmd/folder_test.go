package cmd

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/rguziy/ndrop/internal/client"
)

func TestZipFolderSkipsSymlinks(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if err := os.Symlink("file.txt", filepath.Join(root, "link.txt")); err != nil {
		t.Skipf("symlink not available: %v", err)
	}

	data, err := zipFolder(root)
	if err != nil {
		t.Fatalf("zip folder: %v", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		t.Fatalf("read zip: %v", err)
	}

	entries := map[string]bool{}
	for _, f := range zr.File {
		entries[f.Name] = true
	}
	if !entries["file.txt"] {
		t.Fatalf("expected file.txt in archive")
	}
	if entries["link.txt"] {
		t.Fatalf("did not expect symlink in archive")
	}
}

func TestSaveFolderUsesUniqueDestination(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "bundle"), 0o755); err != nil {
		t.Fatalf("create existing dir: %v", err)
	}

	data := testZip(t, map[string]string{"nested/file.txt": "hello"})
	resp := &client.PullResponse{Type: client.EntryTypeFolder, Name: "bundle"}

	if err := saveFolder(data, resp, dir); err != nil {
		t.Fatalf("save folder: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "bundle-1", "nested", "file.txt"))
	if err != nil {
		t.Fatalf("read extracted file: %v", err)
	}
	if string(got) != "hello" {
		t.Fatalf("unexpected extracted content: %q", got)
	}
}

func TestSaveFolderSanitizesName(t *testing.T) {
	dir := t.TempDir()
	data := testZip(t, map[string]string{"file.txt": "hello"})
	resp := &client.PullResponse{Type: client.EntryTypeFolder, Name: "../bundle"}

	if err := saveFolder(data, resp, dir); err != nil {
		t.Fatalf("save folder: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "bundle", "file.txt")); err != nil {
		t.Fatalf("expected sanitized folder path: %v", err)
	}
}

func TestUnzipSecureRejectsZipSlip(t *testing.T) {
	dir := t.TempDir()
	data := testZip(t, map[string]string{"../evil.txt": "oops"})

	if err := unzipSecure(data, dir); err == nil {
		t.Fatalf("expected zip slip archive to be rejected")
	}
	if _, err := os.Stat(filepath.Join(dir, "..", "evil.txt")); err == nil {
		t.Fatalf("zip slip wrote outside destination")
	}
}

func testZip(t *testing.T, files map[string]string) []byte {
	t.Helper()

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buf.Bytes()
}
