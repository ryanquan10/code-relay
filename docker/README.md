# Code Relay - Docker 一键部署

## 🚀 快速开始

### 方式一：一键启动（推荐）

#### Windows 系统
```bash
# 在 docker 目录下双击运行
start.bat

# 或者在命令行中执行
cd docker
start.bat
```

#### Linux/Mac 系统
```bash
cd docker
chmod +x start.sh
./start.sh
```

### 方式二：手动启动

```bash
cd docker

# 构建并启动
docker compose up -d

# 或使用旧版命令
docker-compose up -d
```

## ⚙️ 配置说明

所有配置都在 `.env.example` 文件中，启动脚本会自动复制为 `.env`。

默认配置：
- **服务端口**: 8087
- **管理员密码**: admin123456
- **MySQL**: a806698083.ticp.io:3306
- **Redis**: a806698083.ticp.io:6379

如需修改，编辑 `docker/.env` 文件即可。

## 📡 服务访问

服务启动后，可通过以下地址访问：

- **管理后台**: http://localhost:8087
- **API 端点**: http://localhost:8087/api
- **健康检查**: http://localhost:8087/api/internal/health
- **Codex 代理**: http://localhost:8087/codex/v1

## 🔧 常用命令

```bash
cd docker

# 查看服务状态
docker compose ps

# 查看日志
docker compose logs -f

# 重启服务
docker compose restart

# 停止服务
docker compose down

# 重新构建
docker compose up -d --build
```

## 🌐 部署到其他机器

### 方法一：直接复制 docker 目录

1. 将整个 `code-relay` 项目复制到目标机器
2. 进入 `docker` 目录
3. 运行 `start.bat`（Windows）或 `start.sh`（Linux）

### 方法二：使用 Docker 镜像

```bash
# 1. 在源机器构建镜像
cd docker
docker compose build
docker save -o code-relay.tar code-relay:latest

# 2. 传输 code-relay.tar 到目标机器

# 3. 在目标机器导入镜像
docker load -i code-relay.tar

# 4. 复制 docker 目录并启动
cd docker
./start.sh
```

## 🐛 故障排查

### 查看日志
```bash
docker compose logs code-relay
```

### 进入容器调试
```bash
docker compose exec code-relay sh
```

### 健康检查
```bash
docker compose ps
curl http://localhost:8087/api/internal/health
```

---

**部署完成后，访问 http://localhost:8087 即可使用！** 🎉
