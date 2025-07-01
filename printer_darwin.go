//go:build darwin

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// listPrinters handles requests to list all available printers on the system.
func listPrinters(w http.ResponseWriter, r *http.Request) {
	writeJSONError(w, "Listing printers is not supported on this OS", http.StatusNotImplemented)
}

// handlePrint handles the printing request directly to the printer.
func handlePrint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileName    string `json:"fileName"`
		URL         string `json:"url"`
		PrinterName string `json:"printerName,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Print request JSON decoding error: %v", err)
		writeJSONError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if (req.FileName == "" && req.URL == "") || (req.FileName != "" && req.URL != "") {
		writeJSONError(w, "fileName and url are mutually exclusive", http.StatusBadRequest)
		return
	}

	var filePath string
	if req.URL != "" {
		var err error
		filePath, err = downloadFile(req.URL)
		if err != nil {
			log.Printf("Failed to download file: %v", err)
			writeJSONError(w, "Failed to download file from URL", http.StatusInternalServerError)
			return
		}
	} else {
		filePath = filepath.Join(uploadDir, req.FileName)
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("Failed to get absolute path for %s: %v", filePath, err)
		absFilePath = filePath
	}
	filePath = absFilePath

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", filePath)
		writeJSONError(w, "File not found", http.StatusNotFound)
		return
	}

	printerToUse := req.PrinterName

	var cmd *exec.Cmd
	if printerToUse != "" {
		cmd = exec.Command("lp", "-d", printerToUse, filePath)
	} else {
		cmd = exec.Command("lp", filePath)
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to print file: %v Output: %s", err, string(output))
		writeJSONError(w, fmt.Sprintf("Failed to print file: %v Output: %s", err, string(output)), http.StatusInternalServerError)
		return
	}
	log.Printf("Print command sent: %s", string(output))
	writeJSONResponse(w, map[string]string{"message": "Print command sent successfully."}, http.StatusOK)
}

// downloadFile downloads a file from a URL and saves it locally.
func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status: %s", resp.Status)
	}

	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".dat"
	}
	fileName := fmt.Sprintf("%s%s", generateUUID(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return filePath, err
}

// generateUUID creates a unique string ID.
func generateUUID() string {
	return strings.ReplaceAll(uuid.New().String(), "-", "")
}
