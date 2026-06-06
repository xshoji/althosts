// Package home resolves the althosts home directory and exposes the
// directory layout described in docs/althosts-design.md.
package home

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// EnvVar is the environment variable used to override the althosts home directory.
const EnvVar = "ALTHOSTS_HOME"

// DefaultDirName is the default directory under the user's home.
const DefaultDirName = ".althosts"

// Home represents the resolved althosts home directory.
type Home struct {
	Root string
}

// Resolve resolves the althosts home directory using the following priority:
//  1. flag value (e.g. --home <path>)
//  2. ALTHOSTS_HOME environment variable
//  3. ~/.althosts
func Resolve(flagValue string) (*Home, error) {
	root := flagValue
	if root == "" {
		root = os.Getenv(EnvVar)
	}
	if root == "" {
		ud, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user home: %w", err)
		}
		root = filepath.Join(ud, DefaultDirName)
	}
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("absolute path: %w", err)
	}
	return &Home{Root: abs}, nil
}

// ProfilesDir returns the directory storing plain profiles (`*.hosts`).
func (h *Home) ProfilesDir() string { return filepath.Join(h.Root, "profiles") }

// CombinedDir returns the directory storing combined profile definitions (`*.toml`).
func (h *Home) CombinedDir() string { return filepath.Join(h.Root, "combined") }

// StatePath returns the path to state.json.
func (h *Home) StatePath() string { return filepath.Join(h.Root, "state.json") }

// ConfigPath returns the path to config.yaml.
func (h *Home) ConfigPath() string { return filepath.Join(h.Root, "config.yaml") }

// EnsureDirs creates all required directories if they do not exist yet.
func (h *Home) EnsureDirs() error {
	dirs := []string{h.Root, h.ProfilesDir(), h.CombinedDir()}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("mkdir %s: %w", d, err)
		}
	}
	return nil
}

// Exists reports whether the home root exists.
func (h *Home) Exists() bool {
	_, err := os.Stat(h.Root)
	return err == nil
}

// MustExist returns an error suitable for users when the home is missing.
func (h *Home) MustExist() error {
	if !h.Exists() {
		return fmt.Errorf("althosts home does not exist: %s (run `althosts init`)", h.Root)
	}
	return nil
}

// ErrAlreadyInitialized indicates that althosts init was called twice.
var ErrAlreadyInitialized = errors.New("althosts home is already initialized")
