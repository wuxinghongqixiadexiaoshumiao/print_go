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

	"github.com/google/uuid"
)

// handlePrint 使用标准库 net/http 处理打印请求
func handlePrint(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileName string `json:"fileName"`
		URL      string `json:"url"`
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

	// 转换为 Windows 路径格式
	filePath = strings.ReplaceAll(filePath, "/", "\\")
	log.Printf("Full file path for printing: %s", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File does not exist: %s", filePath)
		writeJSONError(w, "File not found", http.StatusNotFound)
		return
	}

	// 根据操作系统执行打印或打开命令
	switch runtime.GOOS {
	case "windows":
		log.Printf("Attempting to print file '%s' using default printer...", filePath)
		psCommand := fmt.Sprintf(`Start-Process -FilePath "%s" -Verb Print -WindowStyle Hidden`, filePath)
		cmd := exec.Command("powershell", "-Command", psCommand)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Print command failed: %v", err)
			log.Printf("Command output: %s", string(output))
			writeJSONError(w, fmt.Sprintf("Printing failed: %v\nOutput: %s", err, string(output)), http.StatusInternalServerError)
			return
		}

		if len(output) > 0 {
			log.Printf("Print command output: %s", string(output))
		} else {
			log.Printf("Print command sent successfully with no output.")
		}

		writeJSONResponse(w, map[string]string{
			"message": "Print job sent to the default printer.",
			"details": fmt.Sprintf("File: %s", filePath),
		}, http.StatusOK)

	case "darwin": // macOS
		cmd := exec.Command("open", filePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("Failed to open file: %v\nOutput: %s", err, string(output))
			writeJSONError(w, fmt.Sprintf("Failed to open file: %v\nOutput: %s", err, string(output)), http.StatusInternalServerError)
			return
		}
		log.Printf("File opened: %s", string(output))
		writeJSONResponse(w, map[string]string{
			"message": "File opened. Please use the system's print dialog.",
		}, http.StatusOK)

	default:
		writeJSONError(w, "Unsupported operating system", http.StatusInternalServerError)
	}
}

// downloadFile 从URL下载文件并返回其本地路径
func downloadFile(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != 200 {
		if err == nil {
			return "", fmt.Errorf("bad status: %s", resp.Status)
		}
		return "", err
	}
	defer resp.Body.Close()

	ext := filepath.Ext(url)
	if ext == "" {
		ext = ".dat" // Default extension
	}
	fileName := fmt.Sprintf("%s%s", generateUUID(), ext)
	filePath := filepath.Join(uploadDir, fileName)

	out, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return "", err
	}
	log.Printf("File saved from URL: %s", filePath)
	return filePath, nil
}

// generateUUID 生成一个唯一的字符串ID
func generateUUID() string {
	return strings.ReplaceAll(fmt.Sprintf("%s", uuid.New()), "-", "")
}
