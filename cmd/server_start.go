package cmd

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/internal/collector"
	"github.com/timkrebs/infragraph/internal/store"
)

type ServerStartCmd struct{ Ui cli.Ui }

func (c *ServerStartCmd) Name() string     { return "server start" }
func (c *ServerStartCmd) Synopsis() string { return "Run the InfraGraph server" }
func (c *ServerStartCmd) Help() string {
	return `Usage: infragraph server start [options]

Options:
  --port PORT        Listen port (overrides config file value, default: 8080).
  --config FILE      Path to an infragraph.hcl config file.
  --tls-cert FILE    Path to TLS certificate PEM file (overrides config).
  --tls-key FILE     Path to TLS private key PEM file (overrides config).
`
}

func (c *ServerStartCmd) Run(args []string) int {
	fs := flag.NewFlagSet("server start", flag.ContinueOnError)
	portFlag := fs.Int("port", 0, "Listen port (overrides config file value)")
	configFile := fs.String("config", "", "Path to infragraph.hcl config file")
	tlsCert := fs.String("tls-cert", "", "Path to TLS certificate PEM file")
	tlsKey := fs.String("tls-key", "", "Path to TLS private key PEM file")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	// Load configuration from file when provided.
	var fullCfg *Config
	serverCfg := ServerConfig{}
	listenerCfg := ListenerConfig{}
	if *configFile != "" {
		cfg, err := LoadConfig(*configFile)
		if err != nil {
			c.Ui.Error(fmt.Sprintf("Failed to load config %q: %s", *configFile, err))
			return 1
		}
		fullCfg = cfg
		serverCfg = cfg.Server
		// Use the first TCP listener block if present.
		for _, l := range cfg.Listeners {
			if l.Type == "tcp" {
				listenerCfg = l
				break
			}
		}
	}

	// CLI --tls-cert / --tls-key flags override the config file listener block.
	if *tlsCert != "" {
		listenerCfg.TLSCertFile = *tlsCert
	}
	if *tlsKey != "" {
		listenerCfg.TLSKeyFile = *tlsKey
	}

	// Validate TLS flag pairing.
	if (listenerCfg.TLSCertFile != "") != (listenerCfg.TLSKeyFile != "") {
		c.Ui.Error("both --tls-cert and --tls-key must be provided together")
		return 1
	}

	// Resolve bind address: config → default.
	bindAddr := "0.0.0.0"
	if serverCfg.BindAddr != "" {
		bindAddr = serverCfg.BindAddr
	}

	// Resolve port: --port flag -> config -> default.
	port := 8080
	if serverCfg.Port != 0 {
		port = serverCfg.Port
	}
	if *portFlag != 0 {
		port = *portFlag
	}
	if port < 1 || port > 65535 {
		c.Ui.Error(fmt.Sprintf("port must be between 1 and 65535, got %d", port))
		return 1
	}

	serverCfg.Port = port
	serverCfg.BindAddr = bindAddr

	// Determine store path: config -> default temp path.
	storePath := filepath.Join(os.TempDir(), "infragraph.db")
	if fullCfg != nil && fullCfg.Store.Path != "" {
		storePath = fullCfg.Store.Path
	}

	st, err := store.Open(storePath)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to open store at %q: %s", storePath, err))
		return 1
	}
	defer st.Close()

	// Build collector list. Always include the static seed collector so the
	// server is immediately queryable. Real collectors (K8s, Docker) are added
	// in Phase 2 based on the HCL [collector "..."] blocks.
	collectors := []collector.Collector{
		&collector.StaticCollector{},
	}

	c.Ui.Info(fmt.Sprintf("Starting InfraGraph server on %s:%d...", bindAddr, port))
	c.Ui.Info(fmt.Sprintf("Store: %s", storePath))
	if listenerCfg.TLSCertFile != "" {
		c.Ui.Info(fmt.Sprintf("TLS: cert=%s key=%s", listenerCfg.TLSCertFile, listenerCfg.TLSKeyFile))
	}

	exitCode, err := runServer(bindAddr, port, serverCfg, listenerCfg, st, collectors)
	if err != nil {
		c.Ui.Error(err.Error())
	}
	return exitCode
}
