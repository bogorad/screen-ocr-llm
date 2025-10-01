@echo off
REM Build script for screen-ocr-llm
REM This ensures we build from the correct main package

echo Building screen-ocr-llm...
go build -ldflags "-H=windowsgui" -o screen-ocr-llm.exe ./main

if %ERRORLEVEL% EQU 0 (
    echo.
    echo ✅ Build successful: screen-ocr-llm.exe
    echo.
    echo To run: .\screen-ocr-llm.exe
    echo.
) else (
    echo.
    echo ❌ Build failed!
    echo.
    exit /b 1
)

