package cmd

import (
	"encoding/json"
	"flag"
	"fmt"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/version"
)

// VersionCmd implements `infragraph version`.
type VersionCmd struct{ Ui cli.Ui }

func (c *VersionCmd) Name() string     { return "version" }
func (c *VersionCmd) Synopsis() string { return "Print version information" }
func (c *VersionCmd) Help() string {
	return `Usage: infragraph version [options]

Print the version and build information.

Options:
  --json    Output version information as JSON.
`
}

func (c *VersionCmd) Run(args []string) int {
	fs := flag.NewFlagSet("version", flag.ContinueOnError)
	jsonFlag := fs.Bool("json", false, "Output as JSON")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	info := version.GetInfo()

	if *jsonFlag {
		out, _ := json.MarshalIndent(info, "", "  ")
		c.Ui.Output(string(out))
		return 0
	}

	c.Ui.Output(version.HumanVersion())
	c.Ui.Output(fmt.Sprintf("  Go version:     %s", info.GoVersion))
	c.Ui.Output(fmt.Sprintf("  Platform:       %s", info.Platform))
	return 0
}
