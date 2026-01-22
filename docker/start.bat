@echo off
REM ================================
REM Code Relay Docker 一键启动脚本 (Windows)
REM ================================

echo ========================================
echo   Code Relay Docker 部署
echo ========================================
echo.

REM 切换到脚本所在目录
cd /d "%~dp0"

REM 检查 Docker 是否安装
docker --version >nul 2>&1
if %errorlevel% neq 0 (
    echo 错误: Docker 未安装，请先安装 Docker Desktop
    echo 访问: https://docs.docker.com/desktop/install/windows-install/
    pause
    exit /b 1
)

REM 检查 Docker Compose 是否可用
docker compose version >nul 2>&1
if %errorlevel% neq 0 (
    docker-compose --version >nul 2>&1
    if %errorlevel% neq 0 (
        echo 错误: Docker Compose 未安装
        pause
        exit /b 1
    )
    set DOCKER_COMPOSE=docker-compose
) else (
    set DOCKER_COMPOSE=docker compose
)

REM 检查是否存在 .env 文件
if not exist ".env" (
    echo 提示: 未找到 .env 文件，正在从模板创建...
    if exist ".env.example" (
        copy .env.example .env
        echo 已创建 .env 文件，请根据需要修改配置
    ) else (
        echo 警告: 未找到 .env.example，将使用默认配置
    )
)

REM 创建必要的目录
echo [1/4] 创建必要的目录...
if not exist "config" mkdir config
if not exist "logs" mkdir logs

echo.
echo [2/4] 构建 Docker 镜像...
%DOCKER_COMPOSE% build

echo.
echo [3/4] 启动服务...
%DOCKER_COMPOSE% up -d

echo.
echo [4/4] 检查服务状态...
timeout /t 3 /nobreak >nul
%DOCKER_COMPOSE% ps

echo.
echo ========================================
echo   部署完成！
echo ========================================
echo.
echo 服务访问地址:
echo   - 管理后台: http://localhost:8087
echo   - API 端点: http://localhost:8087/api
echo   - 健康检查: http://localhost:8087/api/internal/health
echo   - Codex 代理: http://localhost:8087/codex/v1
echo.
echo 常用命令:
echo   查看日志: %DOCKER_COMPOSE% logs -f
echo   停止服务: %DOCKER_COMPOSE% down
echo   重启服务: %DOCKER_COMPOSE% restart
echo   查看状态: %DOCKER_COMPOSE% ps
echo.
pause
