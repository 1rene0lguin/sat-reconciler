package sat

import "time"

type DownloadResponseEnvelope struct {
	Body DownloadResponseBody `xml:"Body"`
}

type DownloadResponseBody struct {
	Response DownloadResponse `xml:"RespuestaDescargaMasivaTercerosSalida"`
}

type DownloadResponse struct {
	Header DownloadHeader `xml:"header"`
	Body   DownloadBody   `xml:"body"`
}

type DownloadHeader struct {
	CodeStatus string `xml:"codEstatus,attr"`
	Message    string `xml:"mensaje,attr"`
}

type DownloadBody struct {
	PackageBase64 string `xml:"Paquete"` // ZIP encoded in Base64
}

type Metadata struct {
	UUID              string     `json:"uuid" xml:"Uuid"`
	RfcIssuer         string     `json:"rfc_emisor" xml:"RfcEmisor"`
	NameIssuer        string     `json:"nombre_emisor" xml:"NombreEmisor"`
	RfcReceiver       string     `json:"rfc_receptor" xml:"RfcReceptor"`
	NameReceiver      string     `json:"nombre_receptor" xml:"NombreReceptor"`
	DateEmission      time.Time  `json:"fecha_emision" xml:"FechaEmision"`
	DateCertification time.Time  `json:"fecha_certificacion" xml:"FechaCertificacion"`
	Total             float64    `json:"total" xml:"Total"`
	TypeVoucher       string     `json:"tipo_comprobante" xml:"TipoDeComprobante"`           // I=Ingreso, E=Egreso, P=Pago
	Status            string     `json:"estatus" xml:"Estatus"`                              // Vigente, Cancelado
	DateCancellation  *time.Time `json:"fecha_cancelacion,omitempty" xml:"FechaCancelacion"` // Pointer because it can be null
}

type ReconciliationResult struct {
	Metadata    Metadata `json:"sat_data"`
	ErpAmount   float64  `json:"erp_monto"`
	Discrepancy float64  `json:"discrepancia"`
	StatusMatch bool     `json:"status_match"` // true if SAT Status == ERP Status
	Comment     string   `json:"comentario"`
}

type RequestResponseEnvelope struct {
	Body RequestResponseBody `xml:"Body"`
}

type RequestResponseBody struct {
	Response RequestResponse `xml:"SolicitaDescargaResponse"`
}

type RequestResponse struct {
	Result RequestResult `xml:"SolicitaDescargaResult"`
}

type RequestResult struct {
	IDSolicitud string `xml:"IdSolicitud,attr"`
	CodeStatus  string `xml:"CodEstatus,attr"`
	Message     string `xml:"Mensaje,attr"`
}

type VerifyResponseEnvelope struct {
	Body VerifyResponseBody `xml:"Body"`
}

type VerifyResponseBody struct {
	Response VerifyResponse `xml:"VerificaSolicitudDescargaResponse"`
}

type VerifyResponse struct {
	Result VerifyResult `xml:"VerificaSolicitudDescargaResult"`
}

type VerifyResult struct {
	StatusRequest     int      `xml:"EstadoSolicitud,attr"`       // 1:Accepted, 2:InProcess, 3:Finished, 4:Error, 5:Rejected
	CodeStatusRequest string   `xml:"CodigoEstadoSolicitud,attr"` // 5000: Success
	Message           string   `xml:"Mensaje,attr"`
	NumberCFDIs       int      `xml:"NumeroCFDIs,attr"`
	Packages          []string `xml:"IdsPaquetes"` // Package IDs for download
}

type VerifyParams struct {
	RequestID      string
	RfcSolicitant  string
	DigestValue    string
	SignatureValue string
	Certificate    string
	IssuerName     string
	SerialNumber   string
}
