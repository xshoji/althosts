package cli

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/validate"
)

func newValidateCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "validate <name>",
		Short: "Validate a profile's hosts content",
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
			findings := validate.Hosts(r.Body)
			if len(findings) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "ok")
				return nil
			}
			out := cmd.OutOrStdout()
			for _, f := range findings {
				if f.Line > 0 {
					fmt.Fprintf(out, "[%s] line %d: %s\n", f.Severity, f.Line, f.Message)
				} else {
					fmt.Fprintf(out, "[%s] %s\n", f.Severity, f.Message)
				}
			}
			if validate.HasError(findings) {
				return fmt.Errorf("validation failed")
			}
			return nil
		},
	}
}
