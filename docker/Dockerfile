# ================================
# 多阶段构建 Dockerfile
# ================================

# -------------------- 阶段 1: 前端构建 --------------------
FROM docker.m.daocloud.io/library/node:20.19.0-alpine AS frontend-builder

WORKDIR /app/internal/frontend

# 复制前端依赖文件
COPY internal/frontend/package*.json ./

# 安装依赖（使用国内镜像加速）
RUN npm config set registry https://registry.npmmirror.com && \
    npm install

# 复制前端源码
COPY internal/frontend/ ./

# 构建前端
RUN npm run build

# -------------------- 阶段 2: Go 后端构建 --------------------
FROM docker.m.daocloud.io/library/golang:1.24.0-alpine AS backend-builder

WORKDIR /app

# 安装构建依赖
RUN apk add --no-cache git ca-certificates tzdata

# 配置 Go 环境（使用国内镜像）
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.cn,direct \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64

# 复制 go.mod 和 go.sum
COPY go.mod go.sum ./

# 下载依赖
RUN go mod download

# 复制后端源码
COPY . .

# 从前端构建阶段复制构建好的静态文件
COPY --from=frontend-builder /app/internal/frontend/dist ./internal/frontend/dist

# 构建 Go 应用（使用 -s -w 减小体积）
RUN go build -ldflags="-s -w" -o code-relay .

# -------------------- 阶段 3: 最终运行镜像 --------------------
FROM docker.m.daocloud.io/library/alpine:latest

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache ca-certificates tzdata && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 从构建阶段复制可执行文件
COPY --from=backend-builder /app/code-relay .

# 复制配置文件目录（可选，如果需要默认配置）
RUN mkdir -p /app/config

# 创建非 root 用户
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser && \
    chown -R appuser:appuser /app

USER appuser

# 暴露端口（默认 8087，可通过环境变量覆盖）
EXPOSE 8087

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8087/api/internal/health || exit 1

# 启动应用
CMD ["./code-relay"]
