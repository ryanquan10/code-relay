@echo off
echo ======================================
echo   本地 IP 监控服务 (NSLookup DDNS)
echo ======================================
echo.
nslookup.exe -config ../config/app.yaml
pause
