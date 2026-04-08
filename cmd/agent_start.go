package cmd

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	cli "github.com/timkrebs/gocli"

	"github.com/timkrebs/infragraph/internal/collector"
)

// AgentStartCmd implements `infragraph agent start` — the Vault Agent-style
// process that discovers infrastructure resources and pushes them to a remote
// InfraGraph server over an authenticated HTTPS connection.
type AgentStartCmd struct{ Ui cli.Ui }

func (c *AgentStartCmd) Name() string { return "agent start" }
func (c *AgentStartCmd) Synopsis() string {
	return "Run the InfraGraph agent (collector → server push)"
}
func (c *AgentStartCmd) Help() string {
	return `Usage: infragraph agent start [options]

Run a collector locally and push discovered resources to a remote InfraGraph
server. This is the Vault Agent-style deployment model for secure, distributed
infrastructure discovery.

Options:
  --config FILE      Path to an infragraph-agent.hcl config file (required).
`
}

func (c *AgentStartCmd) Run(args []string) int {
	fs := flag.NewFlagSet("agent start", flag.ContinueOnError)
	configFile := fs.String("config", "", "Path to agent config file (required)")
	if err := fs.Parse(args); err != nil {
		c.Ui.Error(err.Error())
		return 1
	}

	if *configFile == "" {
		c.Ui.Error("--config is required")
		return 1
	}

	cfg, err := LoadAgentConfig(*configFile)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to load agent config %q: %s", *configFile, err))
		return 1
	}

	// Set up structured logger.
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	// Build the local collector.
	col, err := buildAgentCollector(cfg.Collector, logger)
	if err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to configure collector: %s", err))
		return 1
	}

	// Build the push client.
	agentName := cfg.Server.AgentName
	if agentName == "" {
		hostname, _ := os.Hostname()
		agentName = fmt.Sprintf("%s-%s", col.Name(), hostname)
	}

	push := &collector.PushClient{
		ServerAddr:    cfg.Server.Address,
		Token:         cfg.Server.Token,
		AgentName:     agentName,
		TLSCert:       cfg.Server.TLSCert,
		TLSKey:        cfg.Server.TLSKey,
		TLSCACert:     cfg.Server.TLSCACert,
		TLSSkipVerify: cfg.Server.TLSSkipVerify,
		Logger:        logger,
	}
	if err := push.Init(); err != nil {
		c.Ui.Error(fmt.Sprintf("Failed to initialize push client: %s", err))
		return 1
	}

	// Context for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Register with the server.
	c.Ui.Info(fmt.Sprintf("Registering agent %q with server %s...", agentName, cfg.Server.Address))
	if err := push.Register(ctx); err != nil {
		c.Ui.Error(fmt.Sprintf("Registration failed: %s", err))
		return 1
	}
	c.Ui.Info("Agent registered successfully")

	// Set up the batched push emit function.
	emit, flushFn := push.WrapWithPush(ctx, 100, 5*time.Second)

	c.Ui.Info(fmt.Sprintf("Starting %s collector (agent mode, push to %s)...", col.Name(), cfg.Server.Address))

	// Run the collector. This blocks until ctx is cancelled.
	if err := col.Run(ctx, emit); err != nil && ctx.Err() == nil {
		c.Ui.Error(fmt.Sprintf("Collector error: %s", err))
		flushFn() // flush remaining events
		return 1
	}

	flushFn() // flush remaining events
	c.Ui.Info("Agent stopped")
	return 0
}

// buildAgentCollector creates a single collector from the agent config.
func buildAgentCollector(cc CollectorConfig, logger *slog.Logger) (collector.Collector, error) {
	switch cc.Type {
	case "static":
		return &collector.StaticCollector{}, nil

	case "kubernetes":
		interval := 60 * time.Second
		if cc.ReconcileInterval != "" {
			d, err := time.ParseDuration(cc.ReconcileInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid reconcile_interval: %w", err)
			}
			interval = d
		}
		return &collector.KubernetesCollector{
			KubeConfig:        cc.KubeConfig,
			Context:           cc.Context,
			Namespaces:        cc.Namespaces,
			Resources:         cc.Resources,
			ReconcileInterval: interval,
			Logger:            logger,
		}, nil

	case "docker":
		interval := 60 * time.Second
		if cc.ReconcileInterval != "" {
			d, err := time.ParseDuration(cc.ReconcileInterval)
			if err != nil {
				return nil, fmt.Errorf("invalid reconcile_interval: %w", err)
			}
			interval = d
		}
		return &collector.DockerCollector{
			Socket:            cc.Socket,
			ReconcileInterval: interval,
			Logger:            logger,
		}, nil

	default:
		return nil, fmt.Errorf("unknown collector type %q", cc.Type)
	}
}
