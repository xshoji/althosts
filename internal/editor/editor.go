// Package editor launches an external editor to edit a file.
package editor

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Open launches an editor on the given path. The editor is determined by:
//  1. configEditor (config.yaml `editor`)
//  2. $VISUAL
//  3. $EDITOR
//  4. "vi"
//
// configEditor and the env vars may include arguments separated by spaces.
func Open(path, configEditor string) error {
	cmdline := configEditor
	if cmdline == "" {
		cmdline = os.Getenv("VISUAL")
	}
	if cmdline == "" {
		cmdline = os.Getenv("EDITOR")
	}
	if cmdline == "" {
		cmdline = "vi"
	}
	parts := strings.Fields(cmdline)
	if len(parts) == 0 {
		return fmt.Errorf("editor command is empty")
	}
	args := append(parts[1:], path)
	cmd := exec.Command(parts[0], args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
