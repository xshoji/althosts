package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/profile"
	"github.com/xshoji/althosts/internal/state"
)

func newListCmd(_ *app) *cobra.Command {
	var asJSON, currentOnly bool
	c := &cobra.Command{
		Use:   "list",
		Short: "List profiles and combined definitions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			a := appFrom(cmd.Context())
			if err := a.requireHome(); err != nil {
				return err
			}

			st, err := state.Load(a.home.StatePath())
			if err != nil {
				return err
			}

			if currentOnly {
				if st.Active == "" {
					return nil
				}
				fmt.Fprintln(cmd.OutOrStdout(), st.Active)
				return nil
			}

			entries, err := a.store.List()
			if err != nil {
				return err
			}

			if asJSON {
				return printListJSON(cmd.OutOrStdout(), a, st, entries)
			}
			return printListText(cmd.OutOrStdout(), a, st, entries)
		},
	}
	c.Flags().BoolVar(&asJSON, "json", false, "output as JSON")
	c.Flags().BoolVar(&currentOnly, "current", false, "print only the active profile name")
	return c
}

func printListText(w io.Writer, a *app, st state.State, entries []profile.Entry) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, e := range entries {
		mark := " "
		if st.Active == e.Name && string(st.Kind) == string(e.Kind) {
			mark = "*"
		}
		extra := ""
		if e.Kind == profile.KindCombined {
			c, err := a.store.LoadCombined(e.Name)
			if err == nil {
				extra = strings.Join(c.Members, " + ")
			}
		}
		fmt.Fprintf(tw, "%s\t%s\t%s\t%s\n", mark, e.Name, e.Kind, extra)
	}
	return tw.Flush()
}

type listJSONEntry struct {
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Active  bool     `json:"active"`
	Members []string `json:"members,omitempty"`
}

func printListJSON(w io.Writer, a *app, st state.State, entries []profile.Entry) error {
	out := make([]listJSONEntry, 0, len(entries))
	for _, e := range entries {
		je := listJSONEntry{
			Name:   e.Name,
			Kind:   string(e.Kind),
			Active: st.Active == e.Name && string(st.Kind) == string(e.Kind),
		}
		if e.Kind == profile.KindCombined {
			c, err := a.store.LoadCombined(e.Name)
			if err == nil {
				je.Members = c.Members
			}
		}
		out = append(out, je)
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
