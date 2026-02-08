package main

import (
	"net/http"

	sat_adapter "github.com/i4ene0lguin/sat-reconcilier/internal/adapters/sat"
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/services"
)

type PageData struct {
	Title   string
	Version string
}

// --- Entry Point ---
func main() {
	// 1. Infraestructura (Adaptadores)
	soapAdapter := sat_adapter.NewSoapAdapter()

	// 2. Núcleo (Servicio) - Inyectamos el adaptador
	conciliator := services.NewConciliatorService(soapAdapter)

	// 3. Presentación (Handlers)
	// Pasamos el SERVICIO, no el adaptador SOAP.
	// El Handler web no sabe que existe XML o SOAP.
	http.HandleFunc("/check-status", makeCheckStatusHandler(conciliator))

	// ... server start ...
}

func makeCheckStatusHandler(service *services.ConciliatorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// ... parse form ...

		// Llamada limpia al negocio
		msg, err := service.VerifyRequest(rfc, uuid, certPath, keyPath)

		// Render respuesta
	}
}
