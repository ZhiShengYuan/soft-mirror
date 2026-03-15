package storage

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestPathConfinement(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	traversals := [][]string{
		{"../evil", "v1.0.0", "linux", "amd64"},
		{"prog", "../../../etc", "linux", "amd64"},
		{"prog", "v1.0.0", "../evil", "amd64"},
		{"prog", "v1.0.0", "linux", "../evil"},
		{"prog", "v1.0.0/../../evil", "linux", "amd64"},
	}

	for _, parts := range traversals {
		prog, ver, osName, arch := parts[0], parts[1], parts[2], parts[3]
		_, err := store.GetBinaryPath(prog, ver, osName, arch)
		if err == nil {
			t.Errorf("expected error for traversal %v, got nil", parts)
		}
		// Should not have created any files outside root
		err2 := store.PutBinary(prog, ver, osName, arch, "", bytes.NewReader([]byte("test")), 1024)
		if err2 == nil {
			// If put somehow succeeded (shouldn't), verify file is inside root
			absPath, _ := filepath.Abs(filepath.Join(root, prog, ver, osName, arch))
			if len(absPath) < len(root) {
				t.Errorf("file created outside root for traversal %v", parts)
			}
		}
	}
}

func TestPutAndGet(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	content := []byte("hello binary world")
	err = store.PutBinary("myapp", "v1.0.0", "linux", "amd64", "", bytes.NewReader(content), 1024*1024)
	if err != nil {
		t.Fatalf("PutBinary: %v", err)
	}

	path, err := store.GetBinaryPath("myapp", "v1.0.0", "linux", "amd64")
	if err != nil {
		t.Fatalf("GetBinaryPath: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Errorf("content mismatch: got %q, want %q", got, content)
	}

	// Check file permissions
	info, _ := os.Stat(path)
	if info.Mode() != 0644 {
		t.Errorf("file mode: got %v, want 0644", info.Mode())
	}
}

func TestListVersions(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	content := []byte("bin")
	for _, v := range []string{"v1.0.0", "v1.1.0", "v2.0.0"} {
		if err := store.PutBinary("app", v, "linux", "amd64", "", bytes.NewReader(content), 1024); err != nil {
			t.Fatalf("PutBinary %s: %v", v, err)
		}
	}

	versions, err := store.ListVersions("app")
	if err != nil {
		t.Fatalf("ListVersions: %v", err)
	}
	if len(versions) != 3 {
		t.Errorf("expected 3 versions, got %d", len(versions))
	}
}

func TestDeleteBinary(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	content := []byte("bin")
	if err := store.PutBinary("app", "v1.0.0", "linux", "amd64", "", bytes.NewReader(content), 1024); err != nil {
		t.Fatalf("PutBinary: %v", err)
	}

	if err := store.DeleteBinary("app", "v1.0.0", "linux", "amd64"); err != nil {
		t.Fatalf("DeleteBinary: %v", err)
	}

	_, err = store.GetBinaryPath("app", "v1.0.0", "linux", "amd64")
	if err == nil {
		t.Error("expected error after deletion, got nil")
	}
}

func TestMaxSize(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	big := bytes.Repeat([]byte("x"), 1000)
	err = store.PutBinary("app", "v1.0.0", "linux", "amd64", "", bytes.NewReader(big), 100)
	if err == nil {
		t.Error("expected error for oversized file, got nil")
	}
}

func TestWindowsFilename(t *testing.T) {
	root := t.TempDir()
	store, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	content := []byte("exe content")
	if err := store.PutBinary("app", "v1.0.0", "windows", "amd64", "", bytes.NewReader(content), 1024); err != nil {
		t.Fatalf("PutBinary: %v", err)
	}

	path, err := store.GetBinaryPath("app", "v1.0.0", "windows", "amd64")
	if err != nil {
		t.Fatalf("GetBinaryPath: %v", err)
	}
	if filepath.Base(path) != "app.exe" {
		t.Errorf("expected app.exe, got %s", filepath.Base(path))
	}
}
