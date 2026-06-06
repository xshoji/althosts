// Package hostsfile writes /etc/hosts atomically.
package hostsfile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// Write atomically replaces the file at dstPath with body.
//
// The body is written to a temp file in the same directory then renamed.
// File mode is preserved from the existing file when possible; otherwise
// it falls back to 0o644.
func Write(dstPath string, body []byte) error {
	// Resolve symlinks so that the temp file lands on the same filesystem
	// (e.g. /etc/hosts on macOS is a symlink to /private/etc/hosts).
	real, err := filepath.EvalSymlinks(dstPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("eval symlinks: %w", err)
		}
		real = dstPath
	}

	dir := filepath.Dir(real)
	mode := os.FileMode(0o644)
	if st, err := os.Stat(real); err == nil {
		mode = st.Mode().Perm()
	}

	tmp, err := os.CreateTemp(dir, ".althosts.hosts.*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpPath) }

	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return fmt.Errorf("sync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, real); err != nil {
		cleanup()
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// ErrPermission indicates that the hosts file (or its directory) is not
// writable by the current user. Callers can use errors.Is to detect it and
// suggest re-running with sudo.
var ErrPermission = errors.New("permission denied writing hosts file")

// EnsureWritable verifies that we can atomically replace dstPath without
// actually modifying it. It probes by creating and removing a temp file in
// the same directory as the resolved hosts path.
//
// On permission errors the returned error wraps ErrPermission.
func EnsureWritable(dstPath string) error {
	real, err := filepath.EvalSymlinks(dstPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("eval symlinks: %w", err)
		}
		real = dstPath
	}
	dir := filepath.Dir(real)

	// Probe directory write access (needed to create the temp file).
	tmp, err := os.CreateTemp(dir, ".althosts.probe.*")
	if err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("%w: cannot write to %s (re-run with sudo)", ErrPermission, dir)
		}
		return fmt.Errorf("probe %s: %w", dir, err)
	}
	probePath := tmp.Name()
	tmp.Close()
	if err := os.Remove(probePath); err != nil {
		return fmt.Errorf("cleanup probe: %w", err)
	}

	// If the target file exists, also confirm that we have write access
	// (rename(2) needs dir write access only, but a read-only file may
	// also indicate a permission issue worth surfacing early).
	if st, err := os.Stat(real); err == nil && st.Mode().Perm()&0o200 == 0 {
		// best-effort warning - rename will still typically succeed
		// because rename requires directory perms, not file perms.
		_ = st
	}
	return nil
}

// Read reads the hosts file at path, returning an empty slice if it does not exist.
func Read(path string) ([]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return b, nil
}
