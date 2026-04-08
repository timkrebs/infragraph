package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

var testBinaryPath string

// TestMain builds the binary once and runs all tests against it.
func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "infragraph-test-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	binaryName := "infragraph"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}
	testBinaryPath = filepath.Join(tmpDir, binaryName)

	if out, err := exec.Command("go", "build", "-o", testBinaryPath, ".").CombinedOutput(); err != nil {
		panic("build failed: " + string(out))
	}

	os.Exit(m.Run())
}

// startServer launches the binary with the given args, waits for /health to
// respond on the given port, and returns the running *exec.Cmd.
// The caller must call stopServer when done.
func startServer(t *testing.T, port int, extraArgs ...string) *exec.Cmd {
	t.Helper()
	args := append([]string{"server", "start", "--port", fmt.Sprintf("%d", port)}, extraArgs...)
	cmd := exec.Command(testBinaryPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start server: %v", err)
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
		if err == nil {
			resp.Body.Close()
			return cmd
		}
		time.Sleep(50 * time.Millisecond)
	}
	cmd.Process.Kill()
	t.Fatalf("server did not become ready on port %d", port)
	return nil
}

// stopServer sends SIGTERM and waits for the process to exit.
func stopServer(t *testing.T, cmd *exec.Cmd) {
	t.Helper()
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Logf("signal error (process may have already exited): %v", err)
	}
	cmd.Wait()
}

func TestHelp(t *testing.T) {
	out, err := exec.Command(testBinaryPath, "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("--help exited with error: %v\noutput: %s", err, out)
	}
}

func TestServerStart_DefaultPort(t *testing.T) {
	cmd := startServer(t, 18080)
	defer stopServer(t, cmd)

	resp, err := http.Get("http://127.0.0.1:18080/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServerStart_CustomPort(t *testing.T) {
	cmd := startServer(t, 19090)
	defer stopServer(t, cmd)

	resp, err := http.Get("http://127.0.0.1:19090/health")
	if err != nil {
		t.Fatalf("GET /health on port 19090 failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServerStart_Config(t *testing.T) {
	hclContent := `
server {
  bind_addr = "127.0.0.1"
  port      = 17800
  log_level = "info"
}
store {
  path = "/tmp/graph.db"
}
`
	tmpFile, err := os.CreateTemp("", "*.hcl")
	if err != nil {
		t.Fatalf("failed to create temp hcl file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(hclContent); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}
	tmpFile.Close()

	// port in config is 17800; startServer passes --port which wins, so we
	// pass the config's port explicitly to reuse startServer's readiness poll.
	cmd := startServer(t, 17800, "--config", tmpFile.Name())
	defer stopServer(t, cmd)

	resp, err := http.Get("http://127.0.0.1:17800/health")
	if err != nil {
		t.Fatalf("GET /health failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}

func TestServerStart_Config_MissingFile(t *testing.T) {
	err := exec.Command(testBinaryPath, "server", "start", "--config", "/nonexistent/path.hcl").Run()
	if err == nil {
		t.Fatal("expected non-zero exit for missing config file")
	}
}

func TestServerStart_HelpFlag(t *testing.T) {
	out, err := exec.Command(testBinaryPath, "server", "start", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("server start --help exited with error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "Usage:") {
		t.Errorf("expected usage info in output, got: %s", out)
	}
}

func TestServerStart_UnknownArg(t *testing.T) {
	err := exec.Command(testBinaryPath, "server", "start", "--unknown").Run()
	if err == nil {
		t.Fatal("expected non-zero exit for unknown argument")
	}
}

func TestServerStop(t *testing.T) {
	// Start a real server, then stop it via the CLI command.
	cmd := startServer(t, 19191)

	out, err := exec.Command(testBinaryPath, "server", "stop", "--server", "127.0.0.1:19191").CombinedOutput()
	if err != nil {
		stopServer(t, cmd)
		t.Fatalf("server stop failed: %v\noutput: %s", err, out)
	}
	if !strings.Contains(string(out), "shutting down") {
		t.Errorf("expected 'shutting down' in output, got: %s", out)
	}
	cmd.Wait() // server should have exited cleanly after shutdown
}

func TestServerStart_Config_PortOverride(t *testing.T) {
	hclContent := `
server {
  bind_addr = "127.0.0.1"
  port      = 7800
  log_level = "info"
}
store {}
`
	tmpFile, err := os.CreateTemp("", "*.hcl")
	if err != nil {
		t.Fatalf("failed to create temp hcl file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(hclContent); err != nil {
		t.Fatalf("failed to write hcl file: %v", err)
	}
	tmpFile.Close()

	// --port after --config should override the port in the config file.
	cmd := startServer(t, 19292, "--config", tmpFile.Name(), "--port", "19292")
	defer stopServer(t, cmd)

	resp, err := http.Get("http://127.0.0.1:19292/health")
	if err != nil {
		t.Fatalf("GET /health on overridden port failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200, got %d", resp.StatusCode)
	}
}
