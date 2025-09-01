package https

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"nithronos/backend/nosd/internal/fsatomic"
)

// Config represents HTTPS/TLS configuration
type Config struct {
	Enabled      bool         `json:"enabled"`
	Port         int          `json:"port"`
	Certificate  *Certificate `json:"certificate,omitempty"`
	LetsEncrypt  *LEConfig    `json:"letsencrypt,omitempty"`
	RedirectHTTP bool         `json:"redirectHTTP"`
	HSTS         bool         `json:"hsts"`
	LastUpdated  time.Time    `json:"lastUpdated"`
}

// Certificate represents TLS certificate information
type Certificate struct {
	Subject     string    `json:"subject"`
	Issuer      string    `json:"issuer"`
	ValidFrom   time.Time `json:"validFrom"`
	ValidTo     time.Time `json:"validTo"`
	SelfSigned  bool      `json:"selfSigned"`
	Fingerprint string    `json:"fingerprint"`
	SANs        []string  `json:"sans,omitempty"`
}

// LEConfig represents Let's Encrypt configuration
type LEConfig struct {
	Enabled    bool     `json:"enabled"`
	Email      string   `json:"email"`
	Domains    []string `json:"domains"`
	Staging    bool     `json:"staging"`
	LastRenew  *time.Time `json:"lastRenew,omitempty"`
	NextRenew  *time.Time `json:"nextRenew,omitempty"`
}

// Manager manages HTTPS/TLS configuration
type Manager struct {
	storePath   string
	config      *Config
	certPath    string
	keyPath     string
	caddyConfig string
	mu          sync.RWMutex
}

// NewManager creates a new HTTPS manager
func NewManager(storePath string) (*Manager, error) {
	m := &Manager{
		storePath:   storePath,
		certPath:    "/etc/caddy/certs/server.crt",
		keyPath:     "/etc/caddy/certs/server.key",
		caddyConfig: "/etc/caddy/Caddyfile",
	}
	
	// Ensure directories exist
	if err := os.MkdirAll("/etc/caddy/certs", 0755); err != nil {
		return nil, fmt.Errorf("failed to create certs directory: %w", err)
	}
	
	if err := os.MkdirAll(storePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create store directory: %w", err)
	}
	
	// Load existing configuration
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	
	// Initialize config if not exists
	if m.config == nil {
		m.initializeConfig()
	}
	
	// Check current certificate
	m.checkCertificate()
	
	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	configPath := filepath.Join(m.storePath, "https_config.json")
	var config Config
	if ok, err := fsatomic.LoadJSON(configPath, &config); err != nil {
		return err
	} else if ok {
		m.config = &config
	}
	
	return nil
}

func (m *Manager) save() error {
	configPath := filepath.Join(m.storePath, "https_config.json")
	return fsatomic.SaveJSON(context.Background(), configPath, m.config, 0600)
}

func (m *Manager) initializeConfig() {
	m.config = &Config{
		Enabled:      false,
		Port:         443,
		RedirectHTTP: true,
		HSTS:         true,
		LastUpdated:  time.Now(),
	}
	
	_ = m.save()
}

func (m *Manager) checkCertificate() {
	// Check if certificate exists
	if _, err := os.Stat(m.certPath); err != nil {
		return
	}
	
	// Load and parse certificate
	certPEM, err := os.ReadFile(m.certPath)
	if err != nil {
		return
	}
	
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return
	}
	
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return
	}
	
	// Update certificate info
	m.config.Certificate = &Certificate{
		Subject:     cert.Subject.String(),
		Issuer:      cert.Issuer.String(),
		ValidFrom:   cert.NotBefore,
		ValidTo:     cert.NotAfter,
		SelfSigned:  cert.Issuer.String() == cert.Subject.String(),
		Fingerprint: fmt.Sprintf("%x", cert.Signature),
		SANs:        cert.DNSNames,
	}
}

// GetConfig returns the HTTPS configuration
func (m *Manager) GetConfig() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.config
}

