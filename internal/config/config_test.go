package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsDefaultHostsPath(t *testing.T) {
	if !IsDefaultHostsPath(DefaultHostsPath()) {
		t.Fatalf("default hosts path should be recognized as default")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "hosts")
	link := filepath.Join(dir, "hosts-link")
	if err := os.WriteFile(target, []byte("127.0.0.1 localhost\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}

	if canonicalPath(link) != canonicalPath(target) {
		t.Fatalf("canonical symlink path did not match target")
	}
	if IsDefaultHostsPath(target) {
		t.Fatalf("temporary hosts path should not be recognized as default")
	}
}
