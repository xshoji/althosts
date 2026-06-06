// Package profile manages plain hosts profiles and combined profile definitions.
package profile

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/xshoji/althosts/internal/home"
	"gopkg.in/yaml.v3"
)

// Kind distinguishes plain hosts profiles from combined definitions.
type Kind string

const (
	// KindProfile is a plain hosts profile stored as `<name>.hosts`.
	KindProfile Kind = "profile"
	// KindCombined is a combined definition stored as `<name>.yaml`.
	KindCombined Kind = "combined"
)

// ProfileExt is the extension for plain hosts profiles.
const ProfileExt = ".hosts"

// CombinedExt is the extension for combined definitions.
const CombinedExt = ".yaml"

// ErrNotFound is returned when a profile or combined cannot be found.
var ErrNotFound = errors.New("profile not found")

// ErrAlreadyExists is returned when attempting to create a duplicate name.
var ErrAlreadyExists = errors.New("profile already exists")

// ErrInvalidName is returned when a profile name has invalid characters.
var ErrInvalidName = errors.New("invalid profile name")

// Entry describes a profile or combined definition known to althosts.
type Entry struct {
	Name string
	Kind Kind
	Path string
}

// Combined represents the parsed body of a combined definition file.
type Combined struct {
	Members []string `yaml:"members"`
}

// Store is a filesystem-backed store for profiles and combined definitions.
type Store struct {
	Home *home.Home
}

// New returns a Store rooted at the given home.
func New(h *home.Home) *Store { return &Store{Home: h} }

// ValidateName ensures the name is safe to use as a filename component.
func ValidateName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty", ErrInvalidName)
	}
	if strings.ContainsAny(name, `/\:*?"<>|`) {
		return fmt.Errorf("%w: %q contains forbidden characters", ErrInvalidName, name)
	}
	if strings.HasPrefix(name, ".") {
		return fmt.Errorf("%w: %q must not start with '.'", ErrInvalidName, name)
	}
	return nil
}

// ProfilePath returns the absolute path for a plain profile.
func (s *Store) ProfilePath(name string) string {
	return filepath.Join(s.Home.ProfilesDir(), name+ProfileExt)
}

// CombinedPath returns the absolute path for a combined definition.
func (s *Store) CombinedPath(name string) string {
	return filepath.Join(s.Home.CombinedDir(), name+CombinedExt)
}

// Lookup returns the entry for a given name, searching profiles first then combined.
func (s *Store) Lookup(name string) (Entry, error) {
	if err := ValidateName(name); err != nil {
		return Entry{}, err
	}
	pp := s.ProfilePath(name)
	if _, err := os.Stat(pp); err == nil {
		return Entry{Name: name, Kind: KindProfile, Path: pp}, nil
	}
	cp := s.CombinedPath(name)
	if _, err := os.Stat(cp); err == nil {
		return Entry{Name: name, Kind: KindCombined, Path: cp}, nil
	}
	return Entry{}, fmt.Errorf("%w: %s", ErrNotFound, name)
}

// List returns all profiles and combined definitions sorted alphabetically.
func (s *Store) List() ([]Entry, error) {
	var entries []Entry
	if err := s.collectByExt(s.Home.ProfilesDir(), ProfileExt, KindProfile, &entries); err != nil {
		return nil, err
	}
	if err := s.collectByExt(s.Home.CombinedDir(), CombinedExt, KindCombined, &entries); err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Name == entries[j].Name {
			return entries[i].Kind < entries[j].Kind
		}
		return entries[i].Name < entries[j].Name
	})
	return entries, nil
}

func (s *Store) collectByExt(dir, ext string, kind Kind, out *[]Entry) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read dir %s: %w", dir, err)
	}
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ext) {
			continue
		}
		base := strings.TrimSuffix(name, ext)
		if base == "" {
			continue
		}
		*out = append(*out, Entry{
			Name: base,
			Kind: kind,
			Path: filepath.Join(dir, name),
		})
	}
	return nil
}

// CreateProfile creates a new plain profile with the given content.
// It refuses to overwrite an existing profile or combined of the same name.
func (s *Store) CreateProfile(name string, content []byte) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if _, err := s.Lookup(name); err == nil {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, name)
	}
	if err := os.MkdirAll(s.Home.ProfilesDir(), 0o755); err != nil {
		return fmt.Errorf("mkdir profiles: %w", err)
	}
	return os.WriteFile(s.ProfilePath(name), content, 0o644)
}

// LoadProfile reads a plain profile's body.
func (s *Store) LoadProfile(name string) ([]byte, error) {
	if err := ValidateName(name); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(s.ProfilePath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return nil, fmt.Errorf("read profile: %w", err)
	}
	return b, nil
}

// WriteCombined atomically writes a combined definition. It refuses to
// overwrite an existing profile or combined of the same name when create
// is true.
func (s *Store) WriteCombined(name string, c Combined, create bool) error {
	if err := ValidateName(name); err != nil {
		return err
	}
	if create {
		if _, err := s.Lookup(name); err == nil {
			return fmt.Errorf("%w: %s", ErrAlreadyExists, name)
		}
	}
	if err := os.MkdirAll(s.Home.CombinedDir(), 0o755); err != nil {
		return fmt.Errorf("mkdir combined: %w", err)
	}
	body, err := yaml.Marshal(&c)
	if err != nil {
		return fmt.Errorf("encode combined: %w", err)
	}
	dst := s.CombinedPath(name)
	tmp, err := os.CreateTemp(s.Home.CombinedDir(), ".althosts.combined.*")
	if err != nil {
		return fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(body); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Chmod(0o644); err != nil {
		tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("chmod temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Rename(tmpPath, dst); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename: %w", err)
	}
	return nil
}

// DeleteEntry removes the file backing the given entry.
func (s *Store) DeleteEntry(entry Entry) error {
	if err := os.Remove(entry.Path); err != nil {
		return fmt.Errorf("remove %s: %w", entry.Path, err)
	}
	return nil
}

// LoadCombined reads a combined definition.
func (s *Store) LoadCombined(name string) (Combined, error) {
	var c Combined
	if err := ValidateName(name); err != nil {
		return c, err
	}
	b, err := os.ReadFile(s.CombinedPath(name))
	if err != nil {
		if os.IsNotExist(err) {
			return c, fmt.Errorf("%w: %s", ErrNotFound, name)
		}
		return c, fmt.Errorf("read combined: %w", err)
	}
	if err := yaml.Unmarshal(b, &c); err != nil {
		return c, fmt.Errorf("parse combined %s: %w", name, err)
	}
	return c, nil
}
