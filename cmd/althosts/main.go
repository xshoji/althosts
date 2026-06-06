// Command althosts switches /etc/hosts profiles safely on local machines.
package main

import (
	"fmt"
	"os"

	"github.com/xshoji/althosts/internal/cli"
)

func main() {
	if err := cli.NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
