package profile

import (
	"bytes"
	"fmt"
	"strings"
)

// RenderResult is the output of rendering a profile (plain or combined).
type RenderResult struct {
	Name    string
	Kind    Kind
	Members []string // populated for combined; empty for plain
	Body    []byte
}

// Render returns the hosts text that would be applied to /etc/hosts when
// the named profile is activated. Plain profiles return their stored body
// verbatim; combined profiles concatenate their members with comment headers.
//
// Combined profiles must not reference other combined profiles.
func (s *Store) Render(name string) (RenderResult, error) {
	entry, err := s.Lookup(name)
	if err != nil {
		return RenderResult{}, err
	}
	switch entry.Kind {
	case KindProfile:
		body, err := s.LoadProfile(name)
		if err != nil {
			return RenderResult{}, err
		}
		return RenderResult{Name: name, Kind: KindProfile, Body: body}, nil
	case KindCombined:
		return s.renderCombined(name)
	default:
		return RenderResult{}, fmt.Errorf("unknown kind: %s", entry.Kind)
	}
}

func (s *Store) renderCombined(name string) (RenderResult, error) {
	c, err := s.LoadCombined(name)
	if err != nil {
		return RenderResult{}, err
	}
	if len(c.Members) == 0 {
		return RenderResult{}, fmt.Errorf("combined %s has no members", name)
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "# althosts: combined %s\n", name)
	fmt.Fprintf(&buf, "# generated from: %s\n", strings.Join(c.Members, ", "))
	buf.WriteByte('\n')

	for _, m := range c.Members {
		entry, err := s.Lookup(m)
		if err != nil {
			return RenderResult{}, fmt.Errorf("combined %s: member %s: %w", name, m, err)
		}
		if entry.Kind == KindCombined {
			return RenderResult{}, fmt.Errorf("combined %s: member %s is also combined; nested combined is not allowed", name, m)
		}
		body, err := s.LoadProfile(m)
		if err != nil {
			return RenderResult{}, fmt.Errorf("combined %s: load %s: %w", name, m, err)
		}
		fmt.Fprintf(&buf, "# --- profile: %s ---\n\n", m)
		buf.Write(body)
		if len(body) > 0 && body[len(body)-1] != '\n' {
			buf.WriteByte('\n')
		}
		buf.WriteByte('\n')
	}
	return RenderResult{
		Name:    name,
		Kind:    KindCombined,
		Members: c.Members,
		Body:    buf.Bytes(),
	}, nil
}
