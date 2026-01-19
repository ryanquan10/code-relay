# Code Relay (Gin Internal)

提供内部接口并读取 Redis 使用量,配置通过 Viper + YAML + 环境变量覆盖。

**新增**: 集成 React 管理后台,支持货源管理、账号管理和使用量查看。

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
- `server.port`: 服务端口 (默认 8087)
- `server.servlet.context-path`: 统一前缀（默认 `/api`）
- `spring.redis.*`: Redis 地址与鉴权（默认 `a806698083.ticp.io:6379`）

环境变量覆盖示例：

```
INTERNAL_HOST=192.168.3.176
SERVER_PORT=8087
SPRING_REDIS_HOST=a806698083.ticp.io
SPRING_REDIS_PORT=6379
SPRING_REDIS_PASSWORD=
SPRING_REDIS_DB=0
```

## 接口

基础路径为 `server.servlet.context-path`，默认 `/api`。

### 内部接口
- `GET /api/internal/health`
- `GET /api/internal/usage?customerToken=xxx&date=2026-01-19`

### 管理后台 API

#### 货源管理
- `GET /api/admin/sources` - 获取货源列表
- `POST /api/admin/sources` - 创建货源
- `PUT /api/admin/sources/:id` - 更新货源
- `DELETE /api/admin/sources/:id` - 删除货源

#### 账号管理
- `GET /api/admin/accounts` - 获取账号列表
- `GET /api/admin/accounts/token/:token` - 根据Token查询
- `POST /api/admin/accounts` - 创建账号
- `POST /api/admin/accounts/batch` - 批量创建账号
- `PUT /api/admin/accounts/:id/balance` - 更新余额
- `DELETE /api/admin/accounts/:id` - 删除账号

#### 使用量查看
- `GET /api/admin/usage` - 获取使用量列表
- `POST /api/admin/usage/query` - 按 Customer Key 查询
- `GET /api/admin/usage/stats` - 获取统计数据

## 管理后台

### 访问

启动服务后访问: http://localhost:8087

默认登录密码: `admin123` (测试环境)

### 功能特性

- **货源管理**: 添加、编辑、删除发卡来源
- **账号管理**:
  - 单个添加/批量导入账号
  - 按 Token 搜索
  - 余额充值
- **使用量查看**: 统计和查询 Token 使用情况

### 批量导入格式

账号批量导入格式（每行一个）：
```
email,token,balance
email|token|balance
```

示例：
```
user1@example.com,sk-abc123,10.00
user2@example.com,sk-def456,20.00
```

## 前端开发

### 安装依赖
```bash
cd web
npm install
```

### 启动开发服务器
```bash
cd web
npm run dev
```

访问: http://localhost:3000

### 构建生产版本
```bash
cd web
npm run build
```

构建产物位于 `web/dist`。

### 完整构建（前端+后端）

**Windows:**
```bash
build.bat
go build -o code-relay.exe .
```

**Linux/Mac:**
```bash
chmod +x build.sh
./build.sh
go build -o code-relay .
```

## 项目结构

```
code-relay/
├── web/                    # React 前端
│   ├── src/
│   │   ├── api/           # API 客户端
│   │   ├── components/    # React 组件
│   │   ├── pages/         # 页面组件
│   │   └── styles/        # 样式文件
│   └── dist/              # 构建产物
├── internal/              # Go 后端
│   ├── controller/        # 控制器
│   ├── service/           # 服务层
│   ├── repository/        # 数据访问层
│   ├── entity/            # 实体定义
│   ├── server/            # HTTP 服务器
│   ├── mysql/             # MySQL 连接
│   └── redis/             # Redis 连接
├── config/                # 配置文件
└── main.go                # 入口文件
```

## 安全说明

⚠️ **重要**: 当前版本使用硬编码密码仅用于测试。生产环境请实现真实的认证系统。

