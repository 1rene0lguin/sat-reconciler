package sat

const (
	// SOAP Actions & URLs
	actionAutentica  = "http://DescargaMasivaTerceros.sat.gob.mx/IAutenticacion/Autentica"
	urlAutenticacion = "https://cfdidescargamasivasolicitud.clouda.sat.gob.mx/Autenticacion/Autenticacion.svc"
	urlDescarga      = "https://cfdidescargamasiva.clouda.sat.gob.mx/PeticionDescargaMasiva/PeticionDescargaMasiva.svc"

	// Namespaces & Formats
	dateTimeFormat = "2006-01-02T15:04:05.000Z"

	// Templates
	templateNameAuth = "auth_request.xml"

	// XML Elements
	envAutenticaFmt = `<u:Timestamp xmlns:u="http://docs.oasis-open.org/wss/2004/01/oasis-200401-wss-wssecurity-utility-1.0.xsd" u:Id="_0"><u:Created>%s</u:Created><u:Expires>%s</u:Expires></u:Timestamp>`

	// Errores
	errAuthBuild   = "error construyendo petición de autenticación"
	errAuthSign    = "error firmando timestamp de autenticación"
	errAuthRequest = "error enviando solicitud de autenticación"
	errAuthParse   = "error leyendo token de respuesta"
	errEmptyToken  = "el token de autenticación recibido está vacío"

	soapActionDescarga = "http://DescargaMasivaTerceros.sat.gob.mx/IPeticionDescargaMasivaService/PeticionDescargaMasiva"

	headerContentType = "Content-Type"
	headerAuth        = "Authorization"
	mimeTypeXML       = "text/xml; charset=utf-8"
	authPrefix        = "WRAP access_token=\""
	authSuffix        = "\""

	satStatusSuccess = "5000"
)

type AuthParams struct {
	Created     string
	Expires     string
	DigestValue string
	Signature   string
	Certificate string
	BinaryToken string
	Uuid        string
}
