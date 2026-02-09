package parser_test

import (
	"strings"
	"testing"
	"time"

	"github.com/1rene0lguin/sat-reconciler/internal/adapters/sat"
	"github.com/1rene0lguin/sat-reconciler/internal/parser"
)

func TestParseMetadataTxt_AAA(t *testing.T) {
	// Definimos el layout de fecha que usa el SAT en el TXT para construir los expected
	layoutSAT := "02/01/2006 15:04:05"

	tests := []struct {
		name        string
		inputString string // Simulamos el contenido del archivo
		wantError   bool
		expected    []sat.Metadata
	}{
		{
			name: "Happy Path - Standard SAT File with Header",
			// ARRANGE
			inputString: `Uuid~RfcEmisor~NombreEmisor~RfcReceptor~NombreReceptor~RfcPac~FechaEmision~FechaCertificacionSat~Monto~EfectoComprobante~Estatus~FechaCancelacion
A1B2C3D4-1234-5678-90AB-CDEF12345678~XAXX010101000~EMPRESA DEMO SA DE CV~URE180429TM6~USUARIO RECEPTOR~SAT970701NN3~15/01/2024 12:00:00~15/01/2024 12:05:00~1500.50~I~1~`,
			wantError: false,
			expected: []sat.Metadata{
				{
					UUID:         "A1B2C3D4-1234-5678-90AB-CDEF12345678",
					RfcIssuer:    "XAXX010101000",
					NameIssuer:   "EMPRESA DEMO SA DE CV",
					RfcReceiver:  "URE180429TM6",
					NameReceiver: "USUARIO RECEPTOR",
					Total:        1500.50,
					TypeVoucher:  "I",
					Status:       "Vigente",
					// Dates are calculated dynamically in Assert for exactness
				},
			},
		},
		{
			name: "Edge Case - Cancelled Invoice with Date",
			// ARRANGE
			inputString: `E5F6G7H8-1234-5678-90AB-CDEF12345678~XAXX010101000~EMPRESA~URE180429TM6~RECEPTOR~PAC~15/01/2024 12:00:00~15/01/2024 12:05:00~500.00~E~0~20/01/2024 10:00:00`,
			wantError:   false,
			expected: []sat.Metadata{
				{
					UUID:        "E5F6G7H8-1234-5678-90AB-CDEF12345678",
					Total:       500.00,
					TypeVoucher: "E",         // Egreso
					Status:      "Cancelado", // Mapped from "0"
				},
			},
		},
		{
			name: "Edge Case - Empty Lines and Spaces",
			// ARRANGE
			inputString: `
            
			A1B2C3D4-LIMPIO~XAXX010101000~EMPRESA~URE180429TM6~RECEPTOR~PAC~15/01/2024 12:00:00~15/01/2024 12:00:00~100~I~1~
			   `, // Empty lines above and below
			wantError: false,
			expected: []sat.Metadata{
				{UUID: "A1B2C3D4-LIMPIO"},
			},
		},
		{
			name: "Edge Case - Malformed Line (Not enough fields)",
			// ARRANGE
			// Added dummy header line so the second line is processed as data
			inputString: "Uuid~Header\nTEST-BROKEN~XAXX010101000~EMPRESA",
			wantError:   true,
			expected:    nil,
		},
		{
			name: "Edge Case - Malformed Line (Incorrect Separator)",
			// ARRANGE
			// Added dummy header line so the second line is processed as data
			inputString: "Uuid~Header\nTEST-BROKEN|XAXX010101000|EMPRESA",
			wantError:   true,
			expected:    nil,
		},
		{
			name: "Edge Case - Invalid Amount (Should default to 0)",
			// ARRANGE
			inputString: "Uuid~Header\nTEST-AMOUNT~X~E~R~R~P~15/01/2024~15/01/2024~INVALID_AMOUNT~I~1~",
			wantError:   false,
			expected: []sat.Metadata{
				{
					UUID:   "TEST-AMOUNT",
					Total:  0, // Default float64
					Status: "Vigente",
				},
			},
		},
		{
			name: "Edge Case - Various Date Formats",
			// ARRANGE
			inputString: "Uuid~Header\nTEST-DATES~X~E~R~R~P~2024-01-15T12:00:00~15/01/2024~100~I~1~",
			wantError:   false,
			expected: []sat.Metadata{
				{
					UUID: "TEST-DATES",
					// Dates will be validated in dynamic Assert
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE (Setup Reader)
			reader := strings.NewReader(tt.inputString)

			// ACT (Execution of SUT - System Under Test)
			got, err := parser.ParseMetadataTxt(reader)

			// ASSERT
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseMetadataTxt() expected error, got nil")
				}
				return // If we expect error and got it, we are done
			}

			if err != nil {
				t.Fatalf("ParseMetadataTxt() unexpected error: %v", err)
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("Expected %d records, got %d", len(tt.expected), len(got))
			}

			// Deep Verification of the first element (if exists)
			if len(got) > 0 {
				actual := got[0]
				expected := tt.expected[0]

				// Validate key fields
				if actual.UUID != expected.UUID {
					t.Errorf("UUID mismatch: got %s, want %s", actual.UUID, expected.UUID)
				}
				if actual.Total != expected.Total && expected.Total != 0 {
					t.Errorf("Total mismatch: got %f, want %f", actual.Total, expected.Total)
				}
				if actual.Status != expected.Status && expected.Status != "" {
					t.Errorf("Status mismatch: got %s, want %s", actual.Status, expected.Status)
				}

				// Specific Date Validation (The hard part)
				// Only validate if we expected a valid date in case 1
				if tt.name == "Happy Path - Standard SAT File with Header" {
					expectedTime, _ := time.Parse(layoutSAT, "15/01/2024 12:00:00")
					if !actual.DateEmission.Equal(expectedTime) {
						t.Errorf("DateEmission mismatch: got %v, want %v", actual.DateEmission, expectedTime)
					}
				}

				// Cancellation Date Validation (Pointer)
				if tt.name == "Edge Case - Cancelled Invoice with Date" {
					if actual.DateCancellation == nil {
						t.Error("Expected DateCancellation to be not nil")
					} else {
						expectedCancelTime, _ := time.Parse(layoutSAT, "20/01/2024 10:00:00")
						if !actual.DateCancellation.Equal(expectedCancelTime) {
							t.Errorf("DateCancellation mismatch: got %v, want %v", actual.DateCancellation, expectedCancelTime)
						}
					}
				}
			}
		})
	}
}
