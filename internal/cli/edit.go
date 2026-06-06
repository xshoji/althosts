package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/editor"
	"github.com/xshoji/althosts/internal/profile"
	"github.com/xshoji/althosts/internal/state"
)

func newEditCmd(_ *app) *cobra.Command {
	return &cobra.Command{
		Use:   "edit <name>",
		Short: "Edit a profile or combined definition in $EDITOR (auto-applies if active)",
		Long: `Edit a profile or combined definition in $EDITOR.

The kind is detected automatically: plain profiles open their .hosts file
and combined definitions open their .yaml file. When editing a combined
definition, althosts first asks whether to edit one of its member profiles
or the combined YAML itself. If the edited target is currently active (or
a member of the active combined profile), this command automatically runs
"althosts use <active>" after the editor exits so the changes hit
/etc/hosts immediately. The editor itself never runs under sudo; only the
apply step does (and sudo will prompt for a password if needed).`,
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

			target := entry
			if entry.Kind == profile.KindCombined {
				c, err := a.store.LoadCombined(name)
				if err != nil {
					return err
				}
				chosen, err := promptCombinedEditTarget(cmd, a, name, c, entry)
				if err != nil {
					return err
				}
				target = chosen
			}

			if err := editor.Open(target.Path, a.cfg.Editor); err != nil {
				return err
			}
			if target.Kind == profile.KindCombined {
				if _, err := a.store.LoadCombined(target.Name); err != nil {
					return fmt.Errorf("combined %q is invalid after edit: %w", target.Name, err)
				}
			}

			st, err := state.Load(a.home.StatePath())
			if err != nil || st.Active == "" || !isAffected(a, st.Active, target.Name) {
				return nil
			}

			useArgs := []string{}
			if home, _ := cmd.Root().PersistentFlags().GetString("home"); home != "" {
				useArgs = append(useArgs, "--home", home)
			}
			useArgs = append(useArgs, "use", st.Active)
			// runSelf does not prepend sudo; the child `use` will
			// self-elevate only if /etc/hosts is not writable.
			return runSelf(cmd, useArgs)
		},
	}
}

// promptCombinedEditTarget asks the user which file inside a combined
// definition to edit: one of its members, or the combined YAML itself.
func promptCombinedEditTarget(cmd *cobra.Command, a *app, name string, c profile.Combined, self profile.Entry) (profile.Entry, error) {
	type choice struct {
		label string
		entry profile.Entry
	}
	var choices []choice
	for _, m := range c.Members {
		ent, err := a.store.Lookup(m)
		if err != nil {
			choices = append(choices, choice{
				label: fmt.Sprintf("%s (missing)", m),
				entry: profile.Entry{Name: m, Kind: profile.KindProfile, Path: a.store.ProfilePath(m)},
			})
			continue
		}
		choices = append(choices, choice{
			label: fmt.Sprintf("%s (%s)", ent.Name, ent.Kind),
			entry: ent,
		})
	}
	choices = append(choices, choice{
		label: "Edit combined definition itself (YAML)",
		entry: self,
	})

	out := cmd.OutOrStderr()
	fmt.Fprintf(out, "Select a member to edit in %s:\n", name)
	for i, ch := range choices {
		fmt.Fprintf(out, "  [%d] %s\n", i+1, ch.label)
	}
	fmt.Fprintf(out, "Enter number [1-%d]: ", len(choices))

	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && (!errors.Is(err, io.EOF) || line == "") {
		return profile.Entry{}, fmt.Errorf("read selection: %w", err)
	}
	line = strings.TrimSpace(line)
	if line == "" {
		return profile.Entry{}, fmt.Errorf("no selection given")
	}
	n, err := strconv.Atoi(line)
	if err != nil || n < 1 || n > len(choices) {
		return profile.Entry{}, fmt.Errorf("invalid selection %q (expected 1-%d)", line, len(choices))
	}
	return choices[n-1].entry, nil
}

// isAffected reports whether editing `edited` affects the rendered output
// of `active` (either same profile, or active is a combined that lists it).
func isAffected(a *app, active, edited string) bool {
	if active == edited {
		return true
	}
	c, err := a.store.LoadCombined(active)
	if err != nil {
		return false
	}
	for _, m := range c.Members {
		if m == edited {
			return true
		}
	}
	return false
}
