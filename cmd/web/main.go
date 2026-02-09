package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	satAdapter "github.com/1rene0lguin/sat-reconciler/internal/adapters/sat"
	"github.com/1rene0lguin/sat-reconciler/internal/core/domain"
	"github.com/1rene0lguin/sat-reconciler/internal/core/services"
)

// --- Constants (CamelCase) ---

const (
	// Server Config
	defaultPort = "3000"
	envPortKey  = "PORT"

	// Routes
	staticRoute         = "/static/"
	homeRoute           = "/"
	resumeRoute         = "/resume"
	executeReqRoute     = "/execute-request"
	checkStatusRoute    = "/check-status"
	verifyDownloadRoute = "/verify-and-download"

	// Paths
	staticDir  = "./web/static"
	layoutPath = "./web/templates/layout.html"
	homePath   = "./web/templates/index.html"
	resumePath = "./web/templates/resume.html"
	tempDir    = "./tmp"

	// Form Fields - Execute Request
	fieldStartDate = "fecha_inicio"
	fieldEndDate   = "fecha_fin"
	fieldRFCReq    = "rfc"
	fieldCerReq    = "cer"
	fieldKeyReq    = "key"
	fieldPassReq   = "password"

	// Form Fields - Check Status
	fieldRFC  = "rfc_verify"
	fieldUUID = "uuid_verify"
	fieldCer  = "cer_verify"
	fieldKey  = "key_verify"
	fieldPass = "password_verify"

	// Headers
	headerContentType = "Content-Type"
	contentTypeZip    = "application/zip"
	headerContentDisp = "Content-Disposition"
	contentDispAtt    = "attachment; filename=\"sat_metadata_%s.zip\""

	// Config
	maxUploadSize = 10 << 20 // 10MB

	// Messages
	msgMethodNotAllowed = "Método no permitido"
	msgInternalError    = "Error interno del servidor"
	msgParseError       = "Error procesando solicitud"
	msgFileError        = "Error guardando archivos temporales"
	msgInvalidURL       = "URL de descarga inválida"
	msgInvalidService   = "Error en servicio de conciliación"

	// HTML Responses
	htmlUploadSuccess = `<div class="p-4 bg-green-100 text-green-700 rounded border border-green-400">✅ Archivos recibidos en memoria</div>`

	htmlStatusInProgress = `
        <div class="mt-4 p-4 bg-slate-900 rounded border border-slate-700">
            <div class="flex items-center gap-3 mb-2">
                <div class="w-3 h-3 rounded-full bg-yellow-500 animate-pulse"></div>
                <span class="text-white font-bold">%s</span>
            </div>
            <p class="text-xs text-slate-400 font-mono">UUID: %s</p>
        </div>`

	htmlDownloadSuccess = `
        <div class="mt-4 p-4 bg-green-900 rounded border border-green-700">
            <div class="flex items-center gap-3 mb-2">
                <div class="w-3 h-3 rounded-full bg-green-500"></div>
                <span class="text-white font-bold">✅ Descarga Completada</span>
            </div>
            <p class="text-xs text-slate-400 font-mono mb-2">UUID: %s</p>
            <p class="text-xs text-green-400">Se descargaron %d paquete(s)</p>
            <p class="text-xs text-slate-500 mt-2">⚡ Credenciales FIEL eliminadas inmediatamente</p>
        </div>`
)

// --- Structures ---

type PageData struct {
	Title   string
	Version string
}

// --- Entry Point ---

func main() {
	if err := setupTempDir(); err != nil {
		log.Fatalf("Error creando directorio temporal: %v", err)
	}

	// 1. Infraestructura (Adapters)
	soapAdapter := satAdapter.NewSoapAdapter()

	// 2. Núcleo (Service)
	conciliator := services.NewConciliatorService(soapAdapter)

	// 3. Presentación (Router)
	mux := http.NewServeMux()
	setupStaticFiles(mux)
	setupRoutes(mux, conciliator)

	startServer(mux)
}

// --- Setup Functions ---

func setupTempDir() error {
	return os.MkdirAll(tempDir, 0755)
}

func setupStaticFiles(mux *http.ServeMux) {
	fs := http.FileServer(http.Dir(staticDir))
	mux.Handle(staticRoute, http.StripPrefix(staticRoute, fs))
}

func setupRoutes(mux *http.ServeMux, service *services.ConciliatorService) {
	mux.HandleFunc(homeRoute, homeHandler)
	mux.HandleFunc(resumeRoute, resumeHandler)
	mux.HandleFunc(executeReqRoute, makeExecuteRequestHandler(service))
	mux.HandleFunc(checkStatusRoute, makeCheckStatusHandler(service))
	mux.HandleFunc(verifyDownloadRoute, makeVerifyAndDownloadHandler(service))
}

