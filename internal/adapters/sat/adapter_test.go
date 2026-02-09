package sat

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
)

// MockRoundTripper allows mocking HTTP responses
type MockRoundTripper struct {
	RoundTripFunc func(req *http.Request) *http.Response
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.RoundTripFunc(req), nil
}

func TestDownloadPackage_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t) // Reusing helper from request_builder_test.go (same package)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	// Mock HTTP Client
	mockTripper := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) *http.Response {
			// 1. Authentication Request
			if req.URL.String() == urlAutenticacion {
				responseXML := `
				<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
					<s:Body>
						<AutenticaResponse xmlns="http://DescargaMasivaTerceros.sat.gob.mx">
							<AutenticaResult>MOCK_TOKEN</AutenticaResult>
						</AutenticaResponse>
					</s:Body>
				</s:Envelope>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseXML)),
					Header:     make(http.Header),
				}
			}

			// 2. Download Request
			if req.URL.String() == urlDescarga {
				// Validamos que el token se incluya
				authHeader := req.Header.Get("Authorization")
				expectedAuth := authPrefix + "MOCK_TOKEN" + authSuffix
				if authHeader != expectedAuth {
					return &http.Response{
						StatusCode: http.StatusUnauthorized,
						Body:       io.NopCloser(bytes.NewBufferString("Unauthorized")),
					}
				}

				// Respuesta Exitosa con Base64 dummy (Un zip vacío o texto)
				// Base64 de "CONTENIDO_ZIP_MOCK" -> "Q09OVEVOSURPX1pJUF9NT0NL"
				responseXML := `
				<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
					<s:Body>
						<h:RespuestaDescargaMasivaTercerosSalida xmlns:h="http://DescargaMasivaTerceros.sat.gob.mx">
							<header codEstatus="5000" mensaje="Solicitud Aceptada"/>
							<body>
								<Paquete>Q09OVEVOSURPX1pJUF9NT0NL</Paquete>
							</body>
						</h:RespuestaDescargaMasivaTercerosSalida>
					</s:Body>
				</s:Envelope>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseXML)),
				}
			}

			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString("Not Found")),
			}
		},
	}

	adapter := NewSoapAdapter()
	adapter.client.Transport = mockTripper // Inject Mock Transport

	// ACT
	content, err := adapter.DownloadPackage("RFC", "PKG-1", certPath, keyPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if string(content) != "CONTENIDO_ZIP_MOCK" {
		t.Errorf("Expected content 'CONTENIDO_ZIP_MOCK', got '%s'", string(content))
	}
}

func TestRequestMetadata_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	adapter := NewSoapAdapter()
	// No necesitamos mockear HTTP porque RequestMetadata aun no hace llamadas reales (segun analisis)
	// Pero si las hiciera en el futuro, aqui fallaria.
	// Dado el estado actual, probamos que genere el UUID simulado.

	// ACT
	uuid, err := adapter.RequestMetadata("RFC", "ini", "fin", certPath, keyPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if uuid == "" {
		t.Error("Expected UUID, got empty")
	}
}

func TestCheckStatus_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	adapter := NewSoapAdapter()

	// ACT
	res, err := adapter.CheckStatus("RFC", "UUID", certPath, keyPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res.Status != domain.StatusInProcess {
		t.Errorf("Expected Status InProcess, got %v", res.Status)
	}
}
