package sat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
)

type SoapAdapter struct {
	client      *http.Client
	config      AdapterConfig
	rateLimiter *RateLimiter
	cache       *VerificationCache
}

// NewSoapAdapter creates adapter with default production configuration
func NewSoapAdapter() *SoapAdapter {
	return NewSoapAdapterWithConfig(DefaultConfig())
}

// NewSoapAdapterWithConfig creates adapter with custom configuration
func NewSoapAdapterWithConfig(config AdapterConfig) *SoapAdapter {
	return &SoapAdapter{
		client: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		config:      config,
		rateLimiter: NewRateLimiter(config.RequestsPerMinute, config.BurstSize, config.RateLimitEnabled),
		cache:       NewVerificationCache(config.CacheTTL, config.MaxCacheSize, config.CacheEnabled),
	}
}

func (s *SoapAdapter) setHeaders(req *http.Request, action, token string) {
	req.Header.Set(headerContentType, mimeTypeXML)
	// Some WCF configurations require quotes around SOAPAction
	req.Header.Set("SOAPAction", `"`+action+`"`)
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

func (s *SoapAdapter) CheckStatus(rfc, uuid, certPath, keyPath, password string) (*domain.VerificationResult, error) {
	// Sanitize RFC and UUID
	rfc = strings.ToUpper(strings.TrimSpace(rfc))
	uuid = strings.TrimSpace(uuid)

	// Check cache first
	if cachedResult, found := s.cache.Get(rfc, uuid); found {
		logCacheHit(s.config.Logger, "CheckStatus", uuid)
		return cachedResult, nil
	}
	logCacheMiss(s.config.Logger, "CheckStatus", uuid)

	token, err := s.authenticate(certPath, keyPath, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	rb, err := NewRequestBuilder(keyPath, certPath, password)
	if err != nil {
		return nil, fmt.Errorf("error iniciando builder: %w", err)
	}

	xmlBytes, err := rb.BuildVerificationRequest(rfc, uuid)
	if err != nil {
		return nil, fmt.Errorf("error construyendo request: %w", err)
	}

	// Apply rate limiting
	ctx := context.Background()
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limit error: %w", err)
	}

	// Send HTTP POST to SAT
	req, err := http.NewRequest(http.MethodPost, urlVerifica, bytes.NewReader(xmlBytes))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	s.setHeaders(req, actionVerifica, token)

	// Log request
	logHTTPRequest(s.config.Logger, "CheckStatus", urlVerifica, uuid)

	// Perform request with retry if enabled
	var resp *http.Response
	if s.config.RetryEnabled {
		resp, err = s.doRequestWithRetry(ctx, req, "CheckStatus", uuid)
	} else {
		start := time.Now()
		resp, err = s.client.Do(req)
		logHTTPResponse(s.config.Logger, "CheckStatus", uuid, func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}(), time.Since(start), err)
	}

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
	if err := s.validateSatStatus(result.CodeStatusRequest, result.Message); err != nil && result.CodeStatusRequest != "5004" {
		logSATError(s.config.Logger, "CheckStatus", uuid, result.CodeStatusRequest, result.Message)
		return nil, err
	}

	// Map SAT status to domain status
	var status domain.RequestStatus
	if result.CodeStatusRequest == "5004" {
		// 5004 means "No data found". The request is effectively finished with zero results.
		status = domain.StatusFinished
	} else {
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
	}

	verification := &domain.VerificationResult{
		UUID:       uuid,
		Status:     status,
		Message:    result.Message,
		PackageIDs: result.Packages,
	}

	// Store in cache
	s.cache.Set(rfc, uuid, verification)

	return verification, nil
}

