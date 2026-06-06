// Package cli implements the althosts command-line interface.
package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/config"
	"github.com/xshoji/althosts/internal/home"
	"github.com/xshoji/althosts/internal/profile"
)

// Version is set via -ldflags at build time.
var Version = "0.0.0-dev"

// app holds shared dependencies for command implementations.
type app struct {
	home  *home.Home
	cfg   config.Config
	store *profile.Store
}

// newApp resolves the home directory and loads configuration.
// It does NOT require the home to exist; callers should call requireHome
// when an existing althosts home is needed.
func newApp(homeFlag string) (*app, error) {
	h, err := home.Resolve(homeFlag)
	if err != nil {
		return nil, err
	}
	cfg, err := config.Load(h.ConfigPath())
	if err != nil {
		return nil, err
	}
	return &app{
		home:  h,
		cfg:   cfg,
		store: profile.New(h),
	}, nil
}

func (a *app) requireHome() error { return a.home.MustExist() }

// NewRootCmd builds the root cobra command.
func NewRootCmd() *cobra.Command {
	var homeFlag string

	root := &cobra.Command{
		Use:   "althosts",
		Short: "Switch /etc/hosts profiles safely",
		Long:  "althosts manages multiple /etc/hosts profiles locally and switches between them safely.",
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		Version:       Version,
		SilenceUsage:  true,
		SilenceErrors: false,
	}
	root.PersistentFlags().StringVar(&homeFlag, "home", "", "althosts home directory (overrides $ALTHOSTS_HOME)")

	mk := func(build func(*app) *cobra.Command) *cobra.Command {
		// Each subcommand resolves the app at execution time so that --home
		// has already been parsed.
		var c *cobra.Command
		c = build(nil) // build with nil to register flags / use line
		original := c.RunE
		c.RunE = func(cmd *cobra.Command, args []string) error {
			a, err := newApp(homeFlag)
			if err != nil {
				return err
			}
			cmd.SetContext(withApp(cmd.Context(), a))
			if original == nil {
				return fmt.Errorf("command %q has no implementation", c.Use)
			}
			return original(cmd, args)
		}
		return c
	}

	root.AddCommand(
		mk(newInitCmd),
		mk(newListCmd),
		mk(newCreateCmd),
		mk(newCombineCmd),
		mk(newEditCmd),
		mk(newShowCmd),
		mk(newDiffCmd),
		mk(newValidateCmd),
		mk(newUseCmd),
		mk(newRemoveCmd),
		mk(newDoctorCmd),
	)
	return root
}
