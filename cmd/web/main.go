package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	satAdapter "github.com/i4ene0lguin/sat-reconcilier/internal/adapters/sat"
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/domain"
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/services"
)

// --- Constants (CamelCase) ---

const (
	// Server Config
	defaultPort = "3000"
	envPortKey  = "PORT"

	// Routes
	staticRoute   = "/static/"
	homeRoute     = "/"
	resumeRoute   = "/resume"
	uploadRoute   = "/upload-fiel"
	checkRoute    = "/check-status"
	downloadRoute = "/download/"

	// Paths
	staticDir  = "./web/static"
	layoutPath = "./web/templates/layout.html"
	homePath   = "./web/templates/index.html"
	resumePath = "./web/templates/resume.html"
	tempDir    = "./tmp"

	// Form Fields
	fieldRFC  = "rfc_verify"
	fieldUUID = "uuid_verify"
	fieldCer  = "cer_verify"
	fieldKey  = "key_verify"

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

	htmlStatusResult = `
        <div class="mt-4 p-4 bg-slate-900 rounded border border-slate-700">
            <div class="flex items-center gap-3 mb-2">
                <div class="w-3 h-3 rounded-full bg-blue-500 animate-pulse"></div>
                <span class="text-white font-bold">%s</span>
            </div>
            <p class="text-xs text-slate-400 font-mono mb-3">UUID: %s</p>
			%s
        </div>`

	htmlDownloadBtn = `<a href="/download/%s/%s" class="block w-full text-center bg-purple-600 hover:bg-purple-500 text-white font-bold py-2 px-4 rounded transition-colors text-sm">💾 Descargar Paquete</a>`
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

	// 1. Infraestructura (Adapters) - CamelCase
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
	mux.HandleFunc(uploadRoute, uploadHandler)
	mux.HandleFunc(checkRoute, makeCheckStatusHandler(service))
	mux.HandleFunc(downloadRoute, makeDownloadHandler(service))
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

func resumeHandler(w http.ResponseWriter, r *http.Request) {
	_ = r
	render(w, resumePath, PageData{Title: "Resume | Irene Olguin", Version: "v1.0.0"})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureMethod(w, r, http.MethodPost) {
		return
	}
	fmt.Fprint(w, htmlUploadSuccess)
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

		certPath, cleanupCert, err := saveTempFile(r, fieldCer)
		if err != nil {
			http.Error(w, msgFileError, http.StatusInternalServerError)
			return
		}
		defer cleanupCert()

		keyPath, cleanupKey, err := saveTempFile(r, fieldKey)
		if err != nil {
			http.Error(w, msgFileError, http.StatusInternalServerError)
			return
		}
		defer cleanupKey()

		result, err := service.CheckStatus(rfc, uuid, certPath, keyPath)
		if err != nil {
			fmt.Printf("Service Error: %v\n", err)
			http.Error(w, msgInvalidService, http.StatusInternalServerError)
			return
		}

		actionHTML := ""

		if result.Status == domain.StatusFinished && len(result.PackageIDs) > 0 {
			actionHTML = fmt.Sprintf(htmlDownloadBtn, uuid, result.PackageIDs[0])
		}

		statusText := fmt.Sprintf("Estado: %d - %s", result.Status, result.Message)
		fmt.Fprintf(w, htmlStatusResult, statusText, uuid, actionHTML)
	}
}

func makeDownloadHandler(service *services.ConciliatorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_ = service

		if !ensureMethod(w, r, http.MethodGet) {
			return
		}

		path := strings.TrimPrefix(r.URL.Path, downloadRoute)
		parts := strings.Split(path, "/")

		if len(parts) < 2 {
			http.Error(w, msgInvalidURL, http.StatusBadRequest)
			return
		}

		pkgID := parts[1]

		zipBytes := []byte("PK-SIMULATED-ZIP-CONTENT-METADATA-FROM-GO")

		w.Header().Set(headerContentType, contentTypeZip)
		w.Header().Set(headerContentDisp, fmt.Sprintf(contentDispAtt, pkgID))
		if _, err := w.Write(zipBytes); err != nil {
			fmt.Printf("Error escribiendo zip: %v\n", err)
		}
	}
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
