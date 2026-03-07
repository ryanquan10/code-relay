#!/bin/bash
set -e

# ================================
# Code Relay Docker 一键启动脚本
# ================================

echo "========================================"
echo "  Code Relay Docker 部署"
echo "========================================"
echo ""

# 切换到 docker 目录
cd "$(dirname "$0")"

# 拉取当前分支最新代码
if command -v git > /dev/null 2>&1 && git rev-parse --is-inside-work-tree > /dev/null 2>&1; then
    CURRENT_BRANCH="$(git rev-parse --abbrev-ref HEAD)"
    echo "[1/5] 拉取当前分支最新代码: ${CURRENT_BRANCH}"
    git pull origin "${CURRENT_BRANCH}"
else
    echo "[1/5] 未检测到 Git 仓库或 Git 命令，跳过 git pull"
fi

# 检查 Docker 是否安装
if ! command -v docker > /dev/null 2>&1; then
    echo "错误: Docker 未安装，请先安装 Docker"
    echo "访问: https://docs.docker.com/get-docker/"
    exit 1
fi

# 检查 Docker Compose 是否安装
if ! command -v docker-compose > /dev/null 2>&1 && ! docker compose version > /dev/null 2>&1; then
    echo "错误: Docker Compose 未安装"
    echo "访问: https://docs.docker.com/compose/install/"
    exit 1
fi

# 检查是否存在 .env 文件
if [ ! -f ".env" ]; then
    echo "提示: 未找到 .env 文件，正在从模板创建..."
    if [ -f ".env.example" ]; then
        cp .env.example .env
        echo "已创建 .env 文件，请根据需要修改配置"
    else
        echo "警告: 未找到 .env.example，将使用默认配置"
    fi
fi

# 创建必要的目录
echo "[2/5] 创建必要的目录..."
mkdir -p config logs

echo ""
echo "[3/5] 构建 Docker 镜像..."
# 使用 docker compose 或 docker-compose
if docker compose version > /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

$DOCKER_COMPOSE build

echo ""
echo "[4/5] 启动服务..."
$DOCKER_COMPOSE up -d

echo ""
echo "[5/5] 检查服务状态..."
sleep 3
$DOCKER_COMPOSE ps

echo ""
echo "========================================"
echo "  部署完成！"
echo "========================================"
echo ""
echo "服务访问地址:"
echo "  - 管理后台: http://localhost:8087"
echo "  - API 端点: http://localhost:8087/api"
echo "  - 健康检查: http://localhost:8087/api/internal/health"
echo "  - Codex 代理: http://localhost:8087/codex/v1"
echo ""
echo "常用命令:"
echo "  查看日志: $DOCKER_COMPOSE logs -f"
echo "  停止服务: $DOCKER_COMPOSE down"
echo "  重启服务: $DOCKER_COMPOSE restart"
echo "  查看状态: $DOCKER_COMPOSE ps"
echo ""
