package collector

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// kubeConfig represents the subset of a kubeconfig file that we need.
type kubeConfig struct {
	CurrentContext string             `yaml:"current-context"`
	Clusters       []kubeNamedCluster `yaml:"clusters"`
	Contexts       []kubeNamedContext `yaml:"contexts"`
	Users          []kubeNamedUser    `yaml:"users"`
}

type kubeNamedCluster struct {
	Name    string      `yaml:"name"`
	Cluster kubeCluster `yaml:"cluster"`
}

type kubeCluster struct {
	Server                   string `yaml:"server"`
	CertificateAuthority     string `yaml:"certificate-authority"`
	CertificateAuthorityData string `yaml:"certificate-authority-data"`
	InsecureSkipTLSVerify    bool   `yaml:"insecure-skip-tls-verify"`
}

type kubeNamedContext struct {
	Name    string      `yaml:"name"`
	Context kubeContext `yaml:"context"`
}

type kubeContext struct {
	Cluster   string `yaml:"cluster"`
	User      string `yaml:"user"`
	Namespace string `yaml:"namespace"`
}

type kubeNamedUser struct {
	Name string   `yaml:"name"`
	User kubeUser `yaml:"user"`
}

type kubeUser struct {
	Token                 string `yaml:"token"`
	ClientCertificate     string `yaml:"client-certificate"`
	ClientCertificateData string `yaml:"client-certificate-data"`
	ClientKey             string `yaml:"client-key"`
	ClientKeyData         string `yaml:"client-key-data"`
}

// kubeTransport holds the resolved connection parameters for a Kubernetes API.
type kubeTransport struct {
	Server     string
	HTTPClient *http.Client
	Token      string
}

// resolveKubeconfig parses a kubeconfig file and builds an authenticated HTTP
// client for the given context. If contextName is empty, current-context is used.
func resolveKubeconfig(path, contextName string) (*kubeTransport, error) {
	expanded := expandHome(path)
	data, err := os.ReadFile(expanded)
	if err != nil {
		return nil, fmt.Errorf("read kubeconfig %q: %w", expanded, err)
	}

	var kc kubeConfig
	if err := yaml.Unmarshal(data, &kc); err != nil {
		return nil, fmt.Errorf("parse kubeconfig: %w", err)
	}

	if contextName == "" {
		contextName = kc.CurrentContext
	}
	if contextName == "" {
		return nil, fmt.Errorf("no context specified and current-context is empty")
	}

	// Find the named context.
	var ctx *kubeContext
	for i := range kc.Contexts {
		if kc.Contexts[i].Name == contextName {
			ctx = &kc.Contexts[i].Context
			break
		}
	}
	if ctx == nil {
		return nil, fmt.Errorf("context %q not found in kubeconfig", contextName)
	}

	// Find the cluster.
	var cluster *kubeCluster
	for i := range kc.Clusters {
		if kc.Clusters[i].Name == ctx.Cluster {
			cluster = &kc.Clusters[i].Cluster
			break
		}
	}
	if cluster == nil {
		return nil, fmt.Errorf("cluster %q not found in kubeconfig", ctx.Cluster)
	}

	// Find the user.
	var user *kubeUser
	for i := range kc.Users {
		if kc.Users[i].Name == ctx.User {
			user = &kc.Users[i].User
			break
		}
	}
	if user == nil {
		return nil, fmt.Errorf("user %q not found in kubeconfig", ctx.User)
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}

	// CA certificate.
	if cluster.CertificateAuthorityData != "" {
		ca, err := base64.StdEncoding.DecodeString(cluster.CertificateAuthorityData)
		if err != nil {
			return nil, fmt.Errorf("decode CA data: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		tlsCfg.RootCAs = pool
	} else if cluster.CertificateAuthority != "" {
		ca, err := os.ReadFile(expandHome(cluster.CertificateAuthority))
		if err != nil {
			return nil, fmt.Errorf("read CA file: %w", err)
		}
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		tlsCfg.RootCAs = pool
	}

	if cluster.InsecureSkipTLSVerify {
		tlsCfg.InsecureSkipVerify = true //nolint:gosec // user-configured
	}

	// Client certificate auth.
	certPEM, keyPEM, err := resolveClientCert(user)
	if err != nil {
		return nil, err
	}
	if certPEM != nil && keyPEM != nil {
		cert, err := tls.X509KeyPair(certPEM, keyPEM)
		if err != nil {
			return nil, fmt.Errorf("load client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	return &kubeTransport{
		Server: cluster.Server,
		Token:  user.Token,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
	}, nil
}

// resolveInCluster builds a kubeTransport from in-cluster service account credentials.
func resolveInCluster() (*kubeTransport, error) {
	host := os.Getenv("KUBERNETES_SERVICE_HOST")
	port := os.Getenv("KUBERNETES_SERVICE_PORT")
	if host == "" || port == "" {
		return nil, fmt.Errorf("not running in-cluster: KUBERNETES_SERVICE_HOST/PORT not set")
	}

	token, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		return nil, fmt.Errorf("read service account token: %w", err)
	}

	tlsCfg := &tls.Config{MinVersion: tls.VersionTLS12}
	ca, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/ca.crt")
	if err == nil {
		pool := x509.NewCertPool()
		pool.AppendCertsFromPEM(ca)
		tlsCfg.RootCAs = pool
	}

	return &kubeTransport{
		Server: fmt.Sprintf("https://%s:%s", host, port),
		Token:  string(token),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				TLSClientConfig: tlsCfg,
			},
		},
	}, nil
}

func resolveClientCert(u *kubeUser) (certPEM, keyPEM []byte, err error) {
	if u.ClientCertificateData != "" {
		certPEM, err = base64.StdEncoding.DecodeString(u.ClientCertificateData)
		if err != nil {
			return nil, nil, fmt.Errorf("decode client cert data: %w", err)
		}
	} else if u.ClientCertificate != "" {
		certPEM, err = os.ReadFile(expandHome(u.ClientCertificate))
		if err != nil {
			return nil, nil, fmt.Errorf("read client cert: %w", err)
		}
	}

	if u.ClientKeyData != "" {
		keyPEM, err = base64.StdEncoding.DecodeString(u.ClientKeyData)
		if err != nil {
			return nil, nil, fmt.Errorf("decode client key data: %w", err)
		}
	} else if u.ClientKey != "" {
		keyPEM, err = os.ReadFile(expandHome(u.ClientKey))
		if err != nil {
			return nil, nil, fmt.Errorf("read client key: %w", err)
		}
	}
	return certPEM, keyPEM, nil
}

// loadCACertPool reads a PEM-encoded CA certificate file and returns a cert pool.
func loadCACertPool(path string) (*x509.CertPool, error) {
	ca, err := os.ReadFile(expandHome(path))
	if err != nil {
		return nil, fmt.Errorf("read CA cert %q: %w", path, err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(ca) {
		return nil, fmt.Errorf("no valid certificates found in %q", path)
	}
	return pool, nil
}

// expandHome replaces a leading "~/" with the user's home directory.
func expandHome(path string) string {
	if len(path) < 2 || path[:2] != "~/" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	return filepath.Join(home, path[2:])
}
