package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/internal/store"
)

type DbMigrateCmd struct{ Ui cli.Ui }

func (c *DbMigrateCmd) Name() string     { return "db migrate" }
func (c *DbMigrateCmd) Synopsis() string { return "Initialize or migrate the graph database" }
func (c *DbMigrateCmd) Help() string {
	return "Usage: infragraph db migrate [--config FILE]"
}

func (c *DbMigrateCmd) Run(args []string) int {
	fs := flag.NewFlagSet("db migrate", flag.ContinueOnError)
	configFile := fs.String("config", "", "Path to infragraph.hcl config file")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	storePath := filepath.Join(os.TempDir(), "infragraph.db")
	if *configFile != "" {
		cfg, err := LoadConfig(*configFile)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load config %q: %s", *configFile, err))
			return 1
		}
		if cfg.Store.Path != "" {
			storePath = cfg.Store.Path
		}
	}

	c.Ui.Info(fmt.Sprintf("Running database migration on %s...", storePath))

	// Opening the store creates required bbolt buckets if they don't exist.
	st, err := store.Open(storePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Migration failed: %s", err))
		return 1
	}
	defer st.Close()

	nodeCount, err := st.NodeCount()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to count nodes: %s", err))
		return 1
	}
	edgeCount, err := st.EdgeCount()
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to count edges: %s", err))
		return 1
	}

	c.Ui.Info("Migration complete.")
	c.Ui.Info(fmt.Sprintf("  Nodes : %d", nodeCount))
	c.Ui.Info(fmt.Sprintf("  Edges : %d", edgeCount))
	return 0
}
