//go:build windows

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

	"github.com/alexbrainman/printer"
	"github.com/google/uuid"
)

// listPrinters handles requests to list all available printers on the system.
func listPrinters(w http.ResponseWriter, r *http.Request) {
	printers, err := printer.ReadNames()
	if err != nil {
		log.Printf("Failed to read printer names: %v", err)
		writeJSONError(w, "Could not list printers", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, printers, http.StatusOK)
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

	if printerToUse == "" {
		defaultPrinter, err := printer.Default()
		if err != nil {
			log.Printf("Could not get default printer: %v. Please specify a printer.", err)
			writeJSONError(w, "Could not get default printer. Please specify a printer.", http.StatusInternalServerError)
			return
		}
		printerToUse = defaultPrinter
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	imageExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".bmp": true, ".gif": true, ".tif": true, ".tiff": true}

	if ext == ".pdf" || imageExts[ext] {
		// 使用SumatraPDF.exe静默打印PDF和图片，因为这是最可靠的方式
		sumatraPath, err := filepath.Abs("SumatraPDF.exe")
		if err != nil {
			log.Printf("获取SumatraPDF.exe绝对路径失败: %v", err)
			writeJSONError(w, "无法获取SumatraPDF.exe绝对路径", http.StatusInternalServerError)
			return
		}
		args := []string{"-print-to-default", filePath, "-silent"}
		if printerToUse != "" {
			args = []string{"-print-to", printerToUse, filePath, "-silent"}
		}
		cmd := exec.Command(sumatraPath, args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("SumatraPDF打印失败: %v Output: %s", err, string(output))
			writeJSONError(w, fmt.Sprintf("SumatraPDF打印失败: %v Output: %s", err, string(output)), http.StatusInternalServerError)
			return
		}
		log.Printf("SumatraPDF打印命令已发送: %s", string(output))
		writeJSONResponse(w, map[string]string{"message": "打印任务已发送。"}, http.StatusOK)
		return
	} else {
		// 对于其他文件类型（如Word文档），使用PowerShell
		var psCommand string
		if printerToUse != "" {
			psCommand = fmt.Sprintf(`Start-Process -FilePath "%s" -Verb PrintTo -ArgumentList '"%s"' -WindowStyle Hidden -PassThru | Wait-Process`, filePath, printerToUse)
		} else {
			psCommand = fmt.Sprintf(`Start-Process -FilePath "%s" -Verb Print -WindowStyle Hidden`, filePath)
		}

		cmd := exec.Command("powershell", "-Command", psCommand)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("PowerShell print command failed: %v Output: %s", err, string(output))
			writeJSONError(w, fmt.Sprintf("Failed to print file using PowerShell: %v Output: %s", err, string(output)), http.StatusInternalServerError)
			return
		}

		log.Printf("Print command sent via PowerShell for file '%s' to printer '%s'. Output: %s", filePath, printerToUse, string(output))
		writeJSONResponse(w, map[string]string{
			"message": "Print job sent successfully via PowerShell.",
			"details": fmt.Sprintf("File: %s, Printer: %s", filePath, printerToUse),
		}, http.StatusOK)
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
