#!/bin/bash

echo "Compiling DIAGNOSTIC TEST for Windows 7 (32-bit)..."

# Set build environment variables
export GOOS=windows
export GOARCH=386
export CGO_ENABLED=0

# Set output file name
OUTPUT_NAME="diag_printer_test.exe"

# Build flags to reduce file size
LDFLAGS="-s -w"

echo "Compiling diag_printer_test.go inside ./diag_tool/ directory..."

# Change to the diagnostic tool directory to compile
cd diag_tool || exit

go build -o "../$OUTPUT_NAME" -ldflags="$LDFLAGS" .

# Check if compilation was successful
if [ $? -eq 0 ]; then
    echo "Compilation successful!"
    echo "Generated file: $OUTPUT_NAME"
    echo "Please run this file on your Windows 7 machine and report the output."
else
    echo "Compilation FAILED!"
    exit 1
fi