func startServer(mux *http.ServeMux) {
	port := getServerPort()
	fmt.Printf("🐺 Irene Olguin - SAT Reconciler Web v1.0\n")
	fmt.Printf("🚀 Servidor corriendo en http://localhost:%s\n", port)

	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func getServerPort() string {
	if port := os.Getenv(envPortKey); port != "" {
		return port
	}
	return defaultPort
}

// --- Handlers ---

func homeHandler(w http.ResponseWriter, r *http.Request) {
	_ = r
	render(w, homePath, PageData{Title: "SAT Reconciler", Version: "v1.0.0"})
}

func makeExecuteRequestHandler(service *services.ConciliatorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodPost) {
			return
		}

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, msgParseError, http.StatusBadRequest)
			return
		}

		// Extract form values
		rfc := r.FormValue(fieldRFCReq)
		startDate := r.FormValue(fieldStartDate)
		endDate := r.FormValue(fieldEndDate)
		// password := r.FormValue(fieldPassReq) // TODO: Use for key decryption

		// Save FIEL files temporarily
		certPath, cleanupCert, err := saveTempFile(r, fieldCerReq)
		if err != nil {
			fmt.Fprintf(w, `<div class="p-4 bg-red-100 text-red-700 rounded border border-red-400">❌ Error guardando certificado</div>`)
			return
		}

		keyPath, cleanupKey, err := saveTempFile(r, fieldKeyReq)
		if err != nil {
			cleanupCert()
			fmt.Fprintf(w, `<div class="p-4 bg-red-100 text-red-700 rounded border border-red-400">❌ Error guardando llave privada</div>`)
			return
		}

		// CRITICAL: Ensure cleanup happens no matter what
		defer func() {
			cleanupCert()
			cleanupKey()
		}()

		// Call service to create request
		uuid, err := service.RequestMetadata(rfc, startDate, endDate, certPath, keyPath)
		if err != nil {
			fmt.Printf("Service Error: %v\n", err)
			fmt.Fprintf(w, `<div class="p-4 bg-red-100 text-red-700 rounded border border-red-400">❌ Error al enviar solicitud al SAT: %s</div>`, err.Error())
			return
		}

		// Return success message with UUID
		successHTML := fmt.Sprintf(`
		<div class="mt-4 p-4 bg-green-900 rounded border border-green-700">
			<div class="flex items-center gap-3 mb-2">
				<div class="w-3 h-3 rounded-full bg-green-500"></div>
				<span class="text-white font-bold">✅ Solicitud Enviada Exitosamente</span>
			</div>
			<p class="text-xs text-slate-400 font-mono mb-2">UUID: %s</p>
			<p class="text-xs text-green-400">Ahora puedes verificar el estatus usando la sección inferior.</p>
			<p class="text-xs text-slate-500 mt-2">⚡ Credenciales FIEL eliminadas de memoria</p>
		</div>`, uuid)

		fmt.Fprint(w, successHTML)
	}
}

func makeCheckStatusHandler(service *services.ConciliatorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodPost) {
			return
		}

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, msgParseError, http.StatusBadRequest)
			return
		}

		rfc := r.FormValue(fieldRFC)
		uuid := r.FormValue(fieldUUID)
		// password := r.FormValue(fieldPass) // TODO: Use for key decryption

		// Save FIEL files temporarily
		certPath, cleanupCert, err := saveTempFile(r, fieldCer)
		if err != nil {
			fmt.Fprintf(w, `<div class="p-4 bg-red-100 text-red-700 rounded border border-red-400">❌ Error guardando certificado</div>`)
			return
		}

		keyPath, cleanupKey, err := saveTempFile(r, fieldKey)
		if err != nil {
			cleanupCert()
			fmt.Fprintf(w, `<div class="p-4 bg-red-100 text-red-700 rounded border border-red-400">❌ Error guardando llave privada</div>`)
			return
		}

		// CRITICAL: Ensure cleanup happens no matter what
		defer func() {
			cleanupCert()
			cleanupKey()
		}()

		// Verify status with SAT
		result, err := service.CheckStatus(rfc, uuid, certPath, keyPath)
		if err != nil {
			fmt.Printf("Service Error: %v\n", err)
			fmt.Fprintf(w, `<div class="p-4 bg-red-100 text-red-700 rounded border border-red-400">❌ Error consultando al SAT: %s</div>`, err.Error())
			return
		}

		// Return status based on result
		if result.Status != domain.StatusFinished {
			statusText := fmt.Sprintf("Estado: %d - %s", result.Status, result.Message)
			fmt.Fprintf(w, htmlStatusInProgress, statusText, uuid)
			return
		}

		// If finished
		if len(result.PackageIDs) == 0 {
			statusText := "Solicitud terminada pero no hay paquetes disponibles"
			fmt.Fprintf(w, htmlStatusInProgress, statusText, uuid)
			return
		}

		// Success with packages
		fmt.Fprintf(w, htmlDownloadSuccess, uuid, len(result.PackageIDs))
	}
}

func resumeHandler(w http.ResponseWriter, r *http.Request) {
	_ = r
	render(w, resumePath, PageData{Title: "Resume | Irene Olguin", Version: "v1.0.0"})
}

