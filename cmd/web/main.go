package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

// --- Constants & Configuration ---

const (
	// Server Config
	defaultPort = "3000"
	envPortKey  = "PORT"

	// Routes
	staticRoute = "/static/"
	homeRoute   = "/"
	resumeRoute = "/resume"
	uploadRoute = "/upload-fiel"
	checkRoute  = "/check-status"

	// Filesystem Paths
	staticDir  = "./web/static"
	layoutPath = "./web/templates/layout.html"
	homePath   = "./web/templates/index.html"
	resumePath = "./web/templates/resume.html"

	// Page Metadata
	appTitle    = "SAT Reconciler | Irene Olguin"
	resumeTitle = "Resume | Irene Olguin"
	appVersion  = "v1.0.0"

	// Form Handling
	maxUploadSize = 10 << 20 // 10MB
	fieldRFC      = "rfc_verify"
	fieldUUID     = "uuid_verify"
	fieldPass     = "password_verify"

	// Messages & Responses
	msgMethodNotAllowed = "Método no permitido"
	msgParseError       = "Error procesando el formulario o archivos"
	msgInternalError    = "Error interno del servidor"

	htmlUploadSuccess = `<div class="p-4 bg-green-100 text-green-700 rounded border border-green-400">✅ Archivos recibidos (Simulación)</div>`

	// HTML Template para la respuesta simulada (movido aquí para limpiar el handler)
	htmlStatusSimulated = `
        <div class="mt-4 p-4 bg-slate-900 rounded border border-slate-700">
            <div class="flex items-center gap-3 mb-2">
                <div class="w-3 h-3 rounded-full bg-yellow-500 animate-pulse"></div>
                <span class="text-white font-bold">Estado: En Proceso (2)</span>
            </div>
            <p class="text-xs text-slate-400 font-mono">UUID: %s</p>
            <p class="text-xs text-slate-400">El SAT está procesando la solicitud. Intenta en 1 min.</p>
        </div>`
)

// --- Data Structures ---

type PageData struct {
	Title   string
	Version string
}

// --- Entry Point ---

func main() {
	setupStaticFiles()
	setupRoutes()
	startServer()
}

// --- Setup Functions ---

func setupStaticFiles() {
	fs := http.FileServer(http.Dir(staticDir))
	http.Handle(staticRoute, http.StripPrefix(staticRoute, fs))
}

func setupRoutes() {
	http.HandleFunc(homeRoute, homeHandler)
	http.HandleFunc(resumeRoute, resumeHandler)
	http.HandleFunc(uploadRoute, uploadHandler)
	http.HandleFunc(checkRoute, checkStatusHandler)
}

func startServer() {
	port := getServerPort()
	logStartup(port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func getServerPort() string {
	if port := os.Getenv(envPortKey); port != "" {
		return port
	}
	return defaultPort
}

func logStartup(port string) {
	fmt.Printf("🐺 Irene Olguin - SAT Reconciler Web %s\n", appVersion)
	fmt.Printf("🚀 Servidor corriendo en http://localhost:%s\n", port)
}

// --- HTTP Handlers ---

func homeHandler(w http.ResponseWriter, r *http.Request) {
	render(w, homePath, PageData{
		Title:   appTitle,
		Version: appVersion,
	})
}

func resumeHandler(w http.ResponseWriter, r *http.Request) {
	render(w, resumePath, PageData{
		Title:   resumeTitle,
		Version: appVersion,
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureMethod(w, r, http.MethodPost) {
		return
	}
	fmt.Fprint(w, htmlUploadSuccess)
}

func checkStatusHandler(w http.ResponseWriter, r *http.Request) {
	if !ensureMethod(w, r, http.MethodPost) {
		return
	}

	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, msgParseError, http.StatusBadRequest)
		return
	}

	uuid := r.FormValue(fieldUUID)

	// TODO: Aquí conectaríamos con el servicio real de verificación
	// Por ahora, renderizamos la respuesta simulada usando la constante
	fmt.Fprintf(w, htmlStatusSimulated, uuid)
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
		http.Error(w, msgInternalError+": "+err.Error(), http.StatusInternalServerError)
		return
	}

	if err := ts.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, msgInternalError, http.StatusInternalServerError)
	}
}
