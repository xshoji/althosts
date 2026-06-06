package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/config"
	"github.com/xshoji/althosts/internal/hostsfile"
	"github.com/xshoji/althosts/internal/profile"
)

const defaultProfileName = "default"

func newInitCmd(_ *app) *cobra.Command {
	var skipDefault bool
	c := &cobra.Command{
		Use:   "init",
		Short: "Create the althosts home directory and seed a `default` profile",
		Long: `Create the althosts home directory and a default config.

By default, init also snapshots the current /etc/hosts as a profile named
"default" so you can always get back to the original state with:

    sudo althosts use default

Pass --no-default to skip this step.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := appFrom(cmd.Context())
			out := cmd.OutOrStdout()

			if err := a.home.EnsureDirs(); err != nil {
				return err
			}
			cfgPath := a.home.ConfigPath()
			if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
				if err := config.Save(cfgPath, config.Default()); err != nil {
					return err
				}
			}
			fmt.Fprintf(out, "Initialized althosts home at %s\n", a.home.Root)

			if skipDefault {
				return nil
			}
			if _, err := a.store.Lookup(defaultProfileName); err == nil {
				// already exists - leave it untouched
				return nil
			} else if !errors.Is(err, profile.ErrNotFound) {
				return err
			}

			body, err := hostsfile.Read(a.cfg.HostsPath)
			if err != nil {
				fmt.Fprintf(cmd.ErrOrStderr(),
					"warning: skipped seeding %q profile: read %s: %v\n",
					defaultProfileName, a.cfg.HostsPath, err)
				return nil
			}
			if err := a.store.CreateProfile(defaultProfileName, body); err != nil {
				return fmt.Errorf("seed default profile: %w", err)
			}
			fmt.Fprintf(out, "Seeded profile %q from %s\n", defaultProfileName, a.cfg.HostsPath)
			return nil
		},
	}
	c.Flags().BoolVar(&skipDefault, "no-default", false, "do not seed a `default` profile from current /etc/hosts")
	return c
}
