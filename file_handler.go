package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type FileInfo struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// handleFileUpload 使用标准库 net/http 处理文件上传
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	// 解析 multipart form, 10 MB 的内存限制
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSONError(w, "Could not parse multipart form", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		writeJSONError(w, "Could not retrieve file from form-data", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 生成唯一文件名
	ext := filepath.Ext(handler.Filename)
	newFileName := fmt.Sprintf("%s%s", generateUUID(), ext)
	filePath := filepath.Join(uploadDir, newFileName)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSONError(w, "Could not create file on server", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 将上传的文件内容复制到目标文件
	if _, err := io.Copy(dst, file); err != nil {
		writeJSONError(w, "Could not save uploaded file", http.StatusInternalServerError)
		return
	}

	log.Printf("Successfully uploaded file: %s as %s", handler.Filename, newFileName)
	writeJSONResponse(w, map[string]interface{}{
		"message": "File uploaded successfully",
		"file": FileInfo{
			Name: handler.Filename,
			Path: newFileName,
		},
	}, http.StatusOK)
}

// listFiles 使用标准库 net/http 列出已上传的文件
func listFiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var files []FileInfo

	err := filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			files = append(files, FileInfo{
				Name: info.Name(),
				Path: strings.TrimPrefix(path, uploadDir+"/"),
			})
		}
		return nil
	})

	if err != nil {
		log.Printf("Failed to list files: %v", err)
		writeJSONError(w, "Could not list files", http.StatusInternalServerError)
		return
	}

	writeJSONResponse(w, files, http.StatusOK)
}
