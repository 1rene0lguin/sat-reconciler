package sat

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"
)

// TestIntegrationSATErrorCodes tests handling of various SAT error responses
func TestIntegrationSATErrorCodes(t *testing.T) {
	tests := []struct {
		name          string
		satErrorCode  string
		satMessage    string
		uuidInResp    string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Success 5000",
			satErrorCode:  "5000",
			satMessage:    "Solicitud Aceptada",
			uuidInResp:    "SUCCESS-UUID-12345",
			expectError:   false,
			errorContains: "",
		},
		{
			name:          "Rate Limit Error 5003",
			satErrorCode:  "5003",
			satMessage:    "Servicio no disponible. Intentar más tarde",
			uuidInResp:    "",
			expectError:   true,
			errorContains: "5003",
		},
		{
			name:          "Authentication Error 5004",
			satErrorCode:  "5004",
			satMessage:    "Error de autenticación",
			uuidInResp:    "",
			expectError:   true,
			errorContains: "5004",
		},
		{
			name:          "No Data Found 5005",
			satErrorCode:  "5005",
			satMessage:    "No se encontraron datos",
			uuidInResp:    "",
			expectError:   true,
			errorContains: "5005",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			keyPath, certPath := generateTestKeys(t)
			defer os.Remove(keyPath)
			defer os.Remove(certPath)

			mockTripper := &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) *http.Response {
					if req.URL.String() == urlAutenticacion {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(`<s:Envelope><s:Body><AutenticaResponse><AutenticaResult>TOKEN</AutenticaResult></AutenticaResponse></s:Body></s:Envelope>`)),
						}
					}
					responseXML := `
					<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
						<s:Body>
							<SolicitaDescargaRecibidosResponse xmlns="http://DescargaMasivaTerceros.sat.gob.mx">
								<SolicitaDescargaRecibidosResult IdSolicitud="` + tt.uuidInResp + `" CodEstatus="` + tt.satErrorCode + `" Mensaje="` + tt.satMessage + `"/>
							</SolicitaDescargaRecibidosResponse>
						</s:Body>
					</s:Envelope>`
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(responseXML)),
						Header:     make(http.Header),
					}
				},
			}

			adapter := NewSoapAdapter()
			adapter.client.Transport = mockTripper

			// ACT
			_, err := adapter.RequestMetadata("RFC", "2024-01-01T00:00:00", "2024-01-31T23:59:59", "Recibidos", certPath, keyPath, "")

			// ASSERT
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for SAT code %s, got nil", tt.satErrorCode)
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error for SAT code %s: %v", tt.satErrorCode, err)
				}
			}
		})
	}
}

// TestIntegrationNetworkErrors tests handling of network-level errors
func TestIntegrationNetworkErrors(t *testing.T) {
	tests := []struct {
		name          string
		httpStatus    int
		responseBody  string
		expectError   bool
		errorContains string
	}{
		{
			name:          "HTTP 500 Internal Server Error",
			httpStatus:    http.StatusInternalServerError,
			responseBody:  "Internal Server Error",
			expectError:   true,
			errorContains: "request failed after",
		},
		{
			name:          "HTTP 503 Service Unavailable",
			httpStatus:    http.StatusServiceUnavailable,
			responseBody:  "Service Unavailable",
			expectError:   true,
			errorContains: "request failed after",
		},
		{
			name:          "HTTP 401 Unauthorized",
			httpStatus:    http.StatusUnauthorized,
			responseBody:  "Unauthorized",
			expectError:   true,
			errorContains: "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			keyPath, certPath := generateTestKeys(t)
			defer os.Remove(keyPath)
			defer os.Remove(certPath)

			mockTripper := &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) *http.Response {
					if req.URL.String() == urlAutenticacion {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(`<s:Envelope><s:Body><AutenticaResponse><AutenticaResult>TOKEN</AutenticaResult></AutenticaResponse></s:Body></s:Envelope>`)),
						}
					}
					return &http.Response{
						StatusCode: tt.httpStatus,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseBody)),
						Header:     make(http.Header),
					}
				},
			}

			adapter := NewSoapAdapter()
			adapter.client.Transport = mockTripper

			// ACT
			_, err := adapter.RequestMetadata("RFC", "2024-01-01T00:00:00", "2024-01-31T23:59:59", "Recibidos", certPath, keyPath, "")

			// ASSERT
			if tt.expectError {
				if err == nil {
					t.Fatalf("Expected error for HTTP status %d, got nil", tt.httpStatus)
				}
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			}
		})
	}
}

// TestIntegrationMalformedResponses tests handling of invalid XML responses
func TestIntegrationMalformedResponses(t *testing.T) {
	tests := []struct {
		name          string
		responseXML   string
		expectError   bool
		errorContains string
	}{
		{
			name:          "Invalid XML - Not well-formed",
			responseXML:   "<broken><xml>",
			expectError:   true,
			errorContains: "xml parsing error",
		},
		{
			name: "Missing Required Fields",
			responseXML: `
			<s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/">
				<s:Body>
					<SolicitaDescargaResponse xmlns="http://DescargaMasivaTerceros.sat.gob.mx">
						<SolicitaDescargaResult IdSolicitud="VALID-UUID" CodEstatus="" Mensaje=""/>
					</SolicitaDescargaResponse>
				</s:Body>
			</s:Envelope>`,
			expectError:   true,
			errorContains: "sat error",
		},
		{
			name:          "Empty Response Body",
			responseXML:   "",
			expectError:   true,
			errorContains: "xml parsing error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE
			keyPath, certPath := generateTestKeys(t)
			defer os.Remove(keyPath)
			defer os.Remove(certPath)

			mockTripper := &MockRoundTripper{
				RoundTripFunc: func(req *http.Request) *http.Response {
					if req.URL.String() == urlAutenticacion {
						return &http.Response{
							StatusCode: http.StatusOK,
							Body:       io.NopCloser(bytes.NewBufferString(`<s:Envelope><s:Body><AutenticaResponse><AutenticaResult>TOKEN</AutenticaResult></AutenticaResponse></s:Body></s:Envelope>`)),
						}
					}
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewBufferString(tt.responseXML)),
						Header:     make(http.Header),
					}
				},
			}

			adapter := NewSoapAdapter()
			adapter.client.Transport = mockTripper

			// ACT
			_, err := adapter.RequestMetadata("RFC", "2024-01-01T00:00:00", "2024-01-31T23:59:59", "Recibidos", certPath, keyPath, "")

			// ASSERT
			if tt.expectError {
				if err == nil {
					t.Fatal("Expected error for malformed response, got nil")
				}
				if !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain '%s', got '%s'", tt.errorContains, err.Error())
				}
			}
		})
	}
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && bytesContains([]byte(s), []byte(substr))))
}

func bytesContains(b, subslice []byte) bool {
	for i := 0; i <= len(b)-len(subslice); i++ {
		if string(b[i:i+len(subslice)]) == string(subslice) {
			return true
		}
	}
	return false
}
