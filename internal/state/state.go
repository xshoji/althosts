// Package state manages state.json which records the active profile.
package state

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Kind is the kind of the active profile.
type Kind string

const (
	KindProfile  Kind = "profile"
	KindCombined Kind = "combined"
)

// State represents the serialized contents of state.json.
type State struct {
	Active    string    `json:"active"`
	Kind      Kind      `json:"kind"`
	AppliedAt time.Time `json:"applied_at"`
	HostsPath string    `json:"hosts_path"`
}

// Load reads state.json. If the file does not exist, an empty State is returned.
func Load(path string) (State, error) {
	var s State
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return s, nil
		}
		return s, fmt.Errorf("read state: %w", err)
	}
	if len(b) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return s, fmt.Errorf("parse state: %w", err)
	}
	return s, nil
}

// Save writes the state to path with 0o644.
func Save(path string, s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		return fmt.Errorf("write state: %w", err)
	}
	return nil
}
