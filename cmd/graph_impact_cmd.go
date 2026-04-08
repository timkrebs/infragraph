package cmd

import (
	"flag"
	"fmt"
	"strings"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/internal/graph"
)

// GraphImpactCmd implements "infragraph graph impact <id>".
type GraphImpactCmd struct {
	Ui cli.Ui
}

func (c *GraphImpactCmd) Help() string {
	return `Usage: infragraph graph impact <resource-id> [--reverse] [--depth N] [--server HOST:PORT]

  Shows the blast radius of a resource change — all nodes reachable from it.

Arguments:
  <resource-id>  The node ID, e.g. "service/user-api"

Options:
  --reverse           Walk backwards (what does this depend on?) instead of forward.
  --depth N           Maximum traversal depth (default: 10).
  --server HOST:PORT  Address of the InfraGraph server (default: 127.0.0.1:8080).
`
}

func (c *GraphImpactCmd) Synopsis() string {
	return "Show the blast radius of a resource change"
}

func (c *GraphImpactCmd) Run(args []string) int {
	fs := flag.NewFlagSet("graph impact", flag.ContinueOnError)
	serverAddr := fs.String("server", defaultServerAddr(), "InfraGraph server address")
	reverse := fs.Bool("reverse", false, "Traverse incoming edges (reverse impact)")
	depth := fs.Int("depth", 10, "Maximum traversal depth")

	flagArgs, posArgs := separateArgs(args)
	if err := fs.Parse(flagArgs); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if len(posArgs) < 1 {
		c.Ui.Error("resource-id is required")
		c.Ui.Info(c.Help())
		return 1
	}
	id := posArgs[0]

	direction := "forward"
	if *reverse {
		direction = "reverse"
	}

	path := fmt.Sprintf("/v1/graph/impact/%s?direction=%s&depth=%d", id, direction, *depth)
	client := NewAPIClient(*serverAddr)

	var result graph.ImpactResult
	if err := client.GetJSON(path, &result); err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err))
		return 1
	}

	arrow := "→"
	if *reverse {
		arrow = "←"
	}
	c.Ui.Info(fmt.Sprintf("Impact analysis for: %s  [%s]", result.Root.ID, direction))
	c.Ui.Output(fmt.Sprintf("  %s (%s / %s)", result.Root.ID, result.Root.Type, result.Root.Status))

	if len(result.Affected) == 0 {
		c.Ui.Output("  (no affected nodes)")
		return 0
	}

	for _, an := range result.Affected {
		indent := strings.Repeat("  ", an.Depth)
		c.Ui.Output(fmt.Sprintf("%s%s [%s] %s  (%s / %s)",
			indent, arrow, an.Edge.Type, an.Node.ID, an.Node.Type, an.Node.Status))
	}

	c.Ui.Output(fmt.Sprintf("\n%d node(s) affected.", len(result.Affected)))
	return 0
}
