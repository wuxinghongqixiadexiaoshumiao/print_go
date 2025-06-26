package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
)

//go:embed templates
var templatesFS embed.FS

const (
	uploadDir = "./uploads"
)

var templates *template.Template

func main() {
	// Setup logging
	logFile, err := os.OpenFile("printer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	} else {
		log.SetOutput(os.Stdout)
	}

	log.Printf("Application starting - OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH)

	// Windows 7 compatibility: Disable HTTP/2
	if runtime.GOOS == "windows" {
		os.Setenv("GODEBUG", "http2server=0")
	}

	// Ensure the upload directory exists
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Parse templates
	templates = template.Must(template.ParseFS(templatesFS, "templates/*"))

	// Setup HTTP server and routes
	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/upload", handleFileUpload)
	mux.HandleFunc("/files", listFiles)
	mux.HandleFunc("/print", handlePrint)

	// Serve static files if a 'static' directory exists
	if _, err := os.Stat("./static"); !os.IsNotExist(err) {
		mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))
		log.Println("Serving static files from ./static/")
	}

	log.Println("Server starting on port 8081...")
	if err := http.ListenAndServe(":8081", mux); err != nil {
		log.Printf("FATAL: Server failed to start: %v", err)
		log.Println("Press Enter to exit...")
		fmt.Scanln()
	}
}

// handleIndex serves the main HTML page.
func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	templates.ExecuteTemplate(w, "index.html", map[string]interface{}{
		"title": "File Management System",
	})
}

// writeJSONError is a helper to write a JSON error message to the response.
func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

// writeJSONResponse is a helper to write a JSON response.
func writeJSONResponse(w http.ResponseWriter, data interface{}, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}
