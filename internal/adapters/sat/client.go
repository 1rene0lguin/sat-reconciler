package sat

import (
	"fmt"
	// AJUSTA ESTO SEGÚN TU GO.MOD (con o sin la 'i' extra)
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"
)

// SoapAdapter implementa la comunicación con el WebService del SAT.
type SoapAdapter struct {
	// Configuración futura (URLs, Timeouts)
}

// NewSoapAdapter crea una nueva instancia del adaptador.
func NewSoapAdapter() *SoapAdapter {
	return &SoapAdapter{}
}

// CheckStatus consulta el estado de una solicitud de descarga previa.
func (s *SoapAdapter) CheckStatus(rfc, uuid, certPath, keyPath string) (*domain.VerificationResult, error) {
	// Al estar en el mismo package 'sat', ya reconoce NewRequestBuilder sin importar nada
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return nil, fmt.Errorf("error iniciando builder: %w", err)
	}

	xmlBytes, err := rb.BuildVerificationRequest(rfc, uuid)
	if err != nil {
		return nil, fmt.Errorf("error construyendo request: %w", err)
	}

	// TODO: Implementar cliente HTTP SOAP real.
	_ = xmlBytes

	return &domain.VerificationResult{
		UUID:    uuid,
		Status:  domain.StatusInProcess,
		Message: "Simulación: En Proceso",
	}, nil
}

// RequestMetadata solicita la descarga de metadatos (XMLs ligeros) al SAT.
func (s *SoapAdapter) RequestMetadata(rfc, start, end, certPath, keyPath string) (string, error) {
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return "", fmt.Errorf("error iniciando builder: %w", err)
	}

	params := SoapRequestParams{
		RfcSolicitante: rfc,
		FechaInicio:    start,
		FechaFin:       end,
		TipoSolicitud:  "Metadata",
	}

	xmlBytes, err := rb.BuildSignedRequest(params)
	if err != nil {
		return "", fmt.Errorf("error firmando solicitud: %w", err)
	}

	fmt.Printf("--- XML GENERADO (Simulando Envío) ---\n%s\n", string(xmlBytes))

	return "UUID-SIMULADO-12345", nil
}
