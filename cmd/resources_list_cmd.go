package cmd

import (
	"flag"
	"fmt"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/internal/graph"
)

// ResourcesListCmd implements "infragraph resources list".
type ResourcesListCmd struct {
	Ui cli.Ui
}

func (c *ResourcesListCmd) Help() string {
	return `Usage: infragraph resources list [--type TYPE] [--namespace NS] [--server HOST:PORT]

  Lists all discovered infrastructure resources.

Options:
  --type TYPE         Filter by resource type (e.g. pod, service, secret).
  --namespace NS      Filter by Kubernetes namespace.
  --server HOST:PORT  Address of the InfraGraph server (default: 127.0.0.1:8080).
`
}

func (c *ResourcesListCmd) Synopsis() string {
	return "List discovered infrastructure resources"
}

func (c *ResourcesListCmd) Run(args []string) int {
	fs := flag.NewFlagSet("resources list", flag.ContinueOnError)
	serverAddr := fs.String("server", defaultServerAddr(), "InfraGraph server address")
	filterType := fs.String("type", "", "Filter by resource type")
	filterNS := fs.String("namespace", "", "Filter by namespace")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	path := "/v1/resources"
	sep := "?"
	if *filterType != "" {
		path += sep + "type=" + *filterType
		sep = "&"
	}
	if *filterNS != "" {
		path += sep + "namespace=" + *filterNS
	}

	client := NewAPIClient(*serverAddr)

	var resp struct {
		Nodes []*graph.Node `json:"nodes"`
	}
	if err := client.GetJSON(path, &resp); err != nil {
		c.Ui.Error(fmt.Sprintf("Error: %s", err))
		return 1
	}

	if len(resp.Nodes) == 0 {
		c.Ui.Info("No resources found.")
		return 0
	}

	// Print table header.
	c.Ui.Output(fmt.Sprintf("%-40s  %-12s  %-10s  %-10s  %s",
		"ID", "TYPE", "PROVIDER", "STATUS", "NAMESPACE"))
	c.Ui.Output(fmt.Sprintf("%-40s  %-12s  %-10s  %-10s  %s",
		"──────────────────────────────────────", "────────────", "──────────", "──────────", "─────────"))

	for _, n := range resp.Nodes {
		c.Ui.Output(fmt.Sprintf("%-40s  %-12s  %-10s  %-10s  %s",
			n.ID, n.Type, n.Provider, n.Status, n.Namespace))
	}
	return 0
}
