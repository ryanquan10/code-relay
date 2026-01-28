# Codex Relay - 账号管理与中继系统

## 项目概述

这是一个完整的账号管理和 API 中继系统，集成了动态 DDNS、认证系统、使用量统计等功能。

## 主要功能

### 1. 账号管理系统
- ✅ 账号来源管理（货源管理）
- ✅ 产品管理
- ✅ 账号管理（添加、编辑、删除、批量导入）
- ✅ 额度管理（总额度、已用额度、剩余额度）
- ✅ 余额检查（请求前验证余额，不足返回 402）

### 2. 认证系统
- ✅ 登录/登出功能
- ✅ Token 认证（基于 Redis 存储，24 小时有效）
- ✅ 路由保护（所有管理接口需要认证）
- ✅ 自动跳转（Token 过期自动跳转登录页）

### 3. 使用量统计
- ✅ 实时流量统计
- ✅ Token 计数（精确文本分析）
- ✅ 按天分组统计
- ✅ Redis Stream 异步处理
- ✅ 自动更新 used_balance

### 4. 动态 DDNS 系统
- ✅ 本地 IP 监控服务（nslookup）
- ✅ 自动检测公网 IP 变化
- ✅ 动态更新远程服务器连接
- ✅ MySQL/Redis 自动重连

### 5. API 中继
- ✅ Codex API 代理
- ✅ Claude API 代理
- ✅ 请求转发和响应处理
- ✅ 流量计数和计费

## 快速开始

### 1. 配置文件

编辑 `config/app.yaml`:

```yaml
internal:
  host: 192.168.3.176  # 内网 IP（会被 DDNS 动态更新）
  nslookup_token: nslookup-ddns-2026  # DDNS 认证 Token

server:
  port: 8087  # HTTP 服务端口

spring:
  datasource:
    url: "jdbc:mysql://a806698083.ticp.io:3306/codex_rental?..."
    username: admin
    password: your-password

  redis:
    host: a806698083.ticp.io
    port: 6379
    password: your-redis-password
    db: 0

admin:
  password: admin1237788  # 管理员密码

nslookup:
  check_interval: 10  # IP 检查间隔（秒）
  remote_url: http://keySwift.top/api/internal/update-ip
  auth_token: nslookup-ddns-2026  # 必须与 internal.nslookup_token 一致
```

### 2. 数据库迁移

执行 SQL 脚本：

```bash
mysql -u admin -p codex_rental < migrations/add_used_balance.sql
mysql -u admin -p codex_rental < migrations/add_tokens_to_usage.sql
```

### 3. 编译项目

#### 编译后端
```bash
go build -o codex-relay.exe
```

#### 编译前端
```bash
cd frontend
npm run build
cd ..
```

#### 编译本地 IP 监控服务
```bash
cd nslookup
go build -o nslookup.exe
cd ..
```

### 4. 运行服务

#### 远程服务器（keySwift.top）
```bash
./codex-relay.exe
```

#### 本地服务器（Windows）
```bash
cd nslookup
start.bat
```

### 5. 访问系统

- **管理后台**: http://localhost:8087/ 或 http://keySwift.top/
- **登录密码**: `admin1237788`（可在配置文件修改）

## API 端点

### 公开接口
- `POST /api/auth/login` - 登录
- `POST /api/auth/logout` - 登出
- `GET /api/usage/daily` - 按天查询使用量
- `GET /api/internal/health` - 健康检查
- `POST /api/internal/update-ip` - 更新 IP（DDNS）

### 管理接口（需要认证）
- `GET /api/admin/sources` - 获取货源列表
- `POST /api/admin/sources` - 创建货源
- `GET /api/admin/products` - 获取产品列表
- `POST /api/admin/products` - 创建产品
- `GET /api/admin/accounts` - 获取账号列表
- `POST /api/admin/accounts` - 创建账号
- `POST /api/admin/accounts/batch` - 批量创建账号
- `PUT /api/admin/accounts/:id` - 更新账号（含 use_status、余额等）
- `PUT /api/admin/accounts/:id/balance` - 更新账号余额
- `DELETE /api/admin/accounts/:id` - 删除账号
- `GET /api/admin/usage` - 查询使用量

### 中继接口
- `/codex/v1/*` - Codex API 代理
- `/claude/v1/*` - Claude API 代理

## 余额系统说明

### 余额模型

```
总额度 (balance) = 100 元
已用额度 (used_balance) = 30 元
剩余额度 = balance - used_balance = 70 元
```

### 余额检查规则

- 请求前检查: `used_balance > balance` → 返回 402 Payment Required
- 请求完成后: 自动增加 `used_balance`
- 充值: 增加 `balance`
- 修正: 管理员可以直接修改 `used_balance`

