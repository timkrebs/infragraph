package cmd

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/timkrebs/infragraph/internal/api"
	"github.com/timkrebs/infragraph/internal/collector"
	"github.com/timkrebs/infragraph/internal/graph"
	"github.com/timkrebs/infragraph/internal/store"
)

const shutdownTimeout = 10 * time.Second

// newLogger builds an slog.Logger from the [server] config block.
// If LogFile is set the logger also writes to that file; the caller is
// responsible for closing the returned *os.File (may be nil).
func newLogger(cfg ServerConfig) (*slog.Logger, *os.File, error) {
	level := parseLogLevel(cfg.LogLevel)

	var w io.Writer = os.Stdout
	var logFile *os.File

	if cfg.LogFile != "" {
		f, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return nil, nil, fmt.Errorf("open log file %q: %w", cfg.LogFile, err)
		}
		logFile = f
		w = io.MultiWriter(os.Stdout, f)
	}

	opts := &slog.HandlerOptions{Level: level}

	var handler slog.Handler
	if cfg.LogFormat == "json" {
		handler = slog.NewJSONHandler(w, opts)
	} else {
		handler = slog.NewTextHandler(w, opts)
	}

	return slog.New(handler), logFile, nil
}

// parseLogLevel maps the HCL log_level string to an slog.Level.
// Unknown or empty values default to Info.
func parseLogLevel(level string) slog.Level {
	switch level {
	case "debug", "trace":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// runServer starts the HTTP(S) server and blocks until SIGINT/SIGTERM (or /v1/sys/shutdown),
// then performs a graceful shutdown. Returns 0 on clean exit, 1 on error.
func runServer(bindAddr string, port int, cfg ServerConfig, listener ListenerConfig, st store.Store, collectors []collector.Collector) (int, error) {
	logger, logFile, err := newLogger(cfg)
	if err != nil {
		return 1, err
	}
	if logFile != nil {
		defer logFile.Close()
	}

	// Context that is cancelled on OS signal OR /v1/sys/shutdown endpoint.
	ctx, cancel := context.WithCancel(context.Background())

	// Wire OS signals to cancel.
	sigCtx, stop := signal.NotifyContext(ctx, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Load the initial graph snapshot.
	graphPtr := &atomic.Pointer[graph.Graph]{}
	initialGraph, err := st.LoadGraph()
	if err != nil {
		cancel()
		st.Close()
		return 1, fmt.Errorf("load initial graph: %w", err)
	}
	graphPtr.Store(&initialGraph)

	// makeEmitFunc returns the collector callback that persists events and
	// signals the debounced graph-refresh goroutine.
	refreshCh := make(chan struct{}, 1) // signal channel for debounced reload

	makeEmitFunc := func() collector.EventFunc {
		return func(ev collector.Event) {
			if ev.Node == nil {
				return
			}
			switch ev.Kind {
			case collector.EventUpsert:
				if err := st.UpsertNode(ev.Node); err != nil {
					logger.Warn("upsert node failed", "id", ev.Node.ID, "err", err)
					return
				}
				for _, e := range ev.Edges {
					if err := st.UpsertEdge(e); err != nil {
						logger.Warn("upsert edge failed", "from", e.From, "to", e.To, "err", err)
					}
				}
			case collector.EventDelete:
				if err := st.DeleteNode(ev.Node.ID); err != nil {
					logger.Warn("delete node failed", "id", ev.Node.ID, "err", err)
					return
				}
			}
			// Signal graph refresh (non-blocking).
			select {
			case refreshCh <- struct{}{}:
			default:
			}
		}
	}

	// Debounced graph-refresh goroutine: coalesces rapid events and reloads
	// the in-memory graph at most once per 100ms.
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		const debounceInterval = 100 * time.Millisecond
		timer := time.NewTimer(debounceInterval)
		timer.Stop()
		defer timer.Stop()

		for {
			select {
			case <-refreshCh:
				timer.Reset(debounceInterval)
			case <-timer.C:
				newGraph, err := st.LoadGraph()
				if err != nil {
					logger.Warn("reload graph failed", "err", err)
					continue
				}
				graphPtr.Store(&newGraph)
			case <-sigCtx.Done():
				return
			}
		}
	}()

	// Start all collectors.
	for _, c := range collectors {
		c := c
		emit := makeEmitFunc()
		wg.Add(1)
		go func() {
			defer wg.Done()
			logger.Info("collector started", "collector", c.Name())
			if err := c.Run(sigCtx, emit); err != nil && sigCtx.Err() == nil {
				logger.Warn("collector exited with error", "collector", c.Name(), "err", err)
			}
			logger.Info("collector stopped", "collector", c.Name())
		}()
	}

	// Build router with the shutdown cancel wired in.
	router := api.NewRouter(st, graphPtr, logger, api.RouterOpts{
		APIToken:  cfg.APIToken,
		RateLimit: cfg.RateLimit,
	})

	// Expose the shutdown cancel so /v1/sys/shutdown can trigger it.
	// We reach into the *api.Handlers via a type assertion on the mux — instead,
	// we pass cancel through a wrapper that api.NewRouter supports.
	// Since NewRouter returns an http.Handler wrapping *api.Handlers, we set it
	// via a dedicated function on the concrete type through the package.
	api.SetShutdown(router, cancel)
	defer router.Close() // stop rate limiter cleanup goroutine

	// Log effective configuration summary.
	logger.Info("configuration",
		"bind", fmt.Sprintf("%s:%d", bindAddr, port),
		"tls", listener.TLSCertFile != "" && !listener.TLSDisable,
		"auth", cfg.APIToken != "",
		"rate_limit", cfg.RateLimit,
		"log_level", cfg.LogLevel,
		"log_format", cfg.LogFormat,
		"store", st.Path(),
		"collectors", len(collectors),
	)

	addr := fmt.Sprintf("%s:%d", bindAddr, port)
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Configure TLS when cert/key are provided via the listener block.
	useTLS := listener.TLSCertFile != "" && listener.TLSKeyFile != "" && !listener.TLSDisable
	if useTLS {
		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
		}
		if v := listener.TLSMinVersion; v != "" {
			ver, _ := parseTLSVersion(v) // already validated by Config.validate()
			tlsCfg.MinVersion = ver
		}
		if v := listener.TLSMaxVersion; v != "" {
			ver, _ := parseTLSVersion(v)
			tlsCfg.MaxVersion = ver
		}
		srv.TLSConfig = tlsCfg
	}

	serverErr := make(chan error, 1)
	go func() {
		if useTLS {
			logger.Info("server listening (TLS)", "addr", addr,
				"cert", listener.TLSCertFile, "key", listener.TLSKeyFile,
				"min_version", listener.TLSMinVersion, "max_version", listener.TLSMaxVersion)
			if err := srv.ListenAndServeTLS(listener.TLSCertFile, listener.TLSKeyFile); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		} else {
			logger.Info("server listening", "addr", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErr <- err
			}
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		cancel()
		if err != nil {
			return 1, fmt.Errorf("server error: %w", err)
		}
	case <-sigCtx.Done():
		stop()
		cancel()
		logger.Info("shutting down server", "timeout", shutdownTimeout)

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer shutdownCancel()

		if err := srv.Shutdown(shutdownCtx); err != nil {
			return 1, fmt.Errorf("graceful shutdown failed: %w", err)
		}
	}

	// Wait for collectors and debounce goroutine to exit before closing the store.
	wg.Wait()
	logger.Info("server stopped")

	return 0, nil
}
