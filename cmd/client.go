package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

const defaultAddr = "127.0.0.1:8080"

// APIClient is the shared HTTP client used by all CLI client commands.
// It mirrors Vault's pattern: honor INFRAGRAPH_ADDR env var, with a
// per-command --server flag as override.
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewAPIClient creates an APIClient targeting serverAddr (host:port).
func NewAPIClient(serverAddr string) *APIClient {
	return &APIClient{
		baseURL:    "http://" + serverAddr,
		token:      os.Getenv("INFRAGRAPH_TOKEN"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// defaultServerAddr returns the address of the running server.
// Priority: --server flag value → INFRAGRAPH_ADDR env var → "127.0.0.1:8080"
func defaultServerAddr() string {
	if addr := os.Getenv("INFRAGRAPH_ADDR"); addr != "" {
		return addr
	}
	return defaultAddr
}

// GetJSON performs a GET request and JSON-decodes the response body into v.
func (c *APIClient) GetJSON(path string, v any) error {
	req, err := http.NewRequest(http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.doJSON(req, v)
}

// separateArgs splits args into flag args (--flag [value]) and positional args.
// This allows users to put flags before or after positional arguments.
func separateArgs(args []string) (flags []string, positional []string) {
	i := 0
	for i < len(args) {
		arg := args[i]
		if len(arg) >= 2 && arg[0] == '-' && arg[1] == '-' {
			flags = append(flags, arg)
			// If the next element doesn't look like a flag, treat it as the flag's value.
			if i+1 < len(args) && !(len(args[i+1]) >= 2 && args[i+1][0] == '-' && args[i+1][1] == '-') {
				flags = append(flags, args[i+1])
				i++
			}
		} else {
			positional = append(positional, arg)
		}
		i++
	}
	return
}

// PostJSON performs a POST request with no body and JSON-decodes the response into v.
func (c *APIClient) PostJSON(path string, v any) error {
	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return c.doJSON(req, v)
}

// doJSON executes the request, checks for errors, and optionally decodes the response.
func (c *APIClient) doJSON(req *http.Request, v any) error {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errBody map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&errBody); err == nil {
			if msg, ok := errBody["error"]; ok {
				return fmt.Errorf("server error (%d): %s", resp.StatusCode, msg)
			}
		}
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	if v != nil {
		return json.NewDecoder(resp.Body).Decode(v)
	}
	return nil
}
