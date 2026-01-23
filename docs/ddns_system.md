# 动态 DDNS 系统文档

## 功能概述

实现了一个动态 DDNS 系统，让远程服务器（keySwift.top）可以自动连接到本地动态 IP 的 MySQL 和 Redis。

## 系统架构

```
┌──────────────────────────┐         HTTP POST           ┌─────────────────────────┐
│  本地服务 (nslookup)      │  ────────────────────────>  │ 远程服务器 (keySwift.top) │
│                          │                             │                         │
│  - curl ifconfig.me      │  {"ip": "1.2.3.4"}         │  - 接收 IP              │
│  - 检测 IP 变化           │                             │  - 重连 MySQL            │
│  - 定时发送 IP            │                             │  - 重连 Redis            │
│  (每 60 秒)              │                             │                         │
└──────────────────────────┘                             └─────────────────────────┘
     本地 Windows                                               远程 Web 服务器
   (192.168.x.x 内网)                                        (http://keySwift.top)
        │                                                            │
        │                                                            │
        ├── MySQL (3306)                                            │
        └── Redis (6379)  <───────────────── 动态连接 ───────────────┘
```

## 工作流程

### 1. 本地服务 (nslookup)

**运行在**: 本地 Windows 机器（有 MySQL 和 Redis）

**功能**:
1. 使用 `curl ifconfig.me` 获取公网 IP
2. 检测 IP 是否变化
3. IP 变化时，通过 HTTP POST 发送给远程服务器
4. 定时检查（默认 60 秒）

### 2. 远程服务器 (my-codex)

**运行在**: keySwift.top

**功能**:
1. 接收本地服务发送的 IP 地址
2. 验证请求 Token
3. 重新连接 MySQL (IP:3306)
4. 重新连接 Redis (IP:6379)
5. 更新配置，后续请求使用新 IP

## 配置说明

### 统一配置文件 (config/app.yaml)

```yaml
internal:
  host: 192.168.3.176  # 默认内网 IP（会被动态更新）
  nslookup_token: nslookup-ddns-2026  # 接收 IP 更新的认证 Token

nslookup:
  check_interval: 60  # IP 检查间隔（秒）
  remote_url: http://keySwift.top/api/internal/update-ip  # 远程服务器地址
  auth_token: nslookup-ddns-2026  # 发送请求的认证 Token
```

**重要**: `internal.nslookup_token` 必须与 `nslookup.auth_token` 一致！

## 部署步骤

### 本地服务器部署 (nslookup)

1. **编译本地服务**:
```bash
cd C:\work\my-codex\nslookup
go build -o nslookup.exe
```

2. **配置文件**:
   - 使用主配置文件 `../config/app.yaml`
   - 确保 `nslookup` 部分配置正确

3. **运行服务**:
```bash
# 使用默认配置
.\nslookup.exe

# 或指定配置文件
.\nslookup.exe -config ../config/app.yaml
```

4. **设置开机自启动**:
   - 创建批处理文件 `start_nslookup.bat`:
     ```batch
     @echo off
     cd C:\work\my-codex\nslookup
     nslookup.exe -config ../config/app.yaml
     ```
   - 放入开机启动文件夹: `C:\ProgramData\Microsoft\Windows\Start Menu\Programs\StartUp`

### 远程服务器部署 (my-codex)

1. **编译主服务**:
```bash
cd C:\work\my-codex
go build -o codex-relay.exe
```

2. **上传到远程服务器**:
```bash
scp codex-relay.exe user@keySwift.top:/path/to/app/
scp config/app.yaml user@keySwift.top:/path/to/app/config/
```

3. **运行服务**:
```bash
./codex-relay.exe -config config/app.yaml
```

## API 接口

### 更新 IP 接口

**端点**: `POST /api/internal/update-ip`

**请求头**:
```
Authorization: Bearer nslookup-ddns-2026
Content-Type: application/json
```

**请求体**:
```json
{
  "ip": "1.2.3.4",
  "timestamp": 1737619200
}
```

**成功响应**:
```json
{
  "success": true,
  "message": "IP 更新成功",
  "data": {
    "ip": "1.2.3.4",
    "redis_host": "1.2.3.4",
    "mysql_host": "1.2.3.4"
  }
}
```

**错误响应**:
```json
{
  "error": "unauthorized",
  "message": "认证失败"
}
```

