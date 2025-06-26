package main

import (
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gin-gonic/gin"
)

//go:embed templates
var templatesFS embed.FS

const (
	uploadDir = "./uploads"
)

func main() {
	// 设置日志输出到文件，防止闪退时看不到错误信息
	logFile, err := os.OpenFile("printer.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		defer logFile.Close()
		log.SetOutput(logFile)
	}

	// 添加启动日志
	log.Printf("程序启动 - 操作系统: %s, 架构: %s", runtime.GOOS, runtime.GOARCH)

	// Windows 7 兼容性设置
	if runtime.GOOS == "windows" {
		// 设置环境变量以兼容旧版本Windows
		os.Setenv("GODEBUG", "http2server=0")
	}

	// 确保上传目录存在
	if err := createUploadDir(); err != nil {
		log.Printf("无法创建上传目录: %v", err)
		// 不要立即退出，尝试继续运行
	}

	r := gin.Default()

	// 设置gin为发布模式，减少日志输出
	gin.SetMode(gin.ReleaseMode)

	// 设置静态文件服务
	r.Static("/static", "./static")

	// 使用嵌入的模板文件
	templ := template.Must(template.ParseFS(templatesFS, "templates/*"))
	r.SetHTMLTemplate(templ)

	// 文件上传和管理路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", gin.H{
			"title": "文件管理系统",
		})
	})

	r.POST("/upload", handleFileUpload)
	r.GET("/files", listFiles)
	r.POST("/print", handlePrint)

	// 启动服务器
	log.Printf("服务器启动在端口 18080")
	if err := r.Run(":18080"); err != nil {
		log.Printf("服务器启动失败: %v", err)
		// 保持窗口打开，让用户看到错误信息
		log.Printf("按任意键退出...")
		var input string
		fmt.Scanln(&input)
	}
}

func createUploadDir() error {
	return filepath.Walk(uploadDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return os.MkdirAll(uploadDir, 0755)
		}
		return nil
	})
}