### 计费规则

- Token 转换率: 每 1000 tokens = 0.01 元
- 即: 1 token = 0.00001 元

## 使用量统计流程

```
1. 请求到达 → HandleRequest
2. 余额检查 → BalanceCheckService
3. 代理请求 → ReverseProxy
4. 流量计数 → TrafficCounter（统计字节数）
5. 精确计算 → EstimateTokensFromText（分析文本）
6. 发送到 Redis Stream → SendTokenUsageToStream
7. 后台消费者 → TokenUsageConsumer
8. 写入 MySQL → UsageService.RecordTokenUsage
9. 更新余额 → AccountRepository.IncrementUsedBalance
```

## 动态 DDNS 工作流程

```
1. 本地服务 (nslookup)
   └─> curl ifconfig.me（获取公网 IP）
   └─> 检测 IP 变化
   └─> POST /api/internal/update-ip

2. 远程服务器 (my-codex)
   └─> 验证 Token
   └─> 关闭旧连接
   └─> 重连 Redis (新IP:6379)
   └─> 重连 MySQL (新IP:3306)
   └─> 更新配置
```

## 项目结构

```
C:\work\my-codex\
├── client/                          # API 客户端
│   ├── codex.go                     # Codex API 中继
│   ├── claude.go                    # Claude API 中继
│   └── registry.go                  # 客户端注册
├── config/                          # 配置文件
│   ├── config.go                    # 配置结构
│   └── app.yaml                     # 主配置文件
├── internal/
│   ├── controller/                  # 控制器
│   │   ├── auth.go                  # 登录/登出
│   │   ├── ip_update.go             # IP 更新接口
│   │   ├── admin_account.go         # 账号管理
│   │   ├── admin_product.go         # 产品管理
│   │   └── admin_usage.go           # 使用量查看
│   ├── middleware/                  # 中间件
│   │   └── auth.go                  # 认证中间件
│   ├── service/                     # 业务逻辑层
│   │   ├── balance_check_service.go # 余额检查
│   │   ├── traffic_counter.go       # 流量统计
│   │   └── usage_service.go         # 使用量服务
│   └── server/                      # HTTP 服务器
│       └── server.go                # 路由配置
├── nslookup/                        # 本地 IP 监控服务
│   ├── main.go                      # 主程序
│   ├── start.bat                    # 启动脚本
│   └── nslookup.exe                 # 可执行文件
├── frontend/                        # 前端项目（React + TypeScript）
├── docs/                            # 文档
│   ├── authentication_system.md     # 认证系统文档
│   ├── account_edit_feature.md      # 账号编辑功能文档
│   └── ddns_system.md               # 动态 DDNS 文档
├── migrations/                      # 数据库迁移
├── main.go                          # 主程序入口
├── codex-relay.exe                  # 编译后的主服务
└── README.md                        # 本文件
```

## 技术栈

### 后端
- **语言**: Go 1.24+
- **框架**: Gin
- **数据库**: MySQL 8.0+
- **缓存**: Redis 6.0+
- **ORM**: GORM
- **配置**: Viper (YAML)

### 前端
- **语言**: TypeScript
- **框架**: React 18
- **构建**: Vite
- **样式**: Tailwind CSS
- **HTTP**: Axios

## 文档

- [认证系统文档](docs/authentication_system.md) - 详细的认证系统实现说明
- [账号编辑功能](docs/account_edit_feature.md) - 账号编辑功能说明
- [动态 DDNS 系统](docs/ddns_system.md) - DDNS 系统完整文档

## 安全建议

1. **修改默认密码**:
   - 管理员密码: `admin.password`
   - MySQL 密码: `spring.datasource.password`
   - Redis 密码: `spring.redis.password`
   - DDNS Token: `internal.nslookup_token`

2. **使用 HTTPS**:
   - 生产环境使用 SSL 证书
   - 配置 Nginx 反向代理

3. **限制访问**:
   - 使用防火墙限制管理接口访问
   - 配置 IP 白名单

## 故障排查

### 1. 登录失败
- 检查密码是否正确（`config/app.yaml` 中的 `admin.password`）
- 检查 Redis 是否正常运行
- 查看浏览器控制台错误信息

### 2. 余额不更新
- 检查 TokenUsageConsumer 是否启动（查看日志）
- 检查 Redis Stream 是否有消息堆积
- 验证 MySQL 连接是否正常

### 3. DDNS 不工作
- 检查本地服务是否运行
- 验证防火墙和端口转发配置
- 测试远程服务器是否能访问本地 IP
- 检查 Token 是否匹配

## 许可证

本项目仅供学习和研究使用。

---

**最后更新**: 2026-01-23
**版本**: 1.0.0
