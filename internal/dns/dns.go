// Package dns flushes the operating system DNS cache after /etc/hosts is updated.
package dns

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

// Flush flushes the OS DNS cache. Currently implemented for darwin only.
// On other platforms it is a no-op and returns nil.
//
// Output of the underlying commands is written to stderrW.
func Flush(stderrW io.Writer) error {
	switch runtime.GOOS {
	case "darwin":
		return flushDarwin(stderrW)
	default:
		return nil
	}
}

func flushDarwin(stderrW io.Writer) error {
	cmd := exec.Command("/usr/bin/dscacheutil", "-flushcache")
	cmd.Stderr = stderrW
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("dscacheutil -flushcache: %w", err)
	}
	// killall -HUP mDNSResponder is best-effort; ignore failure.
	_ = exec.Command("/usr/bin/killall", "-HUP", "mDNSResponder").Run()
	return nil
}
