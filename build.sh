#!/bin/bash

# --- Version Check ---
REQUIRED_GO_VERSION="go1.20"
CURRENT_GO_VERSION=$(go version | awk '{print $3}')

if [[ ! "$CURRENT_GO_VERSION" == *"$REQUIRED_GO_VERSION"* ]]; then
    echo "[ERROR] Incorrect Go version for Windows 7 compatibility."
    echo "Required: $REQUIRED_GO_VERSION.x"
    echo "You have: $CURRENT_GO_VERSION"
    echo "Please install Go 1.20 and ensure it is in your PATH."
    exit 1
fi

echo "Compiling for Windows 7 (32-bit) with compatible Go version..."

# 设置编译环境变量
export GOOS=windows
export GOARCH=386
export CGO_ENABLED=0

# 设置输出文件名
OUTPUT_NAME="printer_win7.exe"

# 编译参数, -s -w 用于减小文件大小
LDFLAGS="-s -w"

echo "编译参数:"
echo "GOOS=${GOOS}"
echo "GOARCH=${GOARCH}"
echo "CGO_ENABLED=${CGO_ENABLED}"
echo "LDFLAGS=${LDFLAGS}"

# 清理旧文件
if [ -f "$OUTPUT_NAME" ]; then
    echo "删除旧的编译文件: $OUTPUT_NAME"
    rm "$OUTPUT_NAME"
fi

# 编译程序
echo "正在编译..."
go build -o "$OUTPUT_NAME" -ldflags="$LDFLAGS" .

# 检查编译是否成功
if [ $? -eq 0 ]; then
    echo "编译成功！"
    echo "生成文件: $OUTPUT_NAME"
else
    echo "编译失败！"
    exit 1
fi
