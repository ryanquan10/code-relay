@echo off
chcp 65001 >nul 2>&1
echo ========================================
echo   Code Relay Build Script
echo ========================================
echo.

echo [1/3] Checking frontend dependencies...
cd frontend
if not exist "node_modules" (
    echo Installing frontend dependencies...
    call npm install
    if errorlevel 1 (
        echo Frontend dependency installation failed!
        pause
        exit /b 1
    )
) else (
    echo Frontend dependencies already exist
)

echo.
echo [2/3] Building frontend...
call npm run build
if errorlevel 1 (
    echo Frontend build failed!
    cd ..
    pause
    exit /b 1
)
cd ..
echo Frontend build complete! Output directory: frontend\dist

echo.
echo [3/3] Building Go backend (embedding frontend)...
go build -ldflags="-s -w" -o code-relay.exe .
if errorlevel 1 (
    echo Go build failed!
    pause
    exit /b 1
)

echo.
echo ========================================
echo   Build complete!
echo ========================================
echo Executable: code-relay.exe
echo.
echo Usage:
echo   code-relay.exe
echo   or
echo   code-relay.exe -config config.yaml
echo.
pause
