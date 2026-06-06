package validate

import "testing"

func TestHosts_Empty(t *testing.T) {
	f := Hosts(nil)
	if len(f) != 1 || f[0].Severity != SeverityWarning {
		t.Fatalf("findings=%v", f)
	}
}

func TestHosts_Valid(t *testing.T) {
	body := []byte("127.0.0.1 a.local\n# comment\n2.2.2.2 b.local c.local\n127.0.0.1 localhost\n::1 localhost\n")
	if f := Hosts(body); len(f) != 0 {
		t.Fatalf("expected no findings, got %v", f)
	}
}

func TestHosts_DuplicateAndBadIP(t *testing.T) {
	body := []byte("not-an-ip a.local\n127.0.0.1 a.local\n127.0.0.1 a.local\n")
	f := Hosts(body)
	var sawIP, sawDup bool
	for _, x := range f {
		if x.Line == 1 {
			sawIP = true
		}
		if x.Line == 3 {
			sawDup = true
		}
	}
	if !sawIP || !sawDup {
		t.Fatalf("expected ip+dup findings, got %v", f)
	}
}
