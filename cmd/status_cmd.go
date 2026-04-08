package cmd

import (
	"flag"
	"fmt"

	cli "github.com/timkrebs/gocli"
)

// StatusCmd implements "infragraph status".
// It calls GET /v1/sys/status on the running server and prints a summary.
type StatusCmd struct {
	Ui cli.Ui
}

func (c *StatusCmd) Help() string {
	return `Usage: infragraph status [--server HOST:PORT]

  Displays the status of the running InfraGraph server.

Options:
  --server HOST:PORT  Address of the InfraGraph server (default: 127.0.0.1:8080,
                      override with INFRAGRAPH_ADDR env var).
`
}

func (c *StatusCmd) Synopsis() string {
	return "Display server status"
}

func (c *StatusCmd) Run(args []string) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	serverAddr := fs.String("server", defaultServerAddr(), "InfraGraph server address")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	client := NewAPIClient(*serverAddr)

	var status struct {
		Version   string `json:"version"`
		NodeCount int    `json:"node_count"`
		EdgeCount int    `json:"edge_count"`
		StorePath string `json:"store_path"`
	}
	if err := client.GetJSON("/v1/sys/status", &status); err != nil {
		c.Ui.Error(fmt.Sprintf("Cannot reach server at %s: %s", *serverAddr, err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("InfraGraph v%s", status.Version))
	c.Ui.Info(fmt.Sprintf("  Nodes      : %d", status.NodeCount))
	c.Ui.Info(fmt.Sprintf("  Edges      : %d", status.EdgeCount))
	c.Ui.Info(fmt.Sprintf("  Store path : %s", status.StorePath))
	return 0
}
