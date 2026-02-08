package sat

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"

	"github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"
)

const (
	urlDescarga        = "https://cfdidescargamasiva.clouda.sat.gob.mx/PeticionDescargaMasivaService.svc"
	soapActionDescarga = "http://DescargaMasivaTerceros.sat.gob.mx/IPeticionDescargaMasivaService/PeticionDescargaMasiva"

	headerContentType = "Content-Type"
	headerAuth        = "Authorization"
	mimeTypeXML       = "text/xml; charset=utf-8"
	authPrefix        = "WRAP access_token=\""
	authSuffix        = "\""

	satStatusSuccess = "5000"
)

type SoapAdapter struct {
	client *http.Client
}

func NewSoapAdapter() *SoapAdapter {
	return &SoapAdapter{
		client: &http.Client{},
	}
}

func (s *SoapAdapter) setHeaders(req *http.Request, token string) {
	req.Header.Set(headerContentType, mimeTypeXML)
	req.Header.Set("SOAPAction", soapActionDescarga)
	req.Header.Set(headerAuth, authPrefix+token+authSuffix)
}

func (s *SoapAdapter) processDownloadResponse(body io.Reader) ([]byte, error) {
	respBytes, err := io.ReadAll(body)
	if err != nil {
		return nil, fmt.Errorf("reading body error: %w", err)
	}

	var envelope DownloadResponseEnvelope
	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("xml parsing error: %w", err)
	}

	if err := s.validateSatStatus(envelope.Body.Response.Header); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(envelope.Body.Response.Body.PaqueteBase64)
}

func (s *SoapAdapter) validateSatStatus(header DownloadHeader) error {
	if header.CodEstatus != satStatusSuccess {
		return fmt.Errorf("sat error %s: %s", header.CodEstatus, header.Mensaje)
	}
	return nil
}

// authenticate encapsula la lógica de obtención de token (WRAP).
// [TODO] Debes implementar esto usando el servicio de Autenticación del SAT [cite: 34]
func (s *SoapAdapter) authenticate(certPath, keyPath string) (string, error) {
	// Implementación real pendiente:
	// 1. Generar SOAP de Autentica
	// 2. Firmar con certificado
	// 3. Enviar a https://cfdidescargamasivasolicitud.clouda.sat.gob.mx/Autenticacion/Autenticacion.svc
	// 4. Retornar el token string

	// Para MVP/Simulación local retornamos un dummy validable
	return "MOCK_TOKEN_WRAP_ACCESS", nil
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

func (s *SoapAdapter) CheckStatus(rfc, uuid, certPath, keyPath string) (*domain.VerificationResult, error) {
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return nil, fmt.Errorf("error iniciando builder: %w", err)
	}

	xmlBytes, err := rb.BuildVerificationRequest(rfc, uuid)
	if err != nil {
		return nil, fmt.Errorf("error construyendo request: %w", err)
	}

	// TODO: Aquí iría la llamada HTTP Real al SAT (client.Post...)
	// Por ahora simulamos una respuesta positiva para el MVP
	_ = xmlBytes

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
	token, err := s.authenticate(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return nil, fmt.Errorf("builder initialization error: %w", err)
	}

	xmlPayload, err := rb.BuildDownloadRequest(rfc, packageID)
	if err != nil {
		return nil, fmt.Errorf("xml generation error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, urlDescarga, bytes.NewBuffer(xmlPayload))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	s.setHeaders(req, token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	return s.processDownloadResponse(resp.Body)
}
