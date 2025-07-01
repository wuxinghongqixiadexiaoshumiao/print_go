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
	"runtime"
	"strings"

	"github.com/alexbrainman/printer"
	"github.com/google/uuid"
)

// findBrowser attempts to find an installed web browser executable on Windows.
func findBrowser() string {
	browserExecutables := []string{"msedge.exe", "chrome.exe", "firefox.exe"}
	searchPaths := []string{
		os.Getenv("ProgramFiles"),
		os.Getenv("ProgramFiles(x86)"),
		os.Getenv("LOCALAPPDATA"),
	}
	// Use a map for easier lookup of subpaths
	browserSubPaths := map[string][]string{
		"msedge.exe":  {"Microsoft", "Edge", "Application"},
		"chrome.exe":  {"Google", "Chrome", "Application"},
		"firefox.exe": {"Mozilla Firefox"},
	}

	for _, executable := range browserExecutables {
		for _, searchPath := range searchPaths {
			if searchPath == "" {
				continue
			}
			// Build path safely using filepath.Join
			pathParts := append([]string{searchPath}, browserSubPaths[executable]...)
			pathParts = append(pathParts, executable)
			browserPath := filepath.Join(pathParts...)

			if _, err := os.Stat(browserPath); err == nil {
				log.Printf("Found browser: %s", browserPath)
				return browserPath
			}
		}
	}

	log.Println("No supported browser found on the system.")
	return ""
}

// listPrinters handles requests to list all available printers on the system.
func listPrinters(w http.ResponseWriter, r *http.Request) {
	if runtime.GOOS != "windows" {
		writeJSONError(w, "Listing printers is only supported on Windows", http.StatusNotImplemented)
		return
	}

	printers, err := printer.ReadNames()
	if err != nil {
		log.Printf("Failed to read printer names: %v", err)
		writeJSONError(w, "Could not list printers", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, printers, http.StatusOK)
}

// handlePrint handles the printing request, prioritizing browsers for known file types.
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

	switch runtime.GOOS {
	case "windows":
		printerToUse := req.PrinterName
		if printerToUse == "" {
			defaultPrinter, err := printer.Default()
			if err != nil {
				log.Printf("Could not get default printer: %v. Proceeding without a specific printer.", err)
			} else {
				printerToUse = defaultPrinter
			}
		}

		var cmd *exec.Cmd
		fileExt := strings.ToLower(filepath.Ext(filePath))
		browserPrintableTypes := []string{".pdf", ".png", ".jpg", ".jpeg", ".gif", ".webp", ".bmp"}

		isBrowserPrintable := false
		for _, ext := range browserPrintableTypes {
			if fileExt == ext {
				isBrowserPrintable = true
				break
			}
		}

		if isBrowserPrintable {
			browserPath := findBrowser()
			if browserPath != "" {
				log.Printf("Attempting to print file '%s' using browser '%s'", filePath, browserPath)
				browserName := strings.ToLower(filepath.Base(browserPath))
				args := []string{}
				if browserName == "firefox.exe" {
					args = append(args, "-print", filePath)
					if printerToUse != "" {
						args = append(args, "-print-to", printerToUse)
					}
				} else { // For Chrome/Edge and other Chromium-based browsers.
					args = append(args, "--kiosk-printing")
					if printerToUse != "" {
						args = append(args, "--print-to=\""+printerToUse+"\"")
					}
					args = append(args, filePath)
				}
				cmd = exec.Command(browserPath, args...)
			}
		}

		// Fallback to PowerShell if browser printing is not applicable or fails
		if cmd == nil {
			log.Printf("Browser not found or file type not supported by browser, falling back to PowerShell...")
			var psCommand string
			if printerToUse != "" {
				log.Printf("Attempting to print file '%s' on printer '%s' via PowerShell...", filePath, printerToUse)
				psCommand = fmt.Sprintf(`Start-Process -FilePath "%s" -Verb Print -ArgumentList '"%s"' -WindowStyle Hidden`, filePath, printerToUse)
			} else {
				log.Printf("Attempting to print file '%s' using default system method via PowerShell...", filePath)
				psCommand = fmt.Sprintf(`Start-Process -FilePath "%s" -Verb Print -WindowStyle Hidden`, filePath)
			}
			cmd = exec.Command("powershell", "-Command", psCommand)
		}

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Print command failed: %v", err)
			log.Printf("Command output: %s", string(output))
			writeJSONError(w, fmt.Sprintf("Printing failed: %v\nOutput: %s", err, string(output)), http.StatusInternalServerError)
			return
		}

		log.Printf("Print command sent successfully. Output: %s", string(output))
		writeJSONResponse(w, map[string]string{
			"message": "Print job sent to the specified printer.",
			"details": fmt.Sprintf("File: %s, Printer: %s", filePath, printerToUse),
		}, http.StatusOK)

	case "darwin":
		cmd := exec.Command("open", filePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to open file: %v\nOutput: %s", err, string(output))
			writeJSONError(w, fmt.Sprintf("Failed to open file: %v\nOutput: %s", err, string(output)), http.StatusInternalServerError)
			return
		}
		log.Printf("File opened: %s", string(output))
		writeJSONResponse(w, map[string]string{"message": "File opened"}, http.StatusOK)

	default:
		writeJSONError(w, "Unsupported operating system", http.StatusInternalServerError)
	}
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
