package sat

import "time"

// Metadata representa la información clave extraída de los Web Services del SAT.
// No guardamos el XML completo en memoria, solo lo que importa para conciliar.
type Metadata struct {
	UUID             string    `json:"uuid" xml:"Uuid"`
	RfcEmisor        string    `json:"rfc_emisor" xml:"RfcEmisor"`
	NombreEmisor     string    `json:"nombre_emisor" xml:"NombreEmisor"`
	RfcReceptor      string    `json:"rfc_receptor" xml:"RfcReceptor"`
	NombreReceptor   string    `json:"nombre_receptor" xml:"NombreReceptor"`
	FechaEmision     time.Time `json:"fecha_emision" xml:"FechaEmision"`
	FechaCertificacion time.Time `json:"fecha_certificacion" xml:"FechaCertificacion"`
	Total            float64   `json:"total" xml:"Total"`
	TipoComprobante  string    `json:"tipo_comprobante" xml:"TipoDeComprobante"` // I=Ingreso, E=Egreso, P=Pago
	Estatus          string    `json:"estatus" xml:"Estatus"`                   // Vigente, Cancelado
	FechaCancelacion *time.Time `json:"fecha_cancelacion,omitempty" xml:"FechaCancelacion"` // Puntero porque puede ser null
}

// ReconciliationResult es lo que le entregamos al cliente (CSV/JSON final).
type ReconciliationResult struct {
	Metadata      Metadata `json:"sat_data"`
	ErpMonto      float64  `json:"erp_monto"`
	Discrepancia  float64  `json:"discrepancia"`
	StatusMatch   bool     `json:"status_match"` // true si Estatus SAT == Estatus ERP
	Comentario    string   `json:"comentario"`
}
