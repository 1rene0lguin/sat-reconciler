package sat

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"text/template"
	"time"
)

const (
	templatePathSolicitud = "internal/sat/templates/soap_request.xml"
	templatePathVerifica  = "internal/sat/templates/verify_request.xml"

	templateNameSolicitud = "soap_request.xml"
	templateNameVerifica  = "verify_request.xml"

	canonicalSolicitudFmt = `<des:solicitud xmlns:des="http://DescargaMasivaTerceros.sat.gob.mx" FechaFin="%s" FechaInicio="%s" RfcEmisor="%s" RfcReceptor="%s" RfcSolicitante="%s" TipoSolicitud="%s"></des:solicitud>`
	canonicalVerificaFmt  = `<des:solicitud xmlns:des="http://DescargaMasivaTerceros.sat.gob.mx" IdSolicitud="%s" RfcSolicitante="%s"></des:solicitud>`

	signedInfoFmt = `<SignedInfo xmlns="http://www.w3.org/2000/09/xmldsig#"><CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"></CanonicalizationMethod><SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"></SignatureMethod><Reference URI=""><Transforms><Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></Transform></Transforms><DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"></DigestMethod><DigestValue>%s</DigestValue></Reference></SignedInfo>`

	templateNameDescarga = "download_request.xml"
	canonicalDescargaFmt = `<des:peticionDescarga xmlns:des="http://DescargaMasivaTerceros.sat.gob.mx" IdPaquete="%s" RfcSolicitante="%s"></des:peticionDescarga>`
)

type SoapRequestParams struct {
	RfcSolicitante string
	FechaInicio    string
	FechaFin       string
	TipoSolicitud  string
	RfcEmisor      string
	RfcReceptor    string
	DigestValue    string
	SignatureValue string
	Certificate    string
	IssuerName     string
	SerialNumber   string
	IdSolicitud    string
}

type DownloadParams struct {
	IdPaquete      string
	RfcSolicitante string
	DigestValue    string
	SignatureValue string
	Certificate    string
	IssuerName     string
	SerialNumber   string
}

type RequestBuilder struct {
	privateKey *rsa.PrivateKey
	cert       *x509.Certificate
	templates  *template.Template
}

func NewRequestBuilder(keyPath, cerPath string) (*RequestBuilder, error) {
	privKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return nil, err
	}

	cert, err := loadCertificate(cerPath)
	if err != nil {
		return nil, err
	}

	tmpls, err := loadTemplates()
	if err != nil {
		return nil, err
	}

	return &RequestBuilder{
		privateKey: privKey,
		cert:       cert,
		templates:  tmpls,
	}, nil
}

func (a *RequestBuilder) BuildSignedRequest(params SoapRequestParams) ([]byte, error) {
	canonicalString := fmt.Sprintf(
		canonicalSolicitudFmt,
		params.FechaFin, params.FechaInicio, params.RfcEmisor, params.RfcReceptor, params.RfcSolicitante, params.TipoSolicitud,
	)

	return a.buildXML(templateNameSolicitud, canonicalString, &params)
}

func (a *RequestBuilder) BuildVerificationRequest(rfc, idSolicitud string) ([]byte, error) {
	canonicalString := fmt.Sprintf(canonicalVerificaFmt, idSolicitud, rfc)

	params := SoapRequestParams{
		IdSolicitud:    idSolicitud,
		RfcSolicitante: rfc,
	}

	return a.buildXML(templateNameVerifica, canonicalString, &params)
}

// --- Private Helper Functions ---

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, errors.New("no se encontró bloque PEM en la llave privada")
	}

	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}

	key8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("formato de llave no soportado o contraseña incorrecta: %w", err)
	}

	rsaKey, ok := key8.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("la llave no es RSA")
	}

	return rsaKey, nil
}

func loadCertificate(path string) (*x509.Certificate, error) {
	cerBytes, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("error leyendo cer: %w", err)
	}

	block, _ := pem.Decode(cerBytes)
	if block != nil {
		cerBytes = block.Bytes
	}

	return x509.ParseCertificate(cerBytes)
}

