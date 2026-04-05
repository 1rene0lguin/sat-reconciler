package sat

const (
	// SOAP Actions & URLs
	actionAutentica  = "http://DescargaMasivaTerceros.gob.mx/IAutenticacion/Autentica"
	urlAutenticacion = "https://cfdidescargamasivasolicitud.clouda.sat.gob.mx/Autenticacion/Autenticacion.svc"
	urlSolicitud             = "https://cfdidescargamasivasolicitud.clouda.sat.gob.mx/SolicitaDescargaService.svc"
	actionSolicitudEmitidos  = "http://DescargaMasivaTerceros.sat.gob.mx/ISolicitaDescargaService/SolicitaDescargaEmitidos"
	actionSolicitudRecibidos = "http://DescargaMasivaTerceros.sat.gob.mx/ISolicitaDescargaService/SolicitaDescargaRecibidos"
	urlVerifica      = "https://cfdidescargamasivasolicitud.clouda.sat.gob.mx/VerificaSolicitudDescargaService.svc"
	actionVerifica   = "http://DescargaMasivaTerceros.sat.gob.mx/IVerificaSolicitudDescargaService/VerificaSolicitudDescarga"
	urlDescarga      = "https://cfdidescargamasiva.clouda.sat.gob.mx/DescargaMasivaService.svc"
	actionDescarga   = "http://DescargaMasivaTerceros.sat.gob.mx/IDescargaMasivaTercerosService/Descargar"

	// Namespaces & Formats
	dateTimeFormat = "2006-01-02T15:04:05.000Z"

	// Templates
	templateNameAuth = "auth_request.xml"

	// XML Elements
	envAutenticaFmt = `<u:Timestamp xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" u:Id="_0"><u:Created>%s</u:Created><u:Expires>%s</u:Expires></u:Timestamp>`

	headerContentType = "Content-Type"
	headerAuth        = "Authorization"
	mimeTypeXML       = "text/xml; charset=utf-8"
	authPrefix        = "WRAP access_token=\""
	authSuffix        = "\""

	satStatusSuccess = "5000"

	// Metadata Parsing
	MetadataSeparator = "~"
	StatusVigente     = "1"
	StatusCancelado   = "0"
)

type AuthParams struct {
	Created        string
	Expires        string
	DigestValue    string
	SignatureValue string
	Certificate    string
	BinaryToken    string
	BinaryTokenID  string
}
