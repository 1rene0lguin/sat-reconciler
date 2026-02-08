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
					UUID:            "A1B2C3D4-1234-5678-90AB-CDEF12345678",
					RfcEmisor:       "XAXX010101000",
					NombreEmisor:    "EMPRESA DEMO SA DE CV",
					RfcReceptor:     "URE180429TM6",
					NombreReceptor:  "USUARIO RECEPTOR",
					Total:           1500.50,
					TipoComprobante: "I",
					Estatus:         "Vigente",
					// Las fechas las calculamos dinámicamente en el Assert para exactitud
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
					UUID:            "E5F6G7H8-1234-5678-90AB-CDEF12345678",
					Total:           500.00,
					TipoComprobante: "E",         // Egreso
					Estatus:         "Cancelado", // Mapeado de "0"
				},
			},
		},
		{
			name: "Edge Case - Empty Lines and Spaces",
			// ARRANGE
			inputString: `
            
			A1B2C3D4-LIMPIO~XAXX010101000~EMPRESA~URE180429TM6~RECEPTOR~PAC~15/01/2024 12:00:00~15/01/2024 12:00:00~100~I~1~
			   `, // Líneas vacías arriba y abajo
			wantError: false,
			expected: []sat.Metadata{
				{UUID: "A1B2C3D4-LIMPIO"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// ARRANGE (Setup del Reader)
			reader := strings.NewReader(tt.inputString)

			// ACT (Ejecución del SUT - System Under Test)
			got, err := parser.ParseMetadataTxt(reader)

			// ASSERT
			if tt.wantError {
				if err == nil {
					t.Errorf("ParseMetadataTxt() expected error, got nil")
				}
				return // Si esperamos error y llegó, terminamos
			}

			if err != nil {
				t.Fatalf("ParseMetadataTxt() unexpected error: %v", err)
			}

			if len(got) != len(tt.expected) {
				t.Fatalf("Expected %d records, got %d", len(tt.expected), len(got))
			}

			// Deep Verification del primer elemento (si existe)
			if len(got) > 0 {
				actual := got[0]
				expected := tt.expected[0]

				// Validamos campos clave
				if actual.UUID != expected.UUID {
					t.Errorf("UUID mismatch: got %s, want %s", actual.UUID, expected.UUID)
				}
				if actual.Total != expected.Total && expected.Total != 0 {
					t.Errorf("Total mismatch: got %f, want %f", actual.Total, expected.Total)
				}
				if actual.Estatus != expected.Estatus && expected.Estatus != "" {
					t.Errorf("Estatus mismatch: got %s, want %s", actual.Estatus, expected.Estatus)
				}

				// Validación Específica de Fechas (Lo difícil)
				// Solo validamos si esperábamos una fecha válida en el caso 1
				if tt.name == "Happy Path - Standard SAT File with Header" {
					expectedTime, _ := time.Parse(layoutSAT, "15/01/2024 12:00:00")
					if !actual.FechaEmision.Equal(expectedTime) {
						t.Errorf("FechaEmision mismatch: got %v, want %v", actual.FechaEmision, expectedTime)
					}
				}

				// Validación de Fecha Cancelación (Puntero)
				if tt.name == "Edge Case - Cancelled Invoice with Date" {
					if actual.FechaCancelacion == nil {
						t.Error("Expected FechaCancelacion to be not nil")
					} else {
						expectedCancelTime, _ := time.Parse(layoutSAT, "20/01/2024 10:00:00")
						if !actual.FechaCancelacion.Equal(expectedCancelTime) {
							t.Errorf("FechaCancelacion mismatch: got %v, want %v", actual.FechaCancelacion, expectedCancelTime)
						}
					}
				}
			}
		})
	}
}