// SetEnabled enables or disables HTTPS
func (m *Manager) SetEnabled(enabled bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if enabled {
		// Check if certificate exists
		if _, err := os.Stat(m.certPath); err != nil {
			// Generate self-signed certificate if none exists
			if err := m.generateSelfSignedCert(); err != nil {
				return fmt.Errorf("failed to generate certificate: %w", err)
			}
		}
		
		// Apply Caddy configuration
		if err := m.applyCaddyConfig(); err != nil {
			return err
		}
		
		// Reload Caddy
		cmd := exec.Command("systemctl", "reload", "caddy")
		if err := cmd.Run(); err != nil {
			// Try to start if not running
			cmd = exec.Command("systemctl", "start", "caddy")
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to start Caddy: %w", err)
			}
		}
	} else {
		// Update Caddy config to HTTP only
		if err := m.applyCaddyConfigHTTPOnly(); err != nil {
			return err
		}
		
		// Reload Caddy
		cmd := exec.Command("systemctl", "reload", "caddy")
		_ = cmd.Run()
	}
	
	m.config.Enabled = enabled
	m.config.LastUpdated = time.Now()
	return m.save()
}

// GenerateSelfSignedCert generates a self-signed certificate
func (m *Manager) GenerateSelfSignedCert() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	return m.generateSelfSignedCert()
}

func (m *Manager) generateSelfSignedCert() error {
	// Generate RSA key
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}
	
	// Get hostname and IPs
	hostname, _ := os.Hostname()
	ips := []net.IP{net.IPv4(127, 0, 0, 1)}
	
	// Get all network interfaces
	ifaces, err := net.Interfaces()
	if err == nil {
		for _, iface := range ifaces {
			addrs, err := iface.Addrs()
			if err != nil {
				continue
			}
			for _, addr := range addrs {
				if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
					if ipnet.IP.To4() != nil {
						ips = append(ips, ipnet.IP)
					}
				}
			}
		}
	}
	
	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"NithronOS"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           ips,
		DNSNames:              []string{hostname, "localhost"},
	}
	
	// Generate certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &priv.PublicKey, priv)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}
	
	// Encode certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certDER,
	})
	
	// Encode private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(priv),
	})
	
	// Write certificate
	if err := os.WriteFile(m.certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}
	
	// Write private key
	if err := os.WriteFile(m.keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	
	// Update certificate info
	m.checkCertificate()
	
	log.Info().Msg("Generated self-signed certificate")
	
	return nil
}

// UploadCertificate uploads a certificate or key file
func (m *Manager) UploadCertificate(fileType string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	switch fileType {
	case "cert", "certificate":
		// Validate certificate
		block, _ := pem.Decode(data)
		if block == nil {
			return fmt.Errorf("invalid PEM data")
		}
		
		if _, err := x509.ParseCertificate(block.Bytes); err != nil {
			return fmt.Errorf("invalid certificate: %w", err)
		}
		
		// Write certificate
		if err := os.WriteFile(m.certPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write certificate: %w", err)
		}
		
		// Update certificate info
		m.checkCertificate()
		
	case "key", "privatekey":
		// Validate private key
		block, _ := pem.Decode(data)
		if block == nil {
			return fmt.Errorf("invalid PEM data")
		}
		
		// Try to parse as various key types
		var keyErr error
		switch block.Type {
		case "RSA PRIVATE KEY":
			_, keyErr = x509.ParsePKCS1PrivateKey(block.Bytes)
		case "PRIVATE KEY":
			_, keyErr = x509.ParsePKCS8PrivateKey(block.Bytes)
		case "EC PRIVATE KEY":
			_, keyErr = x509.ParseECPrivateKey(block.Bytes)
		default:
			keyErr = fmt.Errorf("unknown key type: %s", block.Type)
		}
		
		if keyErr != nil {
			return fmt.Errorf("invalid private key: %w", keyErr)
		}
		
		// Write private key
		if err := os.WriteFile(m.keyPath, data, 0600); err != nil {
			return fmt.Errorf("failed to write private key: %w", err)
		}
		
	default:
		return fmt.Errorf("unknown file type: %s", fileType)
	}
	
	// Apply configuration if HTTPS is enabled
	if m.config.Enabled {
		if err := m.applyCaddyConfig(); err != nil {
			return err
		}
		
		// Reload Caddy
		cmd := exec.Command("systemctl", "reload", "caddy")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to reload Caddy: %w", err)
		}
	}
	
	m.config.LastUpdated = time.Now()
	return m.save()
}

