# Code Relay (Gin Internal)

提供内部接口并读取 Redis 使用量，配置通过 Viper + YAML + 环境变量覆盖。

## 运行

```powershell
go run .
```

自定义配置文件路径：

```powershell
go run . -config D:\path\app.yaml
```

或使用环境变量：

```
CONFIG_PATH=D:\path\app.yaml
```

## 配置

默认读取 `config/app.yaml`，示例内容见 `config/app.yaml`。

常用配置项：
- `internal.host`: 内网地址，占位符 `INTERNAL_HOST` 可覆盖
- `server.port`: 服务端口
- `server.servlet.context-path`: 统一前缀（默认 `/api`）
- `spring.redis.*`: Redis 地址与鉴权（默认 `a806698083.ticp.io:6379`）

环境变量覆盖示例：

```
INTERNAL_HOST=192.168.3.176
SERVER_PORT=8082
SPRING_REDIS_HOST=a806698083.ticp.io
SPRING_REDIS_PORT=6379
SPRING_REDIS_PASSWORD=
SPRING_REDIS_DB=0
```

## 接口

基础路径为 `server.servlet.context-path`，默认 `/api`。

- `GET /api/internal/health`
- `GET /api/internal/usage?customerToken=xxx&date=2026-01-19`
