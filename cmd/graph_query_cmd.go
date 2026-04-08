package cmd

import (
	"flag"
	"fmt"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/internal/graph"
)

// GraphQueryCmd implements "infragraph graph query <id>".
type GraphQueryCmd struct {
	Ui cli.Ui
}

func (c *GraphQueryCmd) Help() string {
	return `Usage: infragraph graph query <resource-id> [--server HOST:PORT]

  Shows a resource node and its direct incoming/outgoing connections.

Arguments:
  <resource-id>  The node ID, e.g. "service/user-api"

Options:
  --server HOST:PORT  Address of the InfraGraph server (default: 127.0.0.1:8080).
`
}

func (c *GraphQueryCmd) Synopsis() string {
	return "Query a resource node and its direct connections"
}

func (c *GraphQueryCmd) Run(args []string) int {
	fs := flag.NewFlagSet("graph query", flag.ContinueOnError)
	serverAddr := fs.String("server", defaultServerAddr(), "InfraGraph server address")

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

	client := NewAPIClient(*serverAddr)

	var resp struct {
		Node         *graph.Node   `json:"node"`
		Outgoing     []*graph.Edge `json:"outgoing"`
		Incoming     []*graph.Edge `json:"incoming"`
		Neighbors    []*graph.Node `json:"neighbors"`
		Predecessors []*graph.Node `json:"predecessors"`
	}
	if err := client.GetJSON("/v1/graph/node/"+id, &resp); err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err))
		return 1
	}

	n := resp.Node
	c.Ui.Info(fmt.Sprintf("Node: %s", n.ID))
	c.Ui.Output(fmt.Sprintf("  Type      : %s", n.Type))
	c.Ui.Output(fmt.Sprintf("  Provider  : %s", n.Provider))
	c.Ui.Output(fmt.Sprintf("  Namespace : %s", n.Namespace))
	c.Ui.Output(fmt.Sprintf("  Status    : %s", n.Status))

	if len(resp.Outgoing) > 0 {
		c.Ui.Output("\nOutgoing edges:")
		for _, e := range resp.Outgoing {
			c.Ui.Output(fmt.Sprintf("  → [%s] %s  (weight %.1f)", e.Type, e.To, e.Weight))
		}
	}

	if len(resp.Incoming) > 0 {
		c.Ui.Output("\nIncoming edges:")
		for _, e := range resp.Incoming {
			c.Ui.Output(fmt.Sprintf("  ← [%s] %s  (weight %.1f)", e.Type, e.From, e.Weight))
		}
	}

	return 0
}
