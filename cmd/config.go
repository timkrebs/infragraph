package cmd

import (
	"fmt"
	"time"

	"github.com/hashicorp/hcl/v2/hclsimple"
)

// Config is the top-level structure of an infragraph.hcl file.
type Config struct {
	Server     ServerConfig      `hcl:"server,block"`
	Store      StoreConfig       `hcl:"store,block"`
	Listeners  []ListenerConfig  `hcl:"listener,block"`
	Collectors []CollectorConfig `hcl:"collector,block"`
}

// ServerConfig holds the [server] block settings.
type ServerConfig struct {
	BindAddr  string `hcl:"bind_addr,optional"`
	Port      int    `hcl:"port,optional"`
	LogLevel  string `hcl:"log_level,optional"`
	LogFormat string `hcl:"log_format,optional"`
	LogFile   string `hcl:"log_file,optional"`
	APIToken  string `hcl:"api_token,optional"`
	RateLimit int    `hcl:"rate_limit,optional"`
}

// ListenerConfig holds a labeled [listener "<type>"] block (Vault-style).
type ListenerConfig struct {
	Type          string `hcl:"type,label"`
	Address       string `hcl:"address,optional"`
	TLSCertFile   string `hcl:"tls_cert_file,optional"`
	TLSKeyFile    string `hcl:"tls_key_file,optional"`
	TLSMinVersion string `hcl:"tls_min_version,optional"`
	TLSMaxVersion string `hcl:"tls_max_version,optional"`
	TLSDisable    bool   `hcl:"tls_disable,optional"`
}

// StoreConfig holds the [store] block settings.
type StoreConfig struct {
	Path string `hcl:"path,optional"`
}

// CollectorConfig holds a labeled [collector "<type>"] block.
// Fields that only apply to certain collector types are optional.
type CollectorConfig struct {
	Type string `hcl:"type,label"`

	// kubernetes collector fields
	KubeConfig        string   `hcl:"kubeconfig,optional"`
	Context           string   `hcl:"context,optional"`
	Namespaces        []string `hcl:"namespaces,optional"`
	Resources         []string `hcl:"resources,optional"`
	ReconcileInterval string   `hcl:"reconcile_interval,optional"`

	// docker collector fields
	Socket string `hcl:"socket,optional"`
}

// LoadConfig reads and parses an HCL configuration file from the given path.
func LoadConfig(path string) (*Config, error) {
	var cfg Config
	if err := hclsimple.DecodeFile(path, nil, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// validate checks that the config values are within acceptable ranges.
func (c *Config) validate() error {
	if c.Server.Port != 0 && (c.Server.Port < 1 || c.Server.Port > 65535) {
		return fmt.Errorf("server.port must be between 1 and 65535, got %d", c.Server.Port)
	}

	// Validate log_level against known set.
	if v := c.Server.LogLevel; v != "" {
		switch v {
		case "debug", "trace", "info", "warn", "error":
		default:
			return fmt.Errorf("server.log_level must be one of debug, trace, info, warn, error; got %q", v)
		}
	}

	// Validate log_format against known set.
	if v := c.Server.LogFormat; v != "" {
		switch v {
		case "json", "text":
		default:
			return fmt.Errorf("server.log_format must be 'json' or 'text'; got %q", v)
		}
	}

	// Validate rate_limit is non-negative.
	if c.Server.RateLimit < 0 {
		return fmt.Errorf("server.rate_limit must be non-negative, got %d", c.Server.RateLimit)
	}

	for i, l := range c.Listeners {
		if l.TLSCertFile != "" && l.TLSKeyFile == "" {
			return fmt.Errorf("listener[%d]: tls_cert_file requires tls_key_file", i)
		}
		if l.TLSKeyFile != "" && l.TLSCertFile == "" {
			return fmt.Errorf("listener[%d]: tls_key_file requires tls_cert_file", i)
		}
		if v := l.TLSMinVersion; v != "" {
			if _, ok := parseTLSVersion(v); !ok {
				return fmt.Errorf("listener[%d]: unknown tls_min_version %q (use tls12 or tls13)", i, v)
			}
		}
		if v := l.TLSMaxVersion; v != "" {
			if _, ok := parseTLSVersion(v); !ok {
				return fmt.Errorf("listener[%d]: unknown tls_max_version %q (use tls12 or tls13)", i, v)
			}
		}
	}

	// Validate collector configs.
	knownCollectors := map[string]bool{"static": true, "kubernetes": true, "docker": true}
	for i, cc := range c.Collectors {
		if !knownCollectors[cc.Type] {
			return fmt.Errorf("collector[%d]: unknown type %q (known: static, kubernetes, docker)", i, cc.Type)
		}
		if cc.ReconcileInterval != "" {
			if _, err := time.ParseDuration(cc.ReconcileInterval); err != nil {
				return fmt.Errorf("collector[%d]: invalid reconcile_interval %q: %w", i, cc.ReconcileInterval, err)
			}
		}
	}

	return nil
}

// parseTLSVersion maps HCL tls version strings to crypto/tls constants.
func parseTLSVersion(s string) (uint16, bool) {
	switch s {
	case "tls12", "tls1.2":
		return 0x0303, true // tls.VersionTLS12
	case "tls13", "tls1.3":
		return 0x0304, true // tls.VersionTLS13
	default:
		return 0, false
	}
}
