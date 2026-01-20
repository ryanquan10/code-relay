#!/bin/bash
set -e

echo "========================================"
echo "  Code Relay 一键构建脚本"
echo "========================================"
echo ""

echo "[1/3] 检查并安装前端依赖..."
cd frontend
if [ ! -d "node_modules" ]; then
    echo "正在安装前端依赖..."
    npm install
else
    echo "前端依赖已存在"
fi

echo ""
echo "[2/3] 构建前端..."
npm run build
cd ..
echo "前端构建完成！输出目录: frontend/dist"

echo ""
echo "[3/3] 构建 Go 后端（嵌入前端）..."
go build -ldflags="-s -w" -o code-relay .

echo ""
echo "========================================"
echo "  构建完成！"
echo "========================================"
echo "可执行文件: code-relay"
echo ""
echo "运行方式:"
echo "  ./code-relay"
echo "  或"
echo "  ./code-relay -config config.yaml"
echo ""
