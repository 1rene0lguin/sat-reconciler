package sat

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
)

const ()

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

func (s *SoapAdapter) authenticate(certPath, keyPath string) (string, error) {
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return "", err
	}

	authXML, err := rb.BuildAuthRequest()
	if err != nil {
		return "", err
	}

	return s.doAuthRequest(authXML)
}

// doAuthRequest envía el sobre SOAP y extrae el token del body o headers.
func (s *SoapAdapter) doAuthRequest(xmlPayload []byte) (string, error) {
	req, err := http.NewRequest(http.MethodPost, urlAutenticacion, bytes.NewReader(xmlPayload))
	if err != nil {
		return "", fmt.Errorf("%s: %w", errAuthRequest, err)
	}

	// Headers requeridos por el SAT [cite: 35, 36]
	req.Header.Set("Content-Type", "text/xml; charset=utf-8")
	req.Header.Set("SOAPAction", actionAutentica)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errAuthRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error HTTP en autenticación: %d", resp.StatusCode)
	}

	// El SAT devuelve el token dentro del XML de respuesta (AutenticaResult).
	// Por simplicidad y performance (evitar struct gigante), lo extraemos directo.
	// En un refactor futuro, usar un struct XML Decoder es válido.
	return extractTokenFromResponse(resp.Body)
}

// extractTokenFromResponse busca el string del token en la respuesta.
func extractTokenFromResponse(body io.Reader) (string, error) {
	respBytes, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	// La respuesta suele ser: <AutenticaResult>WRAP access_token="..."</AutenticaResult>
	// Buscamos el contenido crudo.
	responseString := string(respBytes)

	// TODO: Mejorar esto con XML Unmarshal para ser más robusto.
	// Esta es una implementación rápida tipo "grep" para el MVP.
	// El token suele venir encoded, Go lo maneja bien como string opaco.
	if len(responseString) < 10 {
		return "", fmt.Errorf(errEmptyToken)
	}

	return responseString, nil
}
