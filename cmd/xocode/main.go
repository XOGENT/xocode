// Command xocode is the entrypoint. It is intentionally thin — all wiring lives
// in internal/cli.
package main

import (
	"os"

	"github.com/xogent/xocode/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
