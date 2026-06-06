package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/hostsfile"
)

func newCreateCmd(_ *app) *cobra.Command {
	var (
		fromCurrent bool
		empty       bool
		fromPath    string
	)
	c := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new plain profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := appFrom(cmd.Context())
			if err := a.requireHome(); err != nil {
				return err
			}
			name := args[0]

			var content []byte
			switch {
			case empty:
				content = []byte{}
			case fromPath != "":
				b, err := readFromSource(a, fromPath)
				if err != nil {
					return fmt.Errorf("read --from: %w", err)
				}
				content = b
			case fromCurrent:
				b, err := hostsfile.Read(a.cfg.HostsPath)
				if err != nil {
					return fmt.Errorf("read current hosts: %w", err)
				}
				content = b
			default:
				// default behaviour: clone current hosts
				b, err := hostsfile.Read(a.cfg.HostsPath)
				if err != nil {
					return fmt.Errorf("read current hosts: %w", err)
				}
				content = b
			}

			if err := a.store.CreateProfile(name, content); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created profile %q at %s\n", name, a.store.ProfilePath(name))
			return nil
		},
	}
	c.Flags().BoolVar(&fromCurrent, "from-current", false, "use the current /etc/hosts as content (default)")
	c.Flags().BoolVar(&empty, "empty", false, "create an empty profile")
	c.Flags().StringVar(&fromPath, "from", "", "create profile from an existing profile name or a local file path")
	return c
}

// readFromSource resolves --from. If the value matches an existing profile
// (plain or combined), its rendered body is returned; otherwise the value
// is treated as a local file path.
func readFromSource(a *app, src string) ([]byte, error) {
	if _, err := a.store.Lookup(src); err == nil {
		r, err := a.store.Render(src)
		if err != nil {
			return nil, err
		}
		return r.Body, nil
	}
	return os.ReadFile(src)
}
