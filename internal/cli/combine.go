package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/profile"
)

func newCombineCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "combine <name> <member> [<member>...]",
		Short: "Create a combined profile from plain profile members",
		Long: `Create a combined profile that concatenates the listed plain profile
members at apply time. Members must be plain profiles that already exist;
combined profiles cannot reference other combined profiles.`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := appFrom(cmd.Context())
			if err := a.requireHome(); err != nil {
				return err
			}
			name := args[0]
			members := args[1:]

			for _, m := range members {
				entry, err := a.store.Lookup(m)
				if err != nil {
					return fmt.Errorf("member %q: %w", m, err)
				}
				if entry.Kind == profile.KindCombined {
					return fmt.Errorf("member %q is a combined profile; nested combined is not allowed", m)
				}
			}

			c := profile.Combined{Members: members}
			if err := a.store.WriteCombined(name, c, true); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created combined %q at %s\n", name, a.store.CombinedPath(name))
			return nil
		},
	}
}
