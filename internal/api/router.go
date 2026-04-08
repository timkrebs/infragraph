package api

import (
	"context"
	"log/slog"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/timkrebs/infragraph/internal/graph"
	"github.com/timkrebs/infragraph/internal/store"
	"github.com/timkrebs/infragraph/ui"
)

// requestTimeout is the per-request context deadline for v1 API handlers.
const requestTimeout = 10 * time.Second

// Router wraps the HTTP mux and exposes server-lifecycle hooks.
// It implements http.Handler so it can be passed directly to http.Server.
type Router struct {
	mux      *http.ServeMux
	handlers *Handlers
	rl       *rateLimiter // may be nil; kept for Close()
}

// ServeHTTP implements http.Handler.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	r.mux.ServeHTTP(w, req)
}

// Close releases background resources (e.g. rate limiter goroutine).
func (r *Router) Close() {
	if r.rl != nil {
		r.rl.Close()
	}
}

// RouterOpts configures the router across security and rate-limiting dimensions.
type RouterOpts struct {
	// APIToken is the Bearer token required for /v1/* endpoints.
	// Empty disables auth (dev mode).
	APIToken string
	// RateLimit is max requests per second per IP. 0 disables rate limiting.
	RateLimit int
}

// NewRouter builds the HTTP handler with all v1 routes wired to their handlers.
// graphState is an atomic pointer that always holds the latest graph snapshot.
// Handlers read from this pointer lock-free; the collector loop swaps it on updates.
func NewRouter(st store.Store, graphState *atomic.Pointer[graph.Graph], logger *slog.Logger, opts RouterOpts) *Router {
	h := &Handlers{store: st, graph: graphState, log: logger}
	mux := http.NewServeMux()

	// Build rate limiter (nil if disabled).
	var rl *rateLimiter
	if opts.RateLimit > 0 {
		rl = newRateLimiter(opts.RateLimit, opts.RateLimit*2, time.Second)
	}

	// wrap chains: requestID → version header → log → rate-limit → auth → timeout → handler
	wrap := func(h http.HandlerFunc) http.HandlerFunc {
		return requestIDMiddleware(
			versionHeaderMiddleware(
				logMiddleware(logger,
					rateLimitMiddleware(rl,
						authMiddleware(opts.APIToken,
							requestTimeoutMiddleware(requestTimeout, h))))))
	}

	// /health is always unauthenticated and unthrottled (load-balancer probes).
	mux.HandleFunc("/health", requestIDMiddleware(versionHeaderMiddleware(h.Health)))

	mux.HandleFunc("/v1/sys/status", wrap(h.SysStatus))
	mux.HandleFunc("/v1/sys/shutdown", wrap(h.SysShutdown))
	mux.HandleFunc("/v1/graph", wrap(h.GraphFull))
	mux.HandleFunc("/v1/resources", wrap(h.ResourcesList))
	mux.HandleFunc("/v1/graph/node/", wrap(h.GraphNode))
	mux.HandleFunc("/v1/graph/impact/", wrap(h.GraphImpact))

	// Serve the embedded web UI at /ui/.
	mux.Handle("/ui/", ui.Handler())
	// Redirect bare /ui to /ui/ so the SPA loads correctly.
	mux.HandleFunc("/ui", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/ui/", http.StatusMovedPermanently)
	})

	return &Router{mux: mux, handlers: h, rl: rl}
}

// SetShutdown registers the cancel function called by POST /v1/sys/shutdown.
func SetShutdown(handler http.Handler, cancel context.CancelFunc) {
	if r, ok := handler.(*Router); ok {
		r.handlers.SetShutdown(cancel)
	}
}

// responseCapture wraps http.ResponseWriter to capture the status code.
type responseCapture struct {
	http.ResponseWriter
	status int
}

func (rc *responseCapture) WriteHeader(code int) {
	rc.status = code
	rc.ResponseWriter.WriteHeader(code)
}

// logMiddleware logs every request with method, path, query, status, and duration.
func logMiddleware(logger *slog.Logger, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rc := &responseCapture{ResponseWriter: w, status: http.StatusOK}

		next(rc, r)

		attrs := []any{
			"method", r.Method,
			"path", r.URL.Path,
			"status", rc.status,
			"duration_ms", time.Since(start).Milliseconds(),
			"remote_addr", r.RemoteAddr,
		}
		if q := r.URL.RawQuery; q != "" {
			attrs = append(attrs, "query", q)
		}
		if ua := r.UserAgent(); ua != "" {
			attrs = append(attrs, "user_agent", ua)
		}
		if rid := w.Header().Get("X-Request-ID"); rid != "" {
			attrs = append(attrs, "request_id", rid)
		}
		logger.Info("request", attrs...)
	}
}
