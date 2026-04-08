package cmd

import (
	"flag"
	"fmt"

	cli "github.com/timkrebs/gocli"
)

type ServerStopCmd struct{ Ui cli.Ui }

func (c *ServerStopCmd) Name() string     { return "server stop" }
func (c *ServerStopCmd) Synopsis() string { return "Stop the running InfraGraph server" }
func (c *ServerStopCmd) Help() string {
	return "Usage: infragraph server stop [--server HOST:PORT]"
}

func (c *ServerStopCmd) Run(args []string) int {
	fs := flag.NewFlagSet("server stop", flag.ContinueOnError)
	serverAddr := fs.String("server", defaultServerAddr(), "InfraGraph server address")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	client := NewAPIClient(*serverAddr)

	var resp map[string]string
	if err := client.PostJSON("/v1/sys/shutdown", &resp); err != nil {
		c.Ui.Error(fmt.Sprintf("Cannot reach server at %s: %s", *serverAddr, err))
		return 1
	}

	c.Ui.Info(fmt.Sprintf("Server at %s is shutting down.", *serverAddr))
	return 0
}