func (s *SoapAdapter) RequestMetadata(rfc, start, end, downloadType, certPath, keyPath, password string) (string, error) {
	token, err := s.authenticate(certPath, keyPath, password)
	if err != nil {
		return "", fmt.Errorf("authentication failed: %w", err)
	}

	rb, err := NewRequestBuilder(keyPath, certPath, password)
	if err != nil {
		return "", fmt.Errorf("error iniciando builder: %w", err)
	}

	// Sanitizar RFC
	rfc = strings.ToUpper(strings.TrimSpace(rfc))

	// El SAT requiere que el formato XML contenga segundos explícitos: "2006-01-02T15:04:05"
	// Si el HTML5 datetime-local envía "YYYY-MM-DDTHH:MM", le anexamos los segundos
	if len(start) == 16 {
		start += ":00"
	}
	if len(end) == 16 {
		end += ":00"
	}

	params := SoapRequestParams{
		RfcSolicitant: rfc,
		DateStart:     start,
		DateEnd:       end,
		TypeRequest:   "Metadata", // Business Rule: MVP only downloads Metadata
		DownloadType:  downloadType,
	}

	// Build signed XML
	xmlBytes, err := rb.BuildSignedRequest(params)
	if err != nil {
		return "", fmt.Errorf("error firmando solicitud: %w", err)
	}

	// Apply rate limiting
	ctx := context.Background()
	if err := s.rateLimiter.Wait(ctx); err != nil {
		return "", fmt.Errorf("rate limit error: %w", err)
	}

	// Send HTTP POST to SAT
	req, err := http.NewRequest(http.MethodPost, urlSolicitud, bytes.NewReader(xmlBytes))
	if err != nil {
		return "", fmt.Errorf("request creation error: %w", err)
	}

	action := actionSolicitudEmitidos
	if downloadType == "Recibidos" {
		action = actionSolicitudRecibidos
	}

	s.setHeaders(req, action, token)

	// Log request
	logHTTPRequest(s.config.Logger, "RequestMetadata", urlSolicitud, "new-request")

	// Perform request with retry if enabled
	var resp *http.Response
	if s.config.RetryEnabled {
		resp, err = s.doRequestWithRetry(ctx, req, "RequestMetadata", "new-request")
	} else {
		start := time.Now()
		resp, err = s.client.Do(req)
		logHTTPResponse(s.config.Logger, "RequestMetadata", "new-request", func() int {
			if resp != nil {
				return resp.StatusCode
			}
			return 0
		}(), time.Since(start), err)
	}

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

	var result *RequestResult
	if envelope.Body.ResponseEmitidos != nil && envelope.Body.ResponseEmitidos.ResultEmitidos != nil {
		result = envelope.Body.ResponseEmitidos.ResultEmitidos
	} else if envelope.Body.ResponseRecibidos != nil && envelope.Body.ResponseRecibidos.ResultRecibidos != nil {
		result = envelope.Body.ResponseRecibidos.ResultRecibidos
	} else {
		return "", fmt.Errorf("sat error: estructura de respuesta XML irreconocible")
	}

	// Validate SAT status
	if err := s.validateSatStatus(result.CodeStatus, result.Message); err != nil {
		logSATError(s.config.Logger, "RequestMetadata", result.IDSolicitud, result.CodeStatus, result.Message)
		return "", err
	}

	if result.IDSolicitud == "" {
		return "", fmt.Errorf("empty UUID in SAT response")
	}

	return result.IDSolicitud, nil
}
func (s *SoapAdapter) DownloadPackage(rfc, packageID, certPath, keyPath, password string) ([]byte, error) {
	// Sanitize RFC and PackageID
	rfc = strings.ToUpper(strings.TrimSpace(rfc))
	packageID = strings.TrimSpace(packageID)

	token, err := s.authenticate(certPath, keyPath, password)
	if err != nil {
		return nil, fmt.Errorf("authentication failed: %w", err)
	}

	rb, err := NewRequestBuilder(keyPath, certPath, password)
	if err != nil {
		return nil, fmt.Errorf("builder initialization error: %w", err)
	}

	xmlPayload, err := rb.BuildDownloadRequest(rfc, packageID)
	if err != nil {
		return nil, fmt.Errorf("xml generation error: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, urlDescarga, bytes.NewReader(xmlPayload))
	if err != nil {
		return nil, fmt.Errorf("request creation error: %w", err)
	}

	s.setHeaders(req, actionDescarga, token)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error: %w", err)
	}
	defer resp.Body.Close()

	return s.processDownloadResponse(resp.Body)
}

func (s *SoapAdapter) authenticate(certPath, keyPath, password string) (string, error) {
	rb, err := NewRequestBuilder(keyPath, certPath, password)
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
	req.Header.Set("SOAPAction", `"`+actionAutentica+`"`)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("%s: %w", errAuthRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("error HTTP en autenticación: %d - %s", resp.StatusCode, string(bodyBytes))
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
