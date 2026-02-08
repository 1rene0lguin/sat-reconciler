package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	sat_adapter "github.com/i4ene0lguin/sat-reconcilier/internal/adapters/sat"
	"github.com/i4ene0lguin/sat-reconcilier/internal/core/services"
)

// --- Constants ---

const (
	defaultPort = "3000"
	envPortKey  = "PORT"

	// Routes
	staticRoute = "/static/"
	homeRoute   = "/"
	resumeRoute = "/resume"
	uploadRoute = "/upload-fiel"
	checkRoute  = "/check-status"

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

	// Config
	maxUploadSize = 10 << 20 // 10MB

	// Messages
	msgMethodNotAllowed = "Método no permitido"
	msgInternalError    = "Error interno del servidor"
	msgParseError       = "Error procesando solicitud"
	msgFileError        = "Error guardando archivos temporales"

	// HTML Responses
	htmlUploadSuccess = `<div class="p-4 bg-green-100 text-green-700 rounded border border-green-400">✅ Archivos recibidos en memoria</div>`
	htmlStatusResult  = `
        <div class="mt-4 p-4 bg-slate-900 rounded border border-slate-700">
            <div class="flex items-center gap-3 mb-2">
                <div class="w-3 h-3 rounded-full bg-blue-500 animate-pulse"></div>
                <span class="text-white font-bold">%s</span>
            </div>
            <p class="text-xs text-slate-400 font-mono">UUID: %s</p>
        </div>`
)

// --- Structures ---

type PageData struct {
	Title   string
	Version string
}

// --- Entry Point ---

func main() {
	setupTempDir()

	// 1. Infrastructure (Adapters)
	soapAdapter := sat_adapter.NewSoapAdapter()

	// 2. Core (Service)
	conciliator := services.NewConciliatorService(soapAdapter)

	// 3. Presentation (Router & Handlers)
	mux := http.NewServeMux()
	setupStaticFiles(mux)
	setupRoutes(mux, conciliator)

	startServer(mux)
}

// --- Setup Functions ---

func setupTempDir() {
	_ = os.Mkdir(tempDir, 0755)
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
	render(w, homePath, PageData{Title: "SAT Reconciler", Version: "v1.0.0"})
}

func resumeHandler(w http.ResponseWriter, r *http.Request) {
	render(w, resumePath, PageData{Title: "Resume | Irene Olguin", Version: "v1.0.0"})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
		return
	}
	fmt.Fprint(w, htmlUploadSuccess)
}

// makeCheckStatusHandler crea el handler inyectando el servicio (Closure)
func makeCheckStatusHandler(service *services.ConciliatorService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, msgMethodNotAllowed, http.StatusMethodNotAllowed)
			return
		}

		if err := r.ParseMultipartForm(maxUploadSize); err != nil {
			http.Error(w, msgParseError, http.StatusBadRequest)
			return
		}

		// Extracción de datos del Formulario
		rfc := r.FormValue(fieldRFC)
		uuid := r.FormValue(fieldUUID)

		// Manejo de Archivos Temporales (Infraestructura Web -> Filesystem)
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

		// Llamada al Núcleo de Negocio
		statusMsg, err := service.VerifyRequest(rfc, uuid, certPath, keyPath)
		if err != nil {
			// En producción, loguearíamos el error real internamente
			fmt.Printf("Error en servicio: %v\n", err)
			http.Error(w, msgInternalError, http.StatusInternalServerError)
			return
		}

		// Renderizado de Respuesta (Presentación)
		fmt.Fprintf(w, htmlStatusResult, statusMsg, uuid)
	}
}

// --- Helper Functions ---

func render(w http.ResponseWriter, templatePath string, data PageData) {
	files := []string{layoutPath, templatePath}
	ts, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, msgInternalError, http.StatusInternalServerError)
		return
	}
	ts.ExecuteTemplate(w, "layout", data)
}

// saveTempFile guarda el archivo del multipart en disco y retorna su ruta + función de limpieza
func saveTempFile(r *http.Request, fieldName string) (string, func(), error) {
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return "", func() {}, err
	}
	defer file.Close()

	// Crear archivo temporal seguro
	tempFile, err := os.CreateTemp(tempDir, "sat-*"+filepath.Ext(header.Filename))
	if err != nil {
		return "", func() {}, err
	}
	defer tempFile.Close()

	if _, err := io.Copy(tempFile, file); err != nil {
		return "", func() {}, err
	}

	path := tempFile.Name()
	cleanup := func() { os.Remove(path) }

	return path, cleanup, nil
}
