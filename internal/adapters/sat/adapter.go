package sat // <--- CAMBIO IMPORTANTE: Debe coincidir con request_builder.go

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	// Asegúrate de que este import apunte a TU repo y carpeta correcta
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"
)

type SoapAdapter struct {
	// Configuración futura (URL, Timeouts, etc.)
}

type AutenticaResponseEnvelope struct {
	Body struct {
		AutenticaResponse struct {
			AutenticaResult string `xml:"AutenticaResult"` // Aquí viene el Token
		} `xml:"AutenticaResponse"`
	} `xml:"Body"`
}

type DescargaResponseEnvelope struct {
	Body struct {
		RespuestaDescargaMasiva struct {
			Paquete string `xml:"Paquete"` // Aquí viene el ZIP en Base64
		} `xml:"RespuestaDescargaMasivaTercerosResult"`
	} `xml:"Body"`
}

func NewSoapAdapter() *SoapAdapter {
	return &SoapAdapter{}
}

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

func (s *SoapAdapter) RequestMetadata(rfc, start, end, certPath, keyPath string) (string, error) {
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return "", fmt.Errorf("error iniciando builder: %w", err)
	}

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

func (s *SoapAdapter) DownloadPackage(rfc, packageID, certPath, keyPath string) ([]byte, error) {
	// 1. Preparar Builder
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return nil, fmt.Errorf("error builder: %w", err)
	}

	// 2. Construir XML Firmado (Usando tu template download_request.xml)
	xmlPayload, err := rb.BuildDownloadRequest(rfc, packageID)
	if err != nil {
		return nil, fmt.Errorf("error building request: %w", err)
	}

	// 3. Crear Request HTTP
	req, err := http.NewRequest("POST", urlDescarga, bytes.NewBuffer(xmlPayload))
	if err != nil {
		return nil, fmt.Errorf("error creating http request: %w", err)
	}

	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", soapActionDescarga)

	// 4. Enviar al SAT
	client := &http.Client{} // Podríamos inyectar esto para testing
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error downloading: %w", err)
	}
	defer resp.Body.Close()

	// 5. Leer Respuesta Raw
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	// 6. Parsear XML
	var envelope DownloadResponseEnvelope
	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("xml parse error: %w", err)
	}

	// 7. Validar Respuesta SAT
	satStatus := envelope.Body.Response.Header.CodEstatus
	if satStatus != "5000" { // 5000 = Éxito según documentación SAT [cite: 365, 366]
		return nil, fmt.Errorf("sat error: %s - %s", satStatus, envelope.Body.Response.Header.Mensaje)
	}

	// 8. Decodificar Base64 a ZIP (Bytes)
	zipBytes, err := base64.StdEncoding.DecodeString(envelope.Body.Response.Body.PaqueteBase64)
	if err != nil {
		return nil, fmt.Errorf("base64 decode error: %w", err)
	}

	return zipBytes, nil
}