// GenerateCSR generates a certificate signing request
func (m *Manager) GenerateCSR(domains []string) ([]byte, error) {
	// Generate RSA key if not exists
	var priv *rsa.PrivateKey
	
	if keyPEM, err := os.ReadFile(m.keyPath); err == nil {
		block, _ := pem.Decode(keyPEM)
		if block != nil {
			priv, _ = x509.ParsePKCS1PrivateKey(block.Bytes)
		}
	}
	
	if priv == nil {
		var err error
		priv, err = rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			return nil, fmt.Errorf("failed to generate private key: %w", err)
		}
		
		// Save private key
		keyPEM := pem.EncodeToMemory(&pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: x509.MarshalPKCS1PrivateKey(priv),
		})
		
		if err := os.WriteFile(m.keyPath, keyPEM, 0600); err != nil {
			return nil, fmt.Errorf("failed to write private key: %w", err)
		}
	}
	
	// Create CSR template
	template := x509.CertificateRequest{
		Subject: pkix.Name{
			Organization:  []string{"NithronOS"},
			Country:       []string{"US"},
		},
		DNSNames: domains,
	}
	
	// Generate CSR
	csrDER, err := x509.CreateCertificateRequest(rand.Reader, &template, priv)
	if err != nil {
		return nil, fmt.Errorf("failed to create CSR: %w", err)
	}
	
	// Encode CSR
	csrPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE REQUEST",
		Bytes: csrDER,
	})
	
	return csrPEM, nil
}

// ConfigureLetsEncrypt configures Let's Encrypt
func (m *Manager) ConfigureLetsEncrypt(email string, domains []string, staging bool) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.config.LetsEncrypt = &LEConfig{
		Enabled: true,
		Email:   email,
		Domains: domains,
		Staging: staging,
	}
	
	m.config.LastUpdated = time.Now()
	
	if err := m.save(); err != nil {
		return err
	}
	
	// Apply Caddy configuration with Let's Encrypt
	if err := m.applyCaddyConfigLE(); err != nil {
		return err
	}
	
	// Reload Caddy
	cmd := exec.Command("systemctl", "reload", "caddy")
	if err := cmd.Run(); err != nil {
		// Try to start if not running
		cmd = exec.Command("systemctl", "start", "caddy")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to start Caddy: %w", err)
		}
	}
	
	return nil
}

// applyCaddyConfig generates and applies Caddy configuration
func (m *Manager) applyCaddyConfig() error {
	var buf bytes.Buffer
	
	// Global options
	buf.WriteString("{\n")
	buf.WriteString("    admin off\n")
	buf.WriteString("    auto_https off\n")
	buf.WriteString("}\n\n")
	
	// HTTP server (redirect to HTTPS)
	if m.config.RedirectHTTP {
		buf.WriteString(":80 {\n")
		buf.WriteString("    redir https://{host}{uri} permanent\n")
		buf.WriteString("}\n\n")
	}
	
	// HTTPS server
	buf.WriteString(fmt.Sprintf(":%d {\n", m.config.Port))
	buf.WriteString(fmt.Sprintf("    tls %s %s\n", m.certPath, m.keyPath))
	
	if m.config.HSTS {
		buf.WriteString("    header Strict-Transport-Security \"max-age=31536000; includeSubDomains\"\n")
	}
	
	// Reverse proxy to backend
	buf.WriteString("    reverse_proxy localhost:8080 {\n")
	buf.WriteString("        header_up X-Real-IP {remote_host}\n")
	buf.WriteString("        header_up X-Forwarded-For {remote_host}\n")
	buf.WriteString("        header_up X-Forwarded-Proto {scheme}\n")
	buf.WriteString("    }\n")
	
	buf.WriteString("}\n")
	
	// Write configuration
	return os.WriteFile(m.caddyConfig, buf.Bytes(), 0644)
}

// applyCaddyConfigHTTPOnly generates HTTP-only Caddy configuration
func (m *Manager) applyCaddyConfigHTTPOnly() error {
	var buf bytes.Buffer
	
	// Global options
	buf.WriteString("{\n")
	buf.WriteString("    admin off\n")
	buf.WriteString("    auto_https off\n")
	buf.WriteString("}\n\n")
	
	// HTTP server
	buf.WriteString(":80 {\n")
	buf.WriteString("    reverse_proxy localhost:8080 {\n")
	buf.WriteString("        header_up X-Real-IP {remote_host}\n")
	buf.WriteString("        header_up X-Forwarded-For {remote_host}\n")
	buf.WriteString("        header_up X-Forwarded-Proto {scheme}\n")
	buf.WriteString("    }\n")
	buf.WriteString("}\n")
	
	// Write configuration
	return os.WriteFile(m.caddyConfig, buf.Bytes(), 0644)
}

