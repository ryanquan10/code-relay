# Codex Relay - 部署检查清单

## ✅ 系统状态

### 编译状态
- ✅ 主服务器: `codex-relay.exe` (19 MB) - 已编译
- ✅ 本地监控: `nslookup/nslookup.exe` (9.6 MB) - 已编译

### 配置文件
- ✅ `config/app.yaml` - 统一配置文件（包含所有必要配置）
  - ✅ `internal.host` - 内网 IP 配置
  - ✅ `internal.nslookup_token` - DDNS 认证 Token
  - ✅ `server.port` - HTTP 服务端口 (8087)
  - ✅ `spring.datasource` - MySQL 配置
  - ✅ `spring.redis` - Redis 配置
  - ✅ `admin.password` - 管理员密码
  - ✅ `nslookup` - 动态 DDNS 配置

### 核心功能
- ✅ 账号管理系统
- ✅ 认证系统 (Token-based)
- ✅ 使用量统计 (Redis Stream + MySQL)
- ✅ 动态 DDNS 系统
- ✅ API 中继 (Codex + Claude)

---

## 🚀 部署步骤

### 1. 远程服务器部署 (keySwift.top)

#### 上传文件
```bash
# 上传主服务
scp codex-relay.exe user@keySwift.top:/opt/codex-relay/

# 上传配置文件
scp config/app.yaml user@keySwift.top:/opt/codex-relay/config/

# 上传前端（如果需要）
scp -r frontend/dist user@keySwift.top:/opt/codex-relay/frontend/
```

#### 启动服务
```bash
# SSH 登录到远程服务器
ssh user@keySwift.top

# 进入目录
cd /opt/codex-relay

# 启动服务（前台）
./codex-relay.exe

# 或者使用 systemd 后台运行（推荐）
sudo systemctl start codex-relay
```

#### 验证服务
```bash
# 检查健康状态
curl http://keySwift.top/api/internal/health

# 预期响应:
# {"status":"ok","service":"account-rental-backend"}
```

---

### 2. 本地服务器部署 (Windows)

#### 配置确认
编辑 `config/app.yaml`，确认以下配置：
```yaml
nslookup:
  check_interval: 60  # IP 检查间隔（秒）
  remote_url: http://keySwift.top/api/internal/update-ip
  auth_token: nslookup-ddns-2026  # 必须与 internal.nslookup_token 一致
```

#### 启动服务
```batch
cd C:\work\my-codex\nslookup
start.bat
```

#### 验证日志
应该看到类似输出：
```
======================================
  本地 IP 监控服务 (NSLookup DDNS)
======================================

🚀 本地 IP 监控服务启动
📡 检查间隔: 60 秒
🔗 远程服务器: http://keySwift.top/api/internal/update-ip
📍 当前公网 IP: 123.45.67.89
✅ IP 未变化，无需通知
```

#### 设置开机自启动
1. 创建启动脚本 `C:\work\my-codex\nslookup\startup.bat`:
```batch
@echo off
cd C:\work\my-codex\nslookup
start /B nslookup.exe -config ../config/app.yaml
```

2. 将快捷方式添加到启动文件夹:
```
Win + R → shell:startup
```

---

## 🔍 测试验证

### 1. 测试动态 DDNS

#### 手动触发 IP 更新
```bash
curl -X POST http://keySwift.top/api/internal/update-ip \
  -H "Authorization: Bearer nslookup-ddns-2026" \
  -H "Content-Type: application/json" \
  -d '{"ip":"123.45.67.89","timestamp":1737619200}'
```

#### 预期响应
```json
{
  "success": true,
  "message": "IP 更新成功",
  "data": {
    "ip": "123.45.67.89",
    "redis_host": "123.45.67.89",
    "mysql_host": "123.45.67.89"
  }
}
```

### 2. 测试管理后台

访问: `http://keySwift.top/` 或 `http://localhost:8087/`

**登录信息**:
- 密码: `admin1237788`

