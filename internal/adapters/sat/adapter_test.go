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

	// Mock HTTP Client
	mockTripper := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) *http.Response {
			if req.URL.String() == urlSolicitud {
				// Successful response with UUID
				responseXML := `
				<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
					<s:Body>
						<SolicitaDescargaResponse xmlns="http://DescargaMasivaTerceros.sat.gob.mx">
							<SolicitaDescargaResult IdSolicitud="12345-ABCDE-67890" CodEstatus="5000" Mensaje="Solicitud Aceptada"/>
						</SolicitaDescargaResponse>
					</s:Body>
				</s:Envelope>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseXML)),
					Header:     make(http.Header),
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
	uuid, err := adapter.RequestMetadata("RFC", "2024-01-01T00:00:00", "2024-01-31T23:59:59", certPath, keyPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if uuid != "12345-ABCDE-67890" {
		t.Errorf("Expected UUID '12345-ABCDE-67890', got '%s'", uuid)
	}
}

func TestCheckStatus_AAA(t *testing.T) {
	// ARRANGE
	keyPath, certPath := generateTestKeys(t)
	defer os.Remove(keyPath)
	defer os.Remove(certPath)

	// Mock HTTP Client
	mockTripper := &MockRoundTripper{
		RoundTripFunc: func(req *http.Request) *http.Response {
			if req.URL.String() == urlVerifica {
				// Successful response with Finished status
				responseXML := `
				<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
					<s:Body>
						<VerificaSolicitudDescargaResponse xmlns="http://DescargaMasivaTerceros.sat.gob.mx">
							<VerificaSolicitudDescargaResult EstadoSolicitud="3" CodigoEstadoSolicitud="5000" Mensaje="Solicitud Terminada" NumeroCFDIs="10">
								<IdsPaquetes>
									<string>PKG-001</string>
									<string>PKG-002</string>
								</IdsPaquetes>
							</VerificaSolicitudDescargaResult>
						</VerificaSolicitudDescargaResponse>
					</s:Body>
				</s:Envelope>`
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(responseXML)),
					Header:     make(http.Header),
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
	res, err := adapter.CheckStatus("RFC123", "TEST-UUID-12345", certPath, keyPath)

	// ASSERT
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if res.Status != domain.StatusFinished {
		t.Errorf("Expected Status Finished (3), got %v", res.Status)
	}
	if res.UUID != "TEST-UUID-12345" {
		t.Errorf("Expected UUID 'TEST-UUID-12345', got '%s'", res.UUID)
	}
	if len(res.PackageIDs) != 2 {
		t.Errorf("Expected 2 package IDs, got %d", len(res.PackageIDs))
	}
}
