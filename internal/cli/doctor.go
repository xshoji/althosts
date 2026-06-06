package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/state"
)

func newDoctorCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose the althosts environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := appFrom(cmd.Context())
			out := cmd.OutOrStdout()

			report(out, "althosts home exists", a.home.Exists(), a.home.Root)
			report(out, "profiles dir exists", dirExists(a.home.ProfilesDir()), a.home.ProfilesDir())
			report(out, "combined dir exists", dirExists(a.home.CombinedDir()), a.home.CombinedDir())

			st, _ := os.Stat(a.cfg.HostsPath)
			report(out, "hosts file exists", st != nil, a.cfg.HostsPath)
			if st != nil {
				warnIfUnwritable(out, a.cfg.HostsPath)
			}

			if runtime.GOOS == "darwin" {
				_, err := exec.LookPath("dscacheutil")
				report(out, "dscacheutil found", err == nil, "/usr/bin/dscacheutil")
			}

			s, err := state.Load(a.home.StatePath())
			if err != nil {
				fmt.Fprintf(out, "[warn] state.json: %v\n", err)
			} else if s.Active != "" {
				_, lerr := a.store.Lookup(s.Active)
				report(out, fmt.Sprintf("active profile %q exists", s.Active), lerr == nil, "")
			}
			return nil
		},
	}
}

func report(w io.Writer, label string, ok bool, detail string) {
	tag := "ok"
	if !ok {
		tag = "warn"
	}
	if detail == "" {
		fmt.Fprintf(w, "[%s] %s\n", tag, label)
	} else {
		fmt.Fprintf(w, "[%s] %s (%s)\n", tag, label, detail)
	}
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}

func warnIfUnwritable(w io.Writer, p string) {
	f, err := os.OpenFile(p, os.O_WRONLY|os.O_APPEND, 0)
	if err != nil {
		fmt.Fprintf(w, "[warn] %s is not writable by current user (sudo needed for `use`)\n", p)
		return
	}
	f.Close()
	fmt.Fprintf(w, "[ok] %s is writable\n", p)
}
