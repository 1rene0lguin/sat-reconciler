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

const (
	timeoutClient = 30 * time.Second
)

type SoapAdapter struct {
	client *http.Client
}

func NewSoapAdapter() *SoapAdapter {
	return &SoapAdapter{
		client: &http.Client{
			Timeout: timeoutClient,
		},
	}
}

func (s *SoapAdapter) setHeaders(req *http.Request, token string) {
	req.Header.Set(headerContentType, mimeTypeXML)
	req.Header.Set("SOAPAction", actionDescarga)
	if token != "" {
		req.Header.Set(headerAuth, authPrefix+token+authSuffix)
	}
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

	if err := s.validateSatStatus(envelope.Body.Response.Header.CodeStatus, envelope.Body.Response.Header.Message); err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(envelope.Body.Response.Body.PackageBase64)
}

func (s *SoapAdapter) validateSatStatus(code, message string) error {
	if code != satStatusSuccess {
		return fmt.Errorf("sat error %s: %s", code, message)
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

	// Send HTTP POST to SAT
	req, err := http.NewRequest(http.MethodPost, urlVerifica, bytes.NewBuffer(xmlBytes))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set(headerContentType, mimeTypeXML)
	req.Header.Set("SOAPAction", actionVerifica)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Parse XML response
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response error: %w", err)
	}

	var envelope VerifyResponseEnvelope
	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return nil, fmt.Errorf("xml parsing error: %w", err)
	}

	result := envelope.Body.Response.Result

	// Validate SAT status
	if err := s.validateSatStatus(result.CodeStatusRequest, result.Message); err != nil {
		return nil, err
	}

	// Map SAT status to domain status
	var status domain.RequestStatus
	switch result.StatusRequest {
	case 1:
		status = domain.StatusAccepted
	case 2:
		status = domain.StatusInProcess
	case 3:
		status = domain.StatusFinished
	case 5:
		status = domain.StatusRejected
	default:
		status = domain.StatusInProcess
	}

	return &domain.VerificationResult{
		UUID:       uuid,
		Status:     status,
		Message:    result.Message,
		PackageIDs: result.Packages,
	}, nil
}

func (s *SoapAdapter) RequestMetadata(rfc, start, end, certPath, keyPath string) (string, error) {
	rb, err := NewRequestBuilder(keyPath, certPath)
	if err != nil {
		return "", fmt.Errorf("error iniciando builder: %w", err)
	}

	params := SoapRequestParams{
		RfcSolicitant: rfc,
		DateStart:     start,
		DateEnd:       end,
		TypeRequest:   "Metadata", // Business Rule: MVP only downloads Metadata
	}

	// Build signed XML
	xmlBytes, err := rb.BuildSignedRequest(params)
	if err != nil {
		return "", fmt.Errorf("error firmando solicitud: %w", err)
	}

	// Send HTTP POST to SAT
	req, err := http.NewRequest(http.MethodPost, urlSolicitud, bytes.NewBuffer(xmlBytes))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}

	req.Header.Set(headerContentType, mimeTypeXML)
	req.Header.Set("SOAPAction", actionSolicitud)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	// Parse XML response
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response error: %w", err)
	}

	var envelope RequestResponseEnvelope
	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return "", fmt.Errorf("xml parsing error: %w", err)
	}

	result := envelope.Body.Response.Result

	// Validate SAT status
	if err := s.validateSatStatus(result.CodeStatus, result.Message); err != nil {
		return "", err
	}

	if result.IDSolicitud == "" {
		return "", fmt.Errorf("empty UUID in SAT response")
	}

	return result.IDSolicitud, nil
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
	req.Header.Set(headerContentType, mimeTypeXML)
	req.Header.Set("SOAPAction", actionAutentica)

	resp, err := s.client.Do(req)
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

func extractTokenFromResponse(body io.Reader) (string, error) {
	respBytes, err := io.ReadAll(body)
	if err != nil {
		return "", fmt.Errorf("error reading body: %w", err)
	}

	var envelope AutenticaResponseEnvelope
	if err := xml.Unmarshal(respBytes, &envelope); err != nil {
		return "", fmt.Errorf("%s: %w", errAuthParse, err)
	}

	token := envelope.Body.AutenticaResponse.AutenticaResult
	if token == "" {
		return "", fmt.Errorf(errEmptyToken)
	}

	return token, nil
}
