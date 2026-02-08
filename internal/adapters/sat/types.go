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
	CodEstatus string `xml:"codEstatus,attr"`
	Mensaje    string `xml:"mensaje,attr"`
}

type DownloadBody struct {
	PaqueteBase64 string `xml:"Paquete"` // Aquí viene el ZIP codificado
}

type Metadata struct {
	UUID               string     `json:"uuid" xml:"Uuid"`
	RfcEmisor          string     `json:"rfc_emisor" xml:"RfcEmisor"`
	NombreEmisor       string     `json:"nombre_emisor" xml:"NombreEmisor"`
	RfcReceptor        string     `json:"rfc_receptor" xml:"RfcReceptor"`
	NombreReceptor     string     `json:"nombre_receptor" xml:"NombreReceptor"`
	FechaEmision       time.Time  `json:"fecha_emision" xml:"FechaEmision"`
	FechaCertificacion time.Time  `json:"fecha_certificacion" xml:"FechaCertificacion"`
	Total              float64    `json:"total" xml:"Total"`
	TipoComprobante    string     `json:"tipo_comprobante" xml:"TipoDeComprobante"`           // I=Ingreso, E=Egreso, P=Pago
	Estatus            string     `json:"estatus" xml:"Estatus"`                              // Vigente, Cancelado
	FechaCancelacion   *time.Time `json:"fecha_cancelacion,omitempty" xml:"FechaCancelacion"` // Puntero porque puede ser null
}

type ReconciliationResult struct {
	Metadata     Metadata `json:"sat_data"`
	ErpMonto     float64  `json:"erp_monto"`
	Discrepancia float64  `json:"discrepancia"`
	StatusMatch  bool     `json:"status_match"` // true si Estatus SAT == Estatus ERP
	Comentario   string   `json:"comentario"`
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
	EstadoSolicitud       int      `xml:"EstadoSolicitud,attr"`       // 1:Aceptada, 2:EnProceso, 3:Terminada, 4:Error, 5:Rechazada
	CodigoEstadoSolicitud string   `xml:"CodigoEstadoSolicitud,attr"` // 5000: Éxito
	Mensaje               string   `xml:"Mensaje,attr"`
	NumeroCFDIs           int      `xml:"NumeroCFDIs,attr"`
	Paquetes              []string `xml:"IdsPaquetes"` // Los IDs para descargar (si terminó)
}

type VerifyParams struct {
	IdSolicitud    string
	RfcSolicitante string
	DigestValue    string
	SignatureValue string
	Certificate    string
	IssuerName     string
	SerialNumber   string
}