## 日志示例

### 本地服务 (nslookup)

```
🚀 本地 IP 监控服务启动
📡 检查间隔: 60 秒
🔗 远程服务器: http://keySwift.top/api/internal/update-ip
📍 当前公网 IP: 123.45.67.89
✅ IP 未变化，无需通知
```

**IP 变化时**:
```
📍 当前公网 IP: 123.45.67.90
🔄 IP 发生变化: 123.45.67.89 -> 123.45.67.90
✅ 已通知远程服务器
```

### 远程服务器 (my-codex)

```
📡 收到 IP 更新请求: 123.45.67.90
🔄 开始更新数据库连接...
✅ Redis 重连成功: 123.45.67.90:6379
✅ MySQL 重连成功: 123.45.67.90:3306
```

## 测试方法

### 1. 测试本地服务

```bash
# 运行本地服务，观察日志
cd C:\work\my-codex\nslookup
.\nslookup.exe
```

### 2. 手动触发 IP 更新

使用 curl 或 Postman 模拟本地服务发送请求：

```bash
curl -X POST http://keySwift.top/api/internal/update-ip \
  -H "Authorization: Bearer nslookup-ddns-2026" \
  -H "Content-Type: application/json" \
  -d '{"ip":"123.45.67.89","timestamp":1737619200}'
```

### 3. 验证连接

远程服务器收到 IP 后，应该能够连接到本地 MySQL 和 Redis：

```bash
# 在远程服务器上测试
redis-cli -h 123.45.67.89 -p 6379 -a "31897197%12312A" PING
mysql -h 123.45.67.89 -u admin -p
```

## 注意事项

1. **防火墙设置**:
   - 本地路由器需要开放 3306 (MySQL) 和 6379 (Redis) 端口
   - 配置端口转发，将外网请求转发到本地机器

2. **安全性**:
   - 修改默认的 `auth_token`，使用强密码
   - 考虑使用 HTTPS 加密传输
   - Redis 和 MySQL 使用强密码

3. **网络要求**:
   - 本地机器需要有公网 IP（或通过端口映射）
   - 本地机器需要能访问 `ifconfig.me`
   - 远程服务器需要能访问本地 IP

4. **IP 变化**:
   - 大部分家庭宽带 IP 会定期变化
   - 本地服务会自动检测并更新
   - 检查间隔可以调整（不建议太频繁）

## 故障排查

### 问题 1: 本地服务无法获取公网 IP

**错误**: `获取公网 IP 失败: curl 执行失败`

**解决**:
- 检查网络连接
- 确保可以访问 `ifconfig.me`
- 尝试其他 IP 查询服务（修改代码中的 URL）

### 问题 2: 远程服务器拒绝连接

**错误**: `服务器返回错误: 401 - unauthorized`

**解决**:
- 检查 `auth_token` 是否匹配
- 确认远程服务器配置正确

### 问题 3: MySQL/Redis 重连失败

**错误**: `Redis 重连失败` 或 `MySQL 重连失败`

**解决**:
- 检查本地防火墙设置
- 检查路由器端口转发配置
- 确认 MySQL/Redis 正在运行
- 测试从远程服务器是否能访问本地端口

## 文件结构

```
C:\work\my-codex\
├── config/
│   └── app.yaml                    # 统一配置文件
├── nslookup/                       # 本地 IP 监控服务
│   ├── main.go                     # 主程序
│   ├── go.mod
│   └── nslookup.exe                # 编译后的可执行文件
├── internal/
│   └── controller/
│       └── ip_update.go            # 远程服务器接收 IP 接口
├── codex-relay.exe                 # 远程服务器主程序
└── docs/
    └── ddns_system.md              # 本文档
```

## 完成状态

✅ 本地 IP 监控服务 (nslookup)
✅ 远程 IP 更新接口 (UpdateIP)
✅ MySQL 动态重连
✅ Redis 动态重连
✅ 统一配置文件
✅ 认证机制
✅ 编译成功

## 后续优化

1. **失败重试**: IP 更新失败时自动重试
2. **心跳机制**: 定期发送心跳，即使 IP 未变化
3. **多服务器支持**: 支持同时向多个远程服务器发送 IP
4. **Web 界面**: 提供 Web 界面查看 IP 变化历史
5. **通知功能**: IP 变化时发送邮件/短信通知
