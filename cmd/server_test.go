package cmd

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/timkrebs/infragraph/internal/api"
	"github.com/timkrebs/infragraph/internal/graph"
	"github.com/timkrebs/infragraph/internal/store"
)

// --- logger tests ---

func TestNewLogger_DefaultsToInfo(t *testing.T) {
	logger, f, err := newLogger(ServerConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		f.Close()
	}
	if !logger.Enabled(nil, slog.LevelInfo) {
		t.Error("expected Info level to be enabled")
	}
	if logger.Enabled(nil, slog.LevelDebug) {
		t.Error("expected Debug level to be disabled by default")
	}
}

func TestNewLogger_DebugLevel(t *testing.T) {
	logger, f, err := newLogger(ServerConfig{LogLevel: "debug"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		f.Close()
	}
	if !logger.Enabled(nil, slog.LevelDebug) {
		t.Error("expected Debug level to be enabled")
	}
}

func TestNewLogger_TraceMapsToDebug(t *testing.T) {
	logger, f, err := newLogger(ServerConfig{LogLevel: "trace"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		f.Close()
	}
	if !logger.Enabled(nil, slog.LevelDebug) {
		t.Error("expected trace to map to Debug level")
	}
}

func TestNewLogger_WarnLevel(t *testing.T) {
	logger, f, err := newLogger(ServerConfig{LogLevel: "warn"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f != nil {
		f.Close()
	}
	if !logger.Enabled(nil, slog.LevelWarn) {
		t.Error("expected Warn level to be enabled")
	}
	if logger.Enabled(nil, slog.LevelInfo) {
		t.Error("expected Info to be suppressed at warn level")
	}
}

func TestNewLogger_WritesToFile(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "test.log")

	logger, f, err := newLogger(ServerConfig{LogLevel: "info", LogFile: logPath})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if f == nil {
		t.Fatal("expected non-nil file handle")
	}

	logger.Info("hello from test")
	f.Close()

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("reading log file: %v", err)
	}
	if !strings.Contains(string(data), "hello from test") {
		t.Errorf("expected log message in file, got: %s", data)
	}
}

func TestNewLogger_BadLogFilePath(t *testing.T) {
	_, _, err := newLogger(ServerConfig{LogFile: "/nonexistent/path/test.log"})
	if err == nil {
		t.Fatal("expected error for invalid log file path")
	}
}

// --- HTTP handler tests ---

// newTestRouter builds a router backed by a real (temp) bbolt store for unit tests.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	g, err := st.LoadGraph()
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	graphPtr := &atomic.Pointer[graph.Graph]{}
	graphPtr.Store(&g)

	return api.NewRouter(st, graphPtr, slog.Default(), api.RouterOpts{})
}

func TestHealthHandler(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if body["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", body)
	}
}

func TestHealthHandler_Post(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodPost, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	// /health accepts any method; it just returns ok.
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 for POST /health, got %d", rec.Code)
	}
}

func TestUnknownRoute(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/notfound", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestSysStatusHandler(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/sys/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("response is not valid JSON: %v", err)
	}
	if _, ok := body["version"]; !ok {
		t.Error("expected 'version' in status response")
	}
}

func TestResourcesListHandler_Empty(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/resources", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestGraphNodeHandler_NotFound(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/graph/node/service/missing", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- auth middleware tests ---

func newTestRouterWithAuth(t *testing.T, token string) http.Handler {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	st, err := store.Open(dbPath)
	if err != nil {
		t.Fatalf("open test store: %v", err)
	}
	t.Cleanup(func() { st.Close() })

	g, err := st.LoadGraph()
	if err != nil {
		t.Fatalf("load graph: %v", err)
	}
	graphPtr := &atomic.Pointer[graph.Graph]{}
	graphPtr.Store(&g)

	return api.NewRouter(st, graphPtr, slog.Default(), api.RouterOpts{APIToken: token})
}

func TestAuth_RejectsMissingToken(t *testing.T) {
	handler := newTestRouterWithAuth(t, "my-secret-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/sys/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_RejectsWrongToken(t *testing.T) {
	handler := newTestRouterWithAuth(t, "my-secret-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/sys/status", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestAuth_AcceptsValidToken(t *testing.T) {
	handler := newTestRouterWithAuth(t, "my-secret-token")

	req := httptest.NewRequest(http.MethodGet, "/v1/sys/status", nil)
	req.Header.Set("Authorization", "Bearer my-secret-token")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestAuth_HealthBypassesAuth(t *testing.T) {
	handler := newTestRouterWithAuth(t, "my-secret-token")

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (no auth required for /health), got %d", rec.Code)
	}
}

func TestAuth_NoTokenConfigDisablesAuth(t *testing.T) {
	handler := newTestRouterWithAuth(t, "")

	req := httptest.NewRequest(http.MethodGet, "/v1/sys/status", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200 (auth disabled when token empty), got %d", rec.Code)
	}
}

// --- node ID validation tests ---

func TestGraphNodeHandler_InvalidNodeID(t *testing.T) {
	handler := newTestRouter(t)

	tests := []struct {
		name string
		path string
	}{
		{"empty", "/v1/graph/node/"},
		{"no_slash", "/v1/graph/node/justname"},
		{"upper_case", "/v1/graph/node/Service/name"},
		{"special_chars", "/v1/graph/node/svc/@#$"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("expected 400 for %q, got %d", tt.path, rec.Code)
			}
		})
	}
}

// --- depth parameter validation tests ---

func TestGraphImpactHandler_BadDepth(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/test?depth=abc", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for bad depth, got %d", rec.Code)
	}
}

func TestGraphImpactHandler_NegativeDepth(t *testing.T) {
	handler := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/v1/graph/impact/service/test?depth=-5", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative depth, got %d", rec.Code)
	}
}
