package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
)

// Estructura para pasar datos a la vista
type PageData struct {
	Title   string
	Version string
}

func main() {
	// 1. Configurar rutas de archivos estáticos (CSS, JS, Logos)
	// Esto servirá lo que pongas en /web/static
	fs := http.FileServer(http.Dir("./web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// 2. Ruta Principal (Landing / Dashboard)
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/resume", handleResume)

	// 3. Ruta de Carga de Archivos (El endpoint que recibe la FIEL)
	http.HandleFunc("/upload-fiel", handleUpload)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default para local
	}

	fmt.Printf("🐺 Irene Olguin - SAT Reconciler Web v1.0\n")
	fmt.Printf("🚀 Servidor corriendo en http://localhost%s\n", port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	// Parsear los templates (Layout + Contenido)
	files := []string{
		"./web/templates/layout.html",
		"./web/templates/index.html",
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, "Error interno: "+err.Error(), 500)
		return
	}

	data := PageData{
		Title:   "SAT Reconciler | Irene Olguin",
		Version: "v0.1.0-alpha",
	}

	// Ejecutar el template "layout"
	ts.ExecuteTemplate(w, "layout", data)
}

func handleUpload(w http.ResponseWriter, r *http.Request) {
	// Aquí procesaremos los archivos más tarde
	if r.Method != "POST" {
		http.Error(w, "Método no permitido", 405)
		return
	}
	fmt.Fprint(w, `<div class="p-4 bg-green-100 text-green-700 rounded border border-green-400">✅ Archivos recibidos (Simulación)</div>`)
}

// Handler para el CV
func handleResume(w http.ResponseWriter, r *http.Request) {
	files := []string{
		"./web/templates/layout.html",
		"./web/templates/resume.html", // Usa el template del resume
	}

	ts, err := template.ParseFiles(files...)
	if err != nil {
		http.Error(w, "Error cargando CV: "+err.Error(), 500)
		return
	}

	data := PageData{
		Title:   "Resume | Irene Olguin",
		Version: "v1.0.0",
	}

	ts.ExecuteTemplate(w, "layout", data)
}
