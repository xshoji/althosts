package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

// runSelf runs this binary with the given argv (without the program name)
// as a child process. No sudo wrapping is added; the child decides whether
// it needs to self-elevate.
func runSelf(cmd *cobra.Command, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate self: %w", err)
	}
	sub := exec.Command(self, args...)
	sub.Stdin = os.Stdin
	sub.Stdout = cmd.OutOrStdout()
	sub.Stderr = cmd.ErrOrStderr()
	return waitChild(sub)
}

// runSelfWithSudo runs this binary with the given argv prefixed by `sudo`
// (or directly when already root). Used for self-elevation.
func runSelfWithSudo(cmd *cobra.Command, args []string) error {
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate self: %w", err)
	}
	var sub *exec.Cmd
	if os.Geteuid() == 0 {
		sub = exec.Command(self, args...)
	} else {
		sudo, err := findSudo()
		if err != nil {
			return fmt.Errorf("sudo not found at a trusted path (re-run as root): %w", err)
		}
		sub = exec.Command(sudo, append([]string{self}, args...)...)
	}
	sub.Stdin = os.Stdin
	sub.Stdout = cmd.OutOrStdout()
	sub.Stderr = cmd.ErrOrStderr()
	return waitChild(sub)
}

// findSudo intentionally avoids PATH lookup. This trades some compatibility
// with unusual Linux setups for safer self-elevation: a user-controlled PATH
// must not be able to replace the password-prompting sudo binary.
func findSudo() (string, error) {
	for _, p := range []string{"/usr/bin/sudo", "/bin/sudo"} {
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.Mode().IsRegular() && st.Mode().Perm()&0o111 != 0 {
			return p, nil
		}
	}
	return "", os.ErrNotExist
}

func waitChild(sub *exec.Cmd) error {
	if err := sub.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("child failed: %w", err)
	}
	return nil
}

// elevateMarkerEnv is set on the sudo'd child so that the inner process
// does not loop indefinitely if EnsureWritable still returns permission
// denied for some reason (e.g. immutable file flags).
const elevateMarkerEnv = "ALTHOSTS_ELEVATED"

func alreadyElevated() bool { return os.Getenv(elevateMarkerEnv) != "" }

// reExecWithSudoSameArgs re-runs the current invocation under sudo using
// os.Args. It sets ALTHOSTS_ELEVATED=1 so the child won't try to elevate
// again.
//
// The resolved althosts home is passed explicitly as --home because sudo
// resets HOME to the target user's home (e.g. /var/root) by default, which
// would otherwise make the child resolve a different althosts home than
// the parent process did.
func reExecWithSudoSameArgs(cmd *cobra.Command, homeRoot string) error {
	if alreadyElevated() {
		return fmt.Errorf("still cannot write hosts file even after elevating")
	}
	os.Setenv(elevateMarkerEnv, "1")
	defer os.Unsetenv(elevateMarkerEnv)
	return runSelfWithSudo(cmd, withHomeArg(os.Args[1:], homeRoot))
}

// withHomeArg prepends --home <root> to args unless --home is already
// specified (either as --home=value or --home value).
func withHomeArg(args []string, root string) []string {
	if root == "" {
		return args
	}
	for _, a := range args {
		if a == "--home" || strings.HasPrefix(a, "--home=") {
			return args
		}
	}
	return append([]string{"--home", root}, args...)
}