func makeVerifyAndDownloadHandler(service *services.ConciliatorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !ensureMethod(w, r, http.MethodPost) {
			return
		}

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, msgParseError, http.StatusBadRequest)
			return
		}

		rfc := r.FormValue(fieldRFC)
		uuid := r.FormValue(fieldUUID)

		// Save FIEL files temporarily
		certPath, cleanupCert, err := saveTempFile(r, fieldCer)
		if err != nil {
			http.Error(w, msgFileError, http.StatusInternalServerError)
			return
		}

		keyPath, cleanupKey, err := saveTempFile(r, fieldKey)
		if err != nil {
			cleanupCert() // Cleanup cert if key fails
			http.Error(w, msgFileError, http.StatusInternalServerError)
			return
		}

		// CRITICAL: Ensure cleanup happens no matter what
		defer func() {
			cleanupCert()
			cleanupKey()
		}()

		// 1. Verify status with SAT
		result, err := service.CheckStatus(rfc, uuid, certPath, keyPath)
		if err != nil {
			fmt.Printf("Service Error: %v\n", err)
			http.Error(w, msgInvalidService, http.StatusInternalServerError)
			return
		}

		// 2. If not finished, return status and exit (credentials destroyed by defer)
		if result.Status != domain.StatusFinished {
			statusText := fmt.Sprintf("Estado: %d - %s", result.Status, result.Message)
			fmt.Fprintf(w, htmlStatusInProgress, statusText, uuid)
			return
		}

		// 3. If finished but no packages, return success message
		if len(result.PackageIDs) == 0 {
			statusText := "Solicitud terminada pero no hay paquetes disponibles"
			fmt.Fprintf(w, htmlStatusInProgress, statusText, uuid)
			return
		}

		// 4. Download ALL packages immediately (atomic operation)
		packages := make(map[string][]byte)
		for _, pkgID := range result.PackageIDs {
			zipBytes, err := service.DownloadPackage(rfc, pkgID, certPath, keyPath)
			if err != nil {
				// Log error but continue with other packages
				fmt.Printf("Error downloading package %s: %v\n", pkgID, err)
				continue
			}
			packages[pkgID] = zipBytes
		}

		// 5. Credentials will be DESTROYED immediately after this function returns (defer)

		// 6. If we got at least one package, create bundle and return
		if len(packages) == 0 {
			http.Error(w, "No se pudo descargar ningún paquete", http.StatusInternalServerError)
			return
		}

		// 7. Bundle packages or return single package
		var finalZip []byte
		if len(packages) == 1 {
			// Single package - return as-is
			for _, zipBytes := range packages {
				finalZip = zipBytes
				break
			}
		} else {
			// Multiple packages - bundle into one ZIP
			finalZip, err = bundlePackages(packages)
			if err != nil {
				http.Error(w, "Error creating package bundle", http.StatusInternalServerError)
				return
			}
		}

		// 8. Return ZIP file
		w.Header().Set(headerContentType, contentTypeZip)
		w.Header().Set(headerContentDisp, fmt.Sprintf(contentDispAtt, uuid))
		if _, err := w.Write(finalZip); err != nil {
			fmt.Printf("Error writing zip: %v\n", err)
		}

		// 9. Log success (credentials already destroyed by defer)
		fmt.Printf("✅ Downloaded %d package(s) for UUID %s - FIEL credentials destroyed\n", len(packages), uuid)
	}
}

// bundlePackages combines multiple SAT packages into a single ZIP file
func bundlePackages(packages map[string][]byte) ([]byte, error) {
	var buf bytes.Buffer
	zipWriter := zip.NewWriter(&buf)

	for pkgID, zipBytes := range packages {
		// Create a file entry in the bundle ZIP
		fileName := fmt.Sprintf("%s.zip", pkgID)
		writer, err := zipWriter.Create(fileName)
		if err != nil {
			return nil, fmt.Errorf("error creating zip entry: %w", err)
		}

		// Write the package ZIP content
		if _, err := writer.Write(zipBytes); err != nil {
			return nil, fmt.Errorf("error writing zip content: %w", err)
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, fmt.Errorf("error closing zip writer: %w", err)
	}

	return buf.Bytes(), nil
}

// --- Helper Functions ---

func ensureMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return false
	}
	return true
}

func render(w http.ResponseWriter, templatePath string, data PageData) {
	files := []string{layoutPath, templatePath}
	ts, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, msgInternalError, http.StatusInternalServerError)
		return
	}
	if err := ts.ExecuteTemplate(w, "layout", data); err != nil {
		fmt.Printf("Error renderizando template: %v\n", err)
	}
}

func saveTempFile(r *http.Request, fieldName string) (string, func(), error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return "", func() {}, err
	}
	defer file.Close()

	safeName := "sat-" + filepath.Base(header.Filename)
	tempFile, err := os.CreateTemp(tempDir, safeName)
	if err != nil {
		return "", func() {}, err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		return "", func() {}, err
	}

	path := tempFile.Name()
	cleanup := func() {
		_ = os.Remove(path)
	}

	return path, cleanup, nil
}
