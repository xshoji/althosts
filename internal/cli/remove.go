package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/state"
)

func newRemoveCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a profile or combined definition",
		Long: `Remove a plain profile or combined definition by name. The kind is
detected automatically. If the target is currently active, removal is
refused — switch to a different profile first.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a := appFrom(cmd.Context())
			if err := a.requireHome(); err != nil {
				return err
			}
			name := args[0]
			entry, err := a.store.Lookup(name)
			if err != nil {
				return err
			}

			st, err := state.Load(a.home.StatePath())
			if err != nil {
				return err
			}
			if st.Active == entry.Name && string(st.Kind) == string(entry.Kind) {
				return fmt.Errorf("%q (%s) is currently active; switch to another profile before removing", entry.Name, entry.Kind)
			}

			if err := a.store.DeleteEntry(entry); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %s %q (%s)\n", entry.Kind, entry.Name, entry.Path)
			return nil
		},
	}
}
