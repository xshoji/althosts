package cli

import (
	"bufio"
	"bytes"
	"fmt"
	"io"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/hostsfile"
)

func newDiffCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "diff <name>",
		Short: "Show diff between current /etc/hosts and a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := appFrom(cmd.Context())
			if err := a.requireHome(); err != nil {
				return err
			}
			r, err := a.store.Render(args[0])
			if err != nil {
				return err
			}
			cur, err := hostsfile.Read(a.cfg.HostsPath)
			if err != nil {
				return fmt.Errorf("read current hosts: %w", err)
			}
			return writeUnifiedDiff(cmd.OutOrStdout(), a.cfg.HostsPath, "althosts:"+args[0], cur, r.Body)
		},
	}
}

// writeUnifiedDiff writes a minimal line-by-line diff between a and b.
// This is not a full unified-diff (no hunk grouping with @@ ... @@), but
// is sufficient to give a clear visual diff for hosts files which are
// usually small.
func writeUnifiedDiff(w io.Writer, aLabel, bLabel string, a, b []byte) error {
	if bytes.Equal(a, b) {
		fmt.Fprintf(w, "no changes\n")
		return nil
	}
	fmt.Fprintf(w, "--- %s\n", aLabel)
	fmt.Fprintf(w, "+++ %s\n", bLabel)
	aLines := splitLines(a)
	bLines := splitLines(b)
	// Plain LCS-free output: mark all of A as `-` and all of B as `+`,
	// but skip equal leading / trailing lines so output stays readable.
	start := 0
	for start < len(aLines) && start < len(bLines) && aLines[start] == bLines[start] {
		start++
	}
	endA, endB := len(aLines), len(bLines)
	for endA > start && endB > start && aLines[endA-1] == bLines[endB-1] {
		endA--
		endB--
	}
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	for i := start; i < endA; i++ {
		fmt.Fprintf(bw, "- %s\n", aLines[i])
	}
	for i := start; i < endB; i++ {
		fmt.Fprintf(bw, "+ %s\n", bLines[i])
	}
	return nil
}

func splitLines(b []byte) []string {
	if len(b) == 0 {
		return nil
	}
	s := string(b)
	if s[len(s)-1] == '\n' {
		s = s[:len(s)-1]
	}
	if s == "" {
		return nil
	}
	out := make([]string, 0, 32)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}
