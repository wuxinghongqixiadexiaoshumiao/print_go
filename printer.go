package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// 获取系统打印机列表
func getPrinters() ([]string, error) {
	if runtime.GOOS != "windows" {
		return nil, fmt.Errorf("不支持的操作系统")
	}

	// 使用 WMI 获取打印机列表，兼容 Windows 7
	cmd := exec.Command("wmic", "printer", "get", "name", "/format:csv")
	output, err := cmd.Output()
	if err != nil {
		// 如果 wmic 失败，尝试使用 PowerShell 2.0 兼容命令
		psCmd := exec.Command("powershell", "-Command", "Get-WmiObject -Class Win32_Printer | Select-Object -ExpandProperty Name")
		output, err = psCmd.Output()
		if err != nil {
			return nil, fmt.Errorf("获取打印机列表失败: %v", err)
		}
	}

	// 处理输出
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var printers []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 跳过标题行和空行
		if line != "" && line != "Name" && !strings.Contains(line, "Node,") {
			// 对于 wmic csv 格式，提取打印机名称
			if strings.Contains(line, ",") {
				parts := strings.Split(line, ",")
				if len(parts) >= 2 {
					printerName := strings.TrimSpace(parts[1])
					if printerName != "" {
						printers = append(printers, printerName)
					}
				}
			} else {
				// 对于 PowerShell 输出，直接使用
				printers = append(printers, line)
			}
		}
	}

	return printers, nil
}

func handlePrint(c *gin.Context) {
	var req struct {
		FileName string `json:"fileName"`
		URL      string `json:"url"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		log.Printf("打印请求参数错误: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if (req.FileName == "" && req.URL == "") || (req.FileName != "" && req.URL != "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "fileName和url必须且只能传一个"})
		return
	}

	var filePath string
	var fileName string
	if req.URL != "" {
		// 下载url文件到uploads目录
		resp, err := http.Get(req.URL)
		if err != nil || resp.StatusCode != 200 {
			log.Printf("下载URL失败: %v", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "下载URL失败"})
			return
		}
		defer resp.Body.Close()

		// 尝试获取文件扩展名
		ext := filepath.Ext(req.URL)
		if ext == "" {
			ext = ".dat"
		}
		fileName = fmt.Sprintf("%s%s", generateUUID(), ext)
		filePath = filepath.Join(uploadDir, fileName)
		out, err := os.Create(filePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存下载文件失败"})
			return
		}
		defer out.Close()
		_, err = io.Copy(out, resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写入文件失败"})
			return
		}
		log.Printf("URL文件已保存: %s", filePath)
	} else {
		fileName = req.FileName
		filePath = filepath.Join(uploadDir, req.FileName)
	}

	// 转换为 Windows 路径格式
	filePath = strings.ReplaceAll(filePath, "/", "\\")
	log.Printf("完整文件路径: %s", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("文件不存在: %s", filePath)
		c.JSON(http.StatusNotFound, gin.H{"error": "文件不存在"})
		return
	}

	// 根据操作系统选择不同的打印命令
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		// 获取打印机列表
		printers, err := getPrinters()
		if err != nil {
			log.Printf("获取打印机列表失败: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("获取打印机列表失败: %v", err),
			})
			return
		}

		if len(printers) == 0 {
			log.Printf("未找到打印机")
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "未找到打印机",
			})
			return
		}

		// 使用默认打印机
		defaultPrinter := printers[0]
		log.Printf("使用默认打印机: %s", defaultPrinter)

		// 使用 PowerShell 发送打印任务，兼容 Windows 7
		psCommand := fmt.Sprintf(`Start-Process -FilePath "%s" -Verb Print -WindowStyle Hidden`, filePath)
		cmd = exec.Command("powershell", "-Command", psCommand)

		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("打印命令执行失败: %v", err)
			log.Printf("命令输出: %s", string(output))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("打印失败: %v\n输出: %s", err, string(output)),
			})
			return
		}

		if len(output) > 0 {
			log.Printf("打印命令输出: %s", string(output))
		} else {
			log.Printf("打印命令执行成功，无输出")
		}

		// 返回打印机信息
		c.JSON(http.StatusOK, gin.H{
			"message": "打印任务已发送",
			"details": fmt.Sprintf("文件: %s\n打印机: %s", filePath, defaultPrinter),
			"printer": defaultPrinter,
		})
		return

	case "darwin": // macOS
		cmd = exec.Command("open", filePath)
		output, err := cmd.CombinedOutput()
		if err != nil {
			log.Printf("打开文件失败: %v\n输出: %s", err, string(output))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("打开文件失败: %v\n输出: %s", err, string(output)),
			})
			return
		}
		log.Printf("文件已打开: %s", string(output))
		c.JSON(http.StatusOK, gin.H{
			"message": "文件已打开，请使用系统打印功能",
		})
		return

	default:
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "不支持的操作系统",
		})
		return
	}
}

// 生成UUID
func generateUUID() string {
	return strings.ReplaceAll(fmt.Sprintf("%s", uuid.New()), "-", "")
}
