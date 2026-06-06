package profile

import (
	"strings"
	"testing"

	"github.com/xshoji/althosts/internal/home"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()
	h := &home.Home{Root: t.TempDir()}
	if err := h.EnsureDirs(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	return New(h)
}

func TestRender_Profile(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateProfile("dev", []byte("127.0.0.1 app.local\n")); err != nil {
		t.Fatalf("create: %v", err)
	}
	r, err := s.Render("dev")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if r.Kind != KindProfile {
		t.Fatalf("kind = %v, want profile", r.Kind)
	}
	if string(r.Body) != "127.0.0.1 app.local\n" {
		t.Fatalf("body = %q", r.Body)
	}
}

func TestRender_Combined(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateProfile("a", []byte("1.1.1.1 a.local\n")); err != nil {
		t.Fatal(err)
	}
	if err := s.CreateProfile("b", []byte("2.2.2.2 b.local")); err != nil {
		t.Fatal(err)
	}
	combinedYAML := "members: [a, b]\n"
	if err := writeFile(t, s.CombinedPath("ab"), combinedYAML); err != nil {
		t.Fatal(err)
	}
	r, err := s.Render("ab")
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if r.Kind != KindCombined {
		t.Fatalf("kind = %v", r.Kind)
	}
	body := string(r.Body)
	for _, want := range []string{
		"# althosts: combined ab",
		"# generated from: a, b",
		"# --- profile: a ---",
		"# --- profile: b ---",
		"1.1.1.1 a.local",
		"2.2.2.2 b.local",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("missing %q in:\n%s", want, body)
		}
	}
}

func TestRender_NestedCombinedRejected(t *testing.T) {
	s := newTestStore(t)
	if err := s.CreateProfile("a", []byte("1.1.1.1 a\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(t, s.CombinedPath("inner"), "members: [a]\n"); err != nil {
		t.Fatal(err)
	}
	if err := writeFile(t, s.CombinedPath("outer"), "members: [inner]\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Render("outer"); err == nil {
		t.Fatal("expected error for nested combined")
	}
}

func TestRender_MissingMember(t *testing.T) {
	s := newTestStore(t)
	if err := writeFile(t, s.CombinedPath("c"), "members: [missing]\n"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Render("c"); err == nil {
		t.Fatal("expected error for missing member")
	}
}

func writeFile(t *testing.T, path, body string) error {
	t.Helper()
	return writeFileBytes(path, []byte(body))
}