// applyCaddyConfigLE generates Caddy configuration with Let's Encrypt
func (m *Manager) applyCaddyConfigLE() error {
	if m.config.LetsEncrypt == nil {
		return fmt.Errorf("let's encrypt not configured")
	}
	
	var buf bytes.Buffer
	
	// Global options
	buf.WriteString("{\n")
	buf.WriteString("    admin off\n")
	buf.WriteString(fmt.Sprintf("    email %s\n", m.config.LetsEncrypt.Email))
	
	if m.config.LetsEncrypt.Staging {
		buf.WriteString("    acme_ca https://acme-staging-v02.api.letsencrypt.org/directory\n")
	}
	
	buf.WriteString("}\n\n")
	
	// HTTPS server with automatic certificates
	domains := strings.Join(m.config.LetsEncrypt.Domains, ", ")
	buf.WriteString(fmt.Sprintf("%s {\n", domains))
	
	if m.config.HSTS {
		buf.WriteString("    header Strict-Transport-Security \"max-age=31536000; includeSubDomains\"\n")
	}
	
	// Reverse proxy to backend
	buf.WriteString("    reverse_proxy localhost:8080 {\n")
	buf.WriteString("        header_up X-Real-IP {remote_host}\n")
	buf.WriteString("        header_up X-Forwarded-For {remote_host}\n")
	buf.WriteString("        header_up X-Forwarded-Proto {scheme}\n")
	buf.WriteString("    }\n")
	
	buf.WriteString("}\n")
	
	// Write configuration
	return os.WriteFile(m.caddyConfig, buf.Bytes(), 0644)
}

// TestConfiguration tests the current HTTPS configuration
func (m *Manager) TestConfiguration() error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Check if certificate and key exist
	if _, err := os.Stat(m.certPath); err != nil {
		return fmt.Errorf("certificate not found: %w", err)
	}
	
	if _, err := os.Stat(m.keyPath); err != nil {
		return fmt.Errorf("private key not found: %w", err)
	}
	
	// Try to load certificate and key as TLS pair
	_, err := tls.LoadX509KeyPair(m.certPath, m.keyPath)
	if err != nil {
		return fmt.Errorf("invalid certificate/key pair: %w", err)
	}
	
	// Test Caddy configuration
	cmd := exec.Command("caddy", "validate", "--config", m.caddyConfig)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("invalid Caddy configuration: %s", string(output))
	}
	
	return nil
}

// GetCertificateInfo returns detailed certificate information
func (m *Manager) GetCertificateInfo() (*Certificate, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.config.Certificate == nil {
		return nil, fmt.Errorf("no certificate configured")
	}
	
	return m.config.Certificate, nil
}

// RenewCertificate attempts to renew the certificate
func (m *Manager) RenewCertificate() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if m.config.LetsEncrypt != nil && m.config.LetsEncrypt.Enabled {
		// Caddy handles Let's Encrypt renewal automatically
		// Just reload to trigger renewal check
		cmd := exec.Command("systemctl", "reload", "caddy")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to reload Caddy: %w", err)
		}
		
		now := time.Now()
		m.config.LetsEncrypt.LastRenew = &now
		nextRenew := now.Add(60 * 24 * time.Hour) // 60 days
		m.config.LetsEncrypt.NextRenew = &nextRenew
		
		return m.save()
	}
	
	return fmt.Errorf("automatic renewal only available with Let's Encrypt")
}

// ExportCertificate exports the current certificate
func (m *Manager) ExportCertificate(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	cert, err := os.ReadFile(m.certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %w", err)
	}
	
	_, err = w.Write(cert)
	return err
}

// ExportPrivateKey exports the current private key
func (m *Manager) ExportPrivateKey(w io.Writer) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	key, err := os.ReadFile(m.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %w", err)
	}
	
	_, err = w.Write(key)
	return err
}
