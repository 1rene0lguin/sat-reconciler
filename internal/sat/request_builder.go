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
	"fmt"
	"os"
	"text/template"
)

// SoapRequestParams son los datos variables para pedir la descarga.
type SoapRequestParams struct {
	RfcSolicitante string
	FechaInicio    string // Formato: YYYY-MM-DDTHH:MM:SS
	FechaFin       string
	TipoSolicitud  string // "CFDI" o "Metadata"
	RfcEmisor      string
	RfcReceptor    string

	// Campos internos calculados
	DigestValue    string
	SignatureValue string
	Certificate    string
	IssuerName     string
	SerialNumber   string
}

// RequestBuilder orquesta la creación del XML firmado.
type RequestBuilder struct {
	privateKey *rsa.PrivateKey
	cert       *x509.Certificate
	template   *template.Template
}

// NewRequestBuilder carga las credenciales y el template.
func NewRequestBuilder(keyPath, cerPath, templatePath string) (*RequestBuilder, error) {
	// 1. Cargar Llave Privada (Asumimos que es PKCS1 o PKCS8 sin password por ahora)
	keyBytes, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo key: %w", err)
	}

	block, _ := pem.Decode(keyBytes)
	if block == nil {
		return nil, fmt.Errorf("no se encontró bloque PEM en la llave")
	}

	// Intentamos parsear como PKCS1 (formato común de OpenSSL)
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Si falla, intentar PKCS8
		k8, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err2 != nil {
			return nil, fmt.Errorf("error parseando llave privada (asegurate que no tenga pass): %v", err)
		}
		privKey = k8.(*rsa.PrivateKey)
	}

	// 2. Cargar Certificado
	cerBytes, err := os.ReadFile(cerPath)
	if err != nil {
		return nil, fmt.Errorf("error leyendo cer: %w", err)
	}

	// Si viene en formato binario (DER) lo convertimos, si es PEM lo leemos directo
	var cert *x509.Certificate
	pBlock, _ := pem.Decode(cerBytes)
	if pBlock != nil {
		cert, err = x509.ParseCertificate(pBlock.Bytes)
	} else {
		cert, err = x509.ParseCertificate(cerBytes)
	}
	if err != nil {
		return nil, fmt.Errorf("error parseando certificado: %w", err)
	}

	// 3. Cargar Template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return nil, fmt.Errorf("error cargando template XML: %w", err)
	}

	return &RequestBuilder{
		privateKey: privKey,
		cert:       cert,
		template:   tmpl,
	}, nil
}

// BuildSignedRequest genera el XML final listo para enviarse.
func (rb *RequestBuilder) BuildSignedRequest(params SoapRequestParams) ([]byte, error) {
	// Paso 1: Canonicalizar y calcular Digest
	// El SAT pide firmar el elemento <des:solicitud> CANONICALIZADO.
	// Como Go no tiene un canonicalizador XML robusto nativo (C14N),
	// haremos un truco: construiremos el string exacto que vamos a firmar.
	// OJO: Los espacios y orden de atributos importan.

	canonicalString := fmt.Sprintf(
		`<des:solicitud xmlns:des="http://DescargaMasivaTerceros.sat.gob.mx" FechaFin="%s" FechaInicio="%s" RfcEmisor="%s" RfcReceptor="%s" RfcSolicitante="%s" TipoSolicitud="%s"></des:solicitud>`,
		params.FechaFin, params.FechaInicio, params.RfcEmisor, params.RfcReceptor, params.RfcSolicitante, params.TipoSolicitud,
	)

	// Paso 2: Calcular Digest (SHA1)
	h := sha1.New()
	h.Write([]byte(canonicalString))
	digest := base64.StdEncoding.EncodeToString(h.Sum(nil))
	params.DigestValue = digest

	// Paso 3: Calcular SignatureValue
	// Firmamos el elemento <SignedInfo> que incluye el Digest que acabamos de calcular.
	signedInfo := fmt.Sprintf(
		`<SignedInfo xmlns="http://www.w3.org/2000/09/xmldsig#"><CanonicalizationMethod Algorithm="http://www.w3.org/TR/2001/REC-xml-c14n-20010315"></CanonicalizationMethod><SignatureMethod Algorithm="http://www.w3.org/2000/09/xmldsig#rsa-sha1"></SignatureMethod><Reference URI=""><Transforms><Transform Algorithm="http://www.w3.org/2000/09/xmldsig#enveloped-signature"></Transform></Transforms><DigestMethod Algorithm="http://www.w3.org/2000/09/xmldsig#sha1"></DigestMethod><DigestValue>%s</DigestValue></Reference></SignedInfo>`,
		digest,
	)

	sh := sha1.New()
	sh.Write([]byte(signedInfo))
	sigBytes, err := rsa.SignPKCS1v15(rand.Reader, rb.privateKey, crypto.SHA1, sh.Sum(nil))
	if err != nil {
		return nil, fmt.Errorf("error firmando SignedInfo: %w", err)
	}
	params.SignatureValue = base64.StdEncoding.EncodeToString(sigBytes)

	// Paso 4: Datos del Certificado para el XML
	params.Certificate = base64.StdEncoding.EncodeToString(rb.cert.Raw)
	params.IssuerName = rb.cert.Issuer.String() // Ojo: SAT a veces pide formato inverso
	params.SerialNumber = rb.cert.SerialNumber.String()

	// Paso 5: Renderizar Template Final
	var finalXML bytes.Buffer
	if err := rb.template.Execute(&finalXML, params); err != nil {
		return nil, fmt.Errorf("error renderizando template: %w", err)
	}

	return finalXML.Bytes(), nil
}