func loadTemplates() (*template.Template, error) {
	return template.ParseFiles(templatePathSolicitud, templatePathVerifica)
}

func (a *RequestBuilder) buildXML(tmplName, canonicalString string, params *SoapRequestParams) ([]byte, error) {
	digest, signature, err := a.computeSignature(canonicalString)
	if err != nil {
		return nil, err
	}

	params.DigestValue = digest
	params.SignatureValue = signature
	params.Certificate = base64.StdEncoding.EncodeToString(a.cert.Raw)
	params.IssuerName = a.cert.Issuer.String()
	params.SerialNumber = a.cert.SerialNumber.String()

	var finalXML bytes.Buffer
	if err := a.templates.ExecuteTemplate(&finalXML, tmplName, params); err != nil {
		return nil, fmt.Errorf("error renderizando template %s: %w", tmplName, err)
	}

	return finalXML.Bytes(), nil
}

func (rb *RequestBuilder) computeSignature(canonicalString string) (digest, signature string, err error) {
	h := sha1.New()
	h.Write([]byte(canonicalString))
	digestBytes := h.Sum(nil)
	digest = base64.StdEncoding.EncodeToString(digestBytes)

	signedInfo := fmt.Sprintf(signedInfoFmt, digest)

	sh := sha1.New()
	sh.Write([]byte(signedInfo))
	signedInfoHash := sh.Sum(nil)

	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, rb.privateKey, crypto.SHA1, signedInfoHash)
	if err != nil {
		return "", "", fmt.Errorf("error firmando RSA: %w", err)
	}

	return digest, base64.StdEncoding.EncodeToString(sigBytes), nil
}

func (rb *RequestBuilder) BuildDownloadRequest(rfc, idPaquete string) ([]byte, error) {
	canonicalString := fmt.Sprintf(canonicalDescargaFmt, idPaquete, rfc)

	digest, signature, err := rb.computeSignature(canonicalString)
	if err != nil {
		return nil, err
	}

	params := DownloadParams{
		IdPaquete:      idPaquete,
		RfcSolicitante: rfc,
		DigestValue:    digest,
		SignatureValue: signature,
		Certificate:    base64.StdEncoding.EncodeToString(rb.cert.Raw),
		IssuerName:     rb.cert.Issuer.String(),
		SerialNumber:   rb.cert.SerialNumber.String(),
	}

	var finalXML bytes.Buffer
	if err := rb.templates.ExecuteTemplate(&finalXML, templateNameDescarga, params); err != nil {
		return nil, fmt.Errorf("error renderizando template descarga: %w", err)
	}

	return finalXML.Bytes(), nil
}

func (rb *RequestBuilder) BuildAuthRequest() ([]byte, error) {
	now := time.Now().UTC()
	created := now.Format(dateTimeFormat)
	expires := now.Add(5 * time.Minute).Format(dateTimeFormat)

	// 1. Canonicalización del Timestamp (Lo que realmente firma el SAT en Auth)
	canonicalTimestamp := fmt.Sprintf(envAutenticaFmt, created, expires)

	// 2. Firmar
	digest, signature, err := rb.computeSignature(canonicalTimestamp)
	if err != nil {
		return nil, err
	}

	// 3. Preparar parámetros para el template
	params := AuthParams{
		Created:     created,
		Expires:     expires,
		DigestValue: digest,
		Signature:   signature,
		Certificate: base64.StdEncoding.EncodeToString(rb.cert.Raw),
		Uuid:        fmt.Sprintf("uuid-%d", now.UnixNano()), // ID único para el BinarySecurityToken
	}

	return rb.renderTemplate(templateNameAuth, params)
}

func (rb *RequestBuilder) renderTemplate(tmplName string, data any) ([]byte, error) {
	var buffer bytes.Buffer

	if err := rb.templates.ExecuteTemplate(&buffer, tmplName, data); err != nil {
		return nil, fmt.Errorf("error renderizando template %s: %w", tmplName, err)
	}

	return buffer.Bytes(), nil
}
