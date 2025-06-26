@echo off
echo 编译文件打印服务...

REM 设置编译环境变量
set GOOS=windows
set GOARCH=386
set CGO_ENABLED=0

REM 设置兼容Windows 7的编译参数
set GOFLAGS=-ldflags="-s -w"

echo 编译参数:
echo GOOS=%GOOS%
echo GOARCH=%GOARCH%
echo CGO_ENABLED=%CGO_ENABLED%
echo GOFLAGS=%GOFLAGS%

REM 清理旧文件
if exist "printer.exe" del "printer.exe"

REM 编译程序
echo 正在编译...
go build -o printer.exe .

if errorlevel 1 (
    echo 编译失败！
    pause
    exit /b 1
)

echo 编译成功！
echo 生成文件: printer.exe
echo.
echo 现在可以运行 run.bat 来启动程序
pause 