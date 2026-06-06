package cli

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/config"
	"github.com/xshoji/althosts/internal/dns"
	"github.com/xshoji/althosts/internal/hostsfile"
	"github.com/xshoji/althosts/internal/state"
	"github.com/xshoji/althosts/internal/validate"
)

func newUseCmd(_ *app) *cobra.Command {
	var skipFlush, force bool
	c := &cobra.Command{
		Use:   "use <name>",
		Short: "Apply a profile to /etc/hosts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := appFrom(cmd.Context())
			if err := a.requireHome(); err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			ee := cmd.ErrOrStderr()

			r, err := a.store.Render(args[0])
			if err != nil {
				return err
			}

			findings := validate.Hosts(r.Body)
			for _, f := range findings {
				if f.Line > 0 {
					fmt.Fprintf(ee, "[%s] line %d: %s\n", f.Severity, f.Line, f.Message)
				} else {
					fmt.Fprintf(ee, "[%s] %s\n", f.Severity, f.Message)
				}
			}
			if validate.HasError(findings) && !force {
				return fmt.Errorf("validation failed (use --force to override)")
			}

			customHostsPath := !config.IsDefaultHostsPath(a.cfg.HostsPath)
			if customHostsPath && os.Geteuid() == 0 {
				return fmt.Errorf("custom hosts_path %q must be applied without sudo/root", a.cfg.HostsPath)
			}

			if err := hostsfile.EnsureWritable(a.cfg.HostsPath); err != nil {
				if errors.Is(err, hostsfile.ErrPermission) && customHostsPath {
					return fmt.Errorf("custom hosts_path %q is not writable by the current user; refusing to use sudo for non-default hosts_path", a.cfg.HostsPath)
				}
				if errors.Is(err, hostsfile.ErrPermission) && os.Geteuid() != 0 && !alreadyElevated() {
					return reExecWithSudoSameArgs(cmd, a.home.Root)
				}
				return err
			}

			if err := hostsfile.Write(a.cfg.HostsPath, r.Body); err != nil {
				return fmt.Errorf("write hosts: %w", err)
			}

			if a.cfg.FlushDNS && !skipFlush {
				if err := dns.Flush(ee); err != nil {
					fmt.Fprintf(ee, "warning: dns flush failed: %v\n", err)
				}
			}

			st := state.State{
				Active:    args[0],
				Kind:      state.Kind(r.Kind),
				AppliedAt: time.Now(),
				HostsPath: a.cfg.HostsPath,
			}
			if err := state.Save(a.home.StatePath(), st); err != nil {
				return fmt.Errorf("write state: %w", err)
			}

			fmt.Fprintf(out, "Applied %s (%s) to %s\n", r.Name, r.Kind, a.cfg.HostsPath)
			return nil
		},
	}
	c.Flags().BoolVar(&skipFlush, "no-flush", false, "skip DNS cache flush")
	c.Flags().BoolVar(&force, "force", false, "apply even when validation fails")
	return c
}
