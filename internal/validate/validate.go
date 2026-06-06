// Package validate inspects hosts file contents and reports issues.
package validate

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"regexp"
	"strings"
)

// Severity describes how serious a finding is.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Finding is a single validation issue.
type Finding struct {
	Severity Severity
	Line     int // 1-based, 0 means file-level
	Message  string
}

// HasError reports whether any finding is an error.
func HasError(findings []Finding) bool {
	for _, f := range findings {
		if f.Severity == SeverityError {
			return true
		}
	}
	return false
}

var hostnameRE = regexp.MustCompile(`^(?i)([a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)(\.[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?)*$`)

// Hosts validates the body of a hosts file.
func Hosts(body []byte) []Finding {
	var findings []Finding
	if len(bytes.TrimSpace(body)) == 0 {
		findings = append(findings, Finding{
			Severity: SeverityWarning,
			Message:  "file is empty",
		})
		return findings
	}

	hostFirstSeen := map[string]int{}

	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNo := 0
	for sc.Scan() {
		lineNo++
		raw := sc.Text()
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				Line:     lineNo,
				Message:  fmt.Sprintf("expected `<ip> <hostname> [<hostname>...]`, got %q", raw),
			})
			continue
		}
		ip := fields[0]
		if net.ParseIP(ip) == nil {
			findings = append(findings, Finding{
				Severity: SeverityWarning,
				Line:     lineNo,
				Message:  fmt.Sprintf("invalid IP address %q", ip),
			})
		}
		for _, h := range fields[1:] {
			if !hostnameRE.MatchString(h) || len(h) > 253 {
				findings = append(findings, Finding{
					Severity: SeverityWarning,
					Line:     lineNo,
					Message:  fmt.Sprintf("invalid hostname %q", h),
				})
				continue
			}
			hl := strings.ToLower(h)
			if hl == "localhost" {
				continue
			}
			if first, ok := hostFirstSeen[hl]; ok {
				findings = append(findings, Finding{
					Severity: SeverityWarning,
					Line:     lineNo,
					Message:  fmt.Sprintf("hostname %q duplicates entry on line %d", h, first),
				})
			} else {
				hostFirstSeen[hl] = lineNo
			}
		}
	}
	if err := sc.Err(); err != nil {
		findings = append(findings, Finding{
			Severity: SeverityError,
			Message:  fmt.Sprintf("scan failed: %v", err),
		})
	}
	return findings
}
