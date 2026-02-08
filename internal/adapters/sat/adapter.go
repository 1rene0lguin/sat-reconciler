package sat // <--- CAMBIO IMPORTANTE: Debe coincidir con request_builder.go

import (
	"fmt"

	// Asegúrate de que este import apunte a TU repo y carpeta correcta
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"
)

// SoapAdapter implementa la interfaz ports.SatGateway
// Al estar en el mismo paquete 'sat', ya "ve" a RequestBuilder sin importarlo.
type SoapAdapter struct {
	// Configuración futura (URL, Timeouts, etc.)
}

func NewSoapAdapter() *SoapAdapter {
	return &SoapAdapter{}
}

// CheckStatus verifica el estado de una solicitud (Implementa interfaz del puerto)
func (s *SoapAdapter) CheckStatus(rfc, uuid, certPath, keyPath string) (*domain.VerificationResult, error) {
	// Como estamos en el mismo package 'sat', podemos llamar a NewRequestBuilder directo
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return nil, fmt.Errorf("error iniciando builder: %w", err)
	}

	// Construir XML
	xmlBytes, err := rb.BuildVerificationRequest(rfc, uuid)
	if err != nil {
		return nil, fmt.Errorf("error construyendo request: %w", err)
	}

	// TODO: Aquí iría la llamada HTTP Real al SAT (client.Post...)
	// Por ahora simulamos una respuesta positiva para el MVP
	_ = xmlBytes

	// Retornamos un objeto de Dominio (limpio)
	return &domain.VerificationResult{
		UUID:    uuid,
		Status:  domain.StatusInProcess, // Simulamos estado 2
		Message: "Simulación: En Proceso (Respuesta del Adapter)",
	}, nil
}

// RequestMetadata solicita la descarga (Implementa interfaz del puerto)
func (s *SoapAdapter) RequestMetadata(rfc, start, end, certPath, keyPath string) (string, error) {
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return "", fmt.Errorf("error iniciando builder: %w", err)
	}

	// Preparamos los parámetros usando el struct definido en request_builder.go
	params := SoapRequestParams{
		RfcSolicitante: rfc,
		FechaInicio:    start,
		FechaFin:       end,
		TipoSolicitud:  "Metadata", // Regla de Negocio: MVP solo baja Metadata
	}

	// Construimos el XML firmado
	xmlBytes, err := rb.BuildSignedRequest(params)
	if err != nil {
		return "", fmt.Errorf("error firmando solicitud: %w", err)
	}

	// TODO: Enviar HTTP al SAT.
	fmt.Printf("--- XML GENERADO (Simulando Envío) ---\n%s\n", string(xmlBytes))

	return "UUID-SIMULADO-DESDE-ADAPTER", nil
}
