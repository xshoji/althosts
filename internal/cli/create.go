package cli

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xshoji/althosts/internal/hostsfile"
)

func newCreateCmd(_ *app) *cobra.Command {
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

			// Interactive: ask which source to base the new profile on.
			content, err := chooseSourceInteractive(cmd, a)
			if err != nil {
				return err
			}

			if err := a.store.CreateProfile(name, content); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created profile %q at %s\n", name, a.store.ProfilePath(name))
			return nil
		},
	}
	return c
}

// sourceChoice describes one selectable source for creating a new profile.
type sourceChoice struct {
	label   string
	resolve func() ([]byte, error)
}

// buildSourceChoices assembles the interactive menu of sources a new plain
// profile can be based on: every existing profile/combined, the current
// hosts file, and an empty profile.
func buildSourceChoices(a *app) ([]sourceChoice, error) {
	entries, err := a.store.List()
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	var choices []sourceChoice
	for _, e := range entries {
		e := e // capture for closure
		choices = append(choices, sourceChoice{
			label: fmt.Sprintf("%s: %s", e.Kind, e.Name),
			resolve: func() ([]byte, error) {
				r, err := a.store.Render(e.Name)
				if err != nil {
					return nil, fmt.Errorf("render %s: %w", e.Name, err)
				}
				return r.Body, nil
			},
		})
	}
	choices = append(choices, sourceChoice{
		label: fmt.Sprintf("current hosts (%s)", a.cfg.HostsPath),
		resolve: func() ([]byte, error) {
			b, err := hostsfile.Read(a.cfg.HostsPath)
			if err != nil {
				return nil, fmt.Errorf("read current hosts: %w", err)
			}
			return b, nil
		},
	})
	choices = append(choices, sourceChoice{
		label:   "empty profile",
		resolve: func() ([]byte, error) { return []byte{}, nil },
	})
	return choices, nil
}

// chooseSourceInteractive prints a numbered menu of sources and reads the
// user's selection from stdin, returning the resolved profile content.
func chooseSourceInteractive(cmd *cobra.Command, a *app) ([]byte, error) {
	choices, err := buildSourceChoices(a)
	if err != nil {
		return nil, err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, "Choose a source for the new profile:")
	for i, c := range choices {
		fmt.Fprintf(out, "  [%d] %s\n", i+1, c.label)
	}
	fmt.Fprint(out, "Enter number: ")

	in := bufio.NewReader(os.Stdin)
	line, err := in.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("read selection: %w", err)
	}
	line = strings.TrimSpace(line)
	idx, err := strconv.Atoi(line)
	if err != nil || idx < 1 || idx > len(choices) {
		return nil, fmt.Errorf("invalid selection %q: expected 1-%d", line, len(choices))
	}
	return choices[idx-1].resolve()
}
