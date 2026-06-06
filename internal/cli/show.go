package cli

import (
	"github.com/spf13/cobra"
)

func newShowCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show the rendered hosts content for a profile",
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
			_, err = cmd.OutOrStdout().Write(r.Body)
			return err
		},
	}
}
