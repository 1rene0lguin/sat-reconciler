package sat

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	// dir is .../internal/adapters/sat
	// We need to go up 3 levels to root to find "internal/adapters/sat/templates/..."
	rootDir := filepath.Join(dir, "../../..")
	if err := os.Chdir(rootDir); err != nil {
		// If we can't change dir, we might already be in root or environment is different.
		// We print but don't panic to allow other scenarios.
		println("Warning: Could not chdir to root:", err.Error())
	}
	os.Exit(m.Run())
}

// Helper to generate temporary key and cert files
func generateTestKeys(t *testing.T) (string, string) {
	t.Helper()

	// 1. Generate Private Key
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate private key: %v", err)
	}

	// 2. Create Certificate Template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Test Org"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	// 3. Create Certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privKey.PublicKey, privKey)
	if err != nil {
		t.Fatalf("Failed to create certificate: %v", err)
	}

	// 4. Write Private Key to Temp File
	keyFile, err := os.CreateTemp("", "key-*.pem")
	if err != nil {
		t.Fatalf("Failed to create key file: %v", err)
	}
	defer keyFile.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(privKey)
	if err := pem.Encode(keyFile, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		t.Fatalf("Failed to write key to file: %v", err)
	}

	// 5. Write Certificate to Temp File
	certFile, err := os.CreateTemp("", "cert-*.cer")
	if err != nil {
		t.Fatalf("Failed to create cert file: %v", err)
	}
	defer certFile.Close()

	if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		t.Fatalf("Failed to write cert to file: %v", err)
	}

	return keyFile.Name(), certFile.Name()
}

func TestNewRequestBuilder_AAA(t *testing.T) {
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	tests := []struct {
		name      string
		keyPath   string
		certPath  string
		wantError bool
	}{
		{
			name:      "Happy Path - Valid Key and Cert",
			keyPath:   keyPath,
			certPath:  certPath,
			wantError: false,
		},
		{
			name:      "Edge Case - Invalid Key Path",
			keyPath:   "invalid/path/key.pem",
			certPath:  certPath,
			wantError: true,
		},
		{
			name:      "Edge Case - Invalid Cert Path",
			keyPath:   keyPath,
			certPath:  "invalid/path/cert.cer",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cwd, _ := os.Getwd()
			t.Logf("Current Working Directory: %s", cwd)
			// ACT
			rb, err := NewRequestBuilder(tt.keyPath, tt.certPath)

			// ASSERT
			if tt.wantError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if rb == nil {
					t.Error("Expected RequestBuilder, got nil")
				}
			}
		})
	}
}

func TestBuildSignedRequest_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		t.Fatalf("Failed to create RequestBuilder: %v", err)
	}

	params := SoapRequestParams{
		RfcSolicitant: "TEST010101TST",
		DateStart:     "2023-01-01T00:00:00",
		DateEnd:       "2023-01-02T00:00:00",
		TypeRequest:   "Metadata",
		RfcIssuer:     "EMISOR010101",
		RfcReceiver:   "TEST010101TST",
	}

	// ACT
	xmlBytes, err := rb.BuildSignedRequest(params)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	xmlStr := string(xmlBytes)

	// Validamos que contenga elementos clave
	expectedTags := []string{
		"des:solicitud",
		"FechaInicio=\"2023-01-01T00:00:00\"",
		"RfcSolicitante=\"TEST010101TST\"",
		"<SignatureValue>",
		"<DigestValue>",
		"<X509Certificate>",
		"</des:solicitud>",
	}

	for _, tag := range expectedTags {
		if !strings.Contains(xmlStr, tag) {
			t.Errorf("XML missing expected content: %s", tag)
		}
	}
}

func TestBuildVerificationRequest_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		t.Fatalf("Failed to create RequestBuilder: %v", err)
	}

	// ACT
	xmlBytes, err := rb.BuildVerificationRequest("TEST010101TST", "UUID-1234")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	xmlStr := string(xmlBytes)

	if !strings.Contains(xmlStr, "IdSolicitud=\"UUID-1234\"") {
		t.Error("XML missing IdSolicitud")
	}
	if !strings.Contains(xmlStr, "RfcSolicitante=\"TEST010101TST\"") {
		t.Error("XML missing RfcSolicitante")
	}
}

func TestBuildDownloadRequest_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		t.Fatalf("Failed to create RequestBuilder: %v", err)
	}

	// ACT
	xmlBytes, err := rb.BuildDownloadRequest("TEST010101TST", "PKG-ID-123")

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	xmlStr := string(xmlBytes)

	// Attributes check
	if !strings.Contains(xmlStr, "IdPaquete=\"PKG-ID-123\"") {
		t.Error("XML missing IdPaquete")
	}
	if !strings.Contains(xmlStr, "RfcSolicitante=\"TEST010101TST\"") {
		t.Error("XML missing RfcSolicitante")
	}
	// Signature check
	if !strings.Contains(xmlStr, "<SignatureValue>") {
		t.Error("XML missing SignatureValue")
	}
}

func TestBuildAuthRequest_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		t.Fatalf("Failed to create RequestBuilder: %v", err)
	}

	// ACT
	xmlBytes, err := rb.BuildAuthRequest()

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	xmlStr := string(xmlBytes)

	// Security token check
	if !strings.Contains(xmlStr, "docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-secext-1.0.xsd") {
		t.Error("XML missing WSSecurity namespace or header")
	}
	if !strings.Contains(xmlStr, "<u:Created>") {
		t.Error("XML missing Created timestamp")
	}
	if !strings.Contains(xmlStr, "<u:Expires>") {
		t.Error("XML missing Expires timestamp")
	}
	if !strings.Contains(xmlStr, "<SignatureValue>") {
		t.Error("XML missing SignatureValue")
	}
}