### 3. 测试 API 中继

```bash
# 测试 Codex API
curl http://keySwift.top/codex/v1/chat/completions \
  -H "Authorization: Bearer your-account-token" \
  -H "Content-Type: application/json" \
  -d '{"messages":[{"role":"user","content":"Hello"}]}'
```

---

## 🛡️ 安全检查清单

### 必须修改的默认密码

- [ ] `admin.password` - 管理员密码
- [ ] `spring.datasource.password` - MySQL 密码
- [ ] `spring.redis.password` - Redis 密码
- [ ] `internal.nslookup_token` 和 `nslookup.auth_token` - DDNS Token

### 网络安全

- [ ] 配置防火墙规则
- [ ] 启用 HTTPS (使用 Nginx 反向代理)
- [ ] 限制管理接口访问 IP
- [ ] 本地路由器端口转发配置:
  - MySQL: 3306
  - Redis: 6379

---

## 📊 监控和维护

### 日志位置
- 主服务器: 标准输出 (建议重定向到文件)
- 本地监控: 控制台输出

### 关键指标
- Redis 连接状态
- MySQL 连接状态
- API 请求成功率
- IP 更新频率
- 账号余额使用情况

### 常见问题

#### 问题 1: 登录失败
- 检查 Redis 是否运行
- 验证管理员密码配置
- 查看浏览器控制台错误

#### 问题 2: DDNS 不工作
- 检查本地服务是否运行
- 验证 Token 是否匹配
- 测试网络连通性:
  ```bash
  curl ifconfig.me  # 本地测试
  telnet <本地IP> 3306  # 远程测试 MySQL
  telnet <本地IP> 6379  # 远程测试 Redis
  ```

#### 问题 3: 余额不更新
- 检查 TokenUsageConsumer 是否启动
- 查看 Redis Stream 消息堆积
- 验证 MySQL 连接正常

---

## 📝 系统架构

```
┌─────────────────────────────────────────────────────────────┐
│                    远程服务器 (keySwift.top)                 │
│                                                             │
│  ┌──────────────┐      ┌──────────────┐                    │
│  │ Codex Relay  │─────→│    MySQL     │←─────┐            │
│  │   (Gin)      │      │ (用户数据)    │       │            │
│  └──────────────┘      └──────────────┘       │            │
│         │                                      │            │
│         │              ┌──────────────┐       │            │
│         └─────────────→│    Redis     │       │            │
│                        │ (Token/Stats) │       │            │
│                        └──────────────┘       │            │
│                               ▲                │            │
└───────────────────────────────┼────────────────┼────────────┘
                                │ 动态           │
                                │ 连接           │
                                │                │
┌───────────────────────────────┼────────────────┼────────────┐
│              本地服务器 (Windows)               │            │
│                               │                │            │
│  ┌──────────────┐      ┌──────────────┐ ┌──────────────┐  │
│  │  nslookup    │─────→│    MySQL     │ │    Redis     │  │
│  │  (监控IP)     │      │   :3306      │ │   :6379      │  │
│  └──────────────┘      └──────────────┘ └──────────────┘  │
│         │                                                   │
│         │ POST /api/internal/update-ip                     │
│         └─────────────────────────────────────────────────→│
│                                                             │
│  📡 每 60 秒检查公网 IP，变化时通知远程服务器                 │
└─────────────────────────────────────────────────────────────┘
```

---

## ✅ 部署完成确认

- [ ] 远程服务器运行正常 (http://keySwift.top/api/internal/health 返回 200)
- [ ] 本地 IP 监控服务运行
- [ ] 管理后台可访问并登录
- [ ] DDNS 功能测试通过
- [ ] API 中继功能正常
- [ ] 所有默认密码已修改
- [ ] 防火墙和端口转发已配置
- [ ] 日志输出正常

---

**部署日期**: 2026-01-23
**版本**: 1.0.0
**文档更新**: 2026-01-23
