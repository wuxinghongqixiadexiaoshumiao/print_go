@echo off
echo 启动文件打印服务...
echo 操作系统: %OS%
echo 架构: %PROCESSOR_ARCHITECTURE%

REM 检查是否存在printer.exe
if not exist "printer.exe" (
    echo 错误: 找不到 printer.exe 文件
    echo 请先编译程序: go build -o printer.exe .
    pause
    exit /b 1
)

REM 运行程序
echo 正在启动程序...
printer.exe

REM 如果程序异常退出，显示错误信息
if errorlevel 1 (
    echo.
    echo 程序异常退出，错误代码: %errorlevel%
    echo 请查看 printer.log 文件获取详细错误信息
)

echo.
echo 程序已退出，按任意键关闭窗口...
pause 