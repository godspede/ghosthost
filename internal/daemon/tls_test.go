// internal/daemon/tls_test.go
package daemon

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/config"
)

// generateTestCert produces a self-signed cert/key valid for 127.0.0.1 and
// writes both as PEM files into dir. Returns cert and key paths.
func generateTestCert(t *testing.T, dir string) (string, string) {
	t.Helper()
	priv, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "ghosthost-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")
	certOut, _ := os.Create(certPath)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: der})
	certOut.Close()
	keyBytes, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	keyOut, _ := os.Create(keyPath)
	pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: keyBytes})
	keyOut.Close()
	return certPath, keyPath
}

func TestCheckTLSFilesReadable(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCert(t, dir)

	if err := checkTLSFilesReadable(certPath, keyPath); err != nil {
		t.Fatalf("valid pair: unexpected error: %v", err)
	}

	missing := filepath.Join(dir, "does-not-exist.pem")
	err := checkTLSFilesReadable(missing, keyPath)
	if err == nil {
		t.Fatal("expected error for missing cert")
	}
	if !containsString(err.Error(), "does-not-exist.pem") {
		t.Fatalf("error should mention missing path: %v", err)
	}
}

func containsString(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestBuildURL_UsesHTTPSWhenTLSConfigured(t *testing.T) {
	core, _ := newCore(t)
	core.cfg.TLSCert = "/some/cert.pem"
	core.cfg.TLSKey = "/some/key.pem"
	got := core.buildURL("tok", "name.txt")
	if !containsString(got, "https://") {
		t.Fatalf("expected https:// scheme, got %s", got)
	}
}

func TestRun_TLS_StartAndFetch(t *testing.T) {
	dir := t.TempDir()
	certPath, keyPath := generateTestCert(t, dir)

	cfg := config.Config{
		Host:         "127.0.0.1",
		Bind:         "127.0.0.1",
		Port:         freePort(t),
		AdminPort:    freePort(t),
		DataDir:      dir,
		DefaultTTL:   time.Minute,
		IdleShutdown: 400 * time.Millisecond,
		TLSCert:      certPath,
		TLSKey:       keyPath,
	}

	done := make(chan error, 1)
	go func() { done <- Run(cfg) }()

	// Wait for the lockfile to confirm the daemon is up.
	lockPath := filepath.Join(dir, "daemon.lock")
	deadline := time.Now().Add(5 * time.Second)
	var meta Meta
	for {
		m, err := ReadLockfile(lockPath)
		if err == nil && m.Secret != "" {
			meta = m
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("daemon not ready")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Admin remains plain HTTP on loopback.
	statusReq, _ := http.NewRequest("GET", "http://127.0.0.1:"+strconv.Itoa(cfg.AdminPort)+"/status", nil)
	statusReq.Header.Set("Authorization", "Bearer "+meta.Secret)
	resp, err := http.DefaultClient.Do(statusReq)
	if err != nil {
		t.Fatalf("admin status: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("admin status code %d", resp.StatusCode)
	}

	// Public port must serve TLS.
	client := &http.Client{Transport: &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}}
	publicURL := "https://127.0.0.1:" + strconv.Itoa(cfg.Port) + "/nonexistent"
	pr, err := client.Get(publicURL)
	if err != nil {
		t.Fatalf("public TLS request failed: %v", err)
	}
	pr.Body.Close()
	if pr.StatusCode != 404 {
		t.Fatalf("expected 404 on unknown path, got %d", pr.StatusCode)
	}

	// Confirm plain HTTP does NOT succeed against the TLS listener. Go's
	// TLS server intercepts plain-HTTP requests and replies with 400 plus
	// a diagnostic body ("Client sent an HTTP request to an HTTPS
	// server."); older stacks just error. Either is fine as proof we're
	// really speaking TLS — what must not happen is a 2xx.
	plainClient := &http.Client{Timeout: 2 * time.Second}
	httpResp, err := plainClient.Get("http://127.0.0.1:" + strconv.Itoa(cfg.Port) + "/")
	if err == nil {
		code := httpResp.StatusCode
		httpResp.Body.Close()
		if code >= 200 && code < 300 {
			t.Fatalf("expected plain HTTP against TLS listener to fail or 4xx, got %d", code)
		}
	}

	// Drive an idle shutdown.
	select {
	case <-done:
	case <-time.After(8 * time.Second):
		t.Fatal("daemon did not exit on idle")
	}
}
