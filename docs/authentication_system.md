# 认证系统实现文档

## 功能概述

实现了基于 Token 的简单认证系统，保护管理后台 API 不被未授权访问。

## 实现方案

### 认证流程

```
用户登录 → 验证密码 → 生成 Token → 存储到 Redis (24小时) → 返回 Token 给前端
                ↓
          前端保存 Token 到 localStorage
                ↓
    后续请求携带 Token 在 Authorization 请求头
                ↓
     中间件验证 Token → 从 Redis 检查是否有效
                ↓
            允许/拒绝访问
```

### 1. 后端实现

#### 1.1 登录接口 (internal/controller/auth.go:40-95)

**接口**: `POST /api/auth/login`

**请求体**:
```json
{
  "password": "admin1237788"
}
```

**成功响应**:
```json
{
  "success": true,
  "message": "登录成功",
  "token": "生成的64位随机token",
  "user": {
    "role": "admin",
    "username": "admin"
  }
}
```

**实现细节**:
- 从配置文件读取管理员密码 (`config.Admin.Password`)
- 生成 64 位随机 hex token
- 将 token 存储到 Redis，key 为 `admin_token:{token}`，过期时间 24 小时
- 返回 token 给客户端

#### 1.2 登出接口 (internal/controller/auth.go:97-125)

**接口**: `POST /api/auth/logout`

**请求头**:
```
Authorization: Bearer {token}
```

**响应**:
```json
{
  "success": true,
  "message": "登出成功"
}
```

**实现细节**:
- 从 Redis 删除 token

#### 1.3 认证中间件 (internal/middleware/auth.go)

**功能**: 保护需要认证的路由

**实现逻辑**:
```go
func AuthMiddleware() gin.HandlerFunc {
    // 1. 从请求头获取 Authorization
    // 2. 支持 "Bearer token" 格式
    // 3. 从 Redis 验证 token 是否存在
    // 4. Token 有效 -> 继续处理
    // 5. Token 无效 -> 返回 401 Unauthorized
}
```

**错误响应**:
```json
{
  "error": "unauthorized",
  "message": "未登录或登录已过期，请先登录"
}
```

#### 1.4 路由保护 (internal/server/server.go:92-120)

```go
// 管理后台路由（需要认证）
admin := api.Group("/admin")
admin.Use(middleware.AuthMiddleware()) // 添加认证中间件
{
    // 所有 /api/admin/* 路由都需要认证
    admin.GET("/sources", ...)
    admin.POST("/accounts", ...)
    // ...
}
```

**受保护的路由**:
- `/api/admin/sources` - 货源管理
- `/api/admin/products` - 产品管理
- `/api/admin/accounts` - 账号管理
- `/api/admin/usage` - 使用量查看

**不需要认证的路由**:
- `/api/auth/login` - 登录
- `/api/auth/logout` - 登出
- `/api/usage/daily` - 公开使用量查询
- `/api/internal/health` - 健康检查

### 2. 前端实现

#### 2.1 登录页面 (frontend/src/pages/Login.tsx:22-35)

**功能**:
1. 用户输入密码
2. 调用 `/api/auth/login`
3. 保存 token 到 `localStorage.admin_token`
4. 保存用户信息到 `localStorage.user`
5. 跳转到管理后台

#### 2.2 请求拦截器 (frontend/src/api/client.ts:11-24)

**自动添加 Token**:
```typescript
api.interceptors.request.use((config) => {
  const token = localStorage.getItem('admin_token');
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});
```

所有通过 `api` 发起的请求都会自动携带 token。

#### 2.3 响应拦截器 (frontend/src/api/client.ts:26-40)

**自动处理 401 错误**:
```typescript
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      // 清除 token 和用户信息
      localStorage.removeItem('admin_token');
      localStorage.removeItem('user');
      // 跳转到登录页
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

#### 2.4 路由保护 (frontend/src/pages/AdminDashboard.tsx:27-44)

**检查登录状态**:
```typescript
useEffect(() => {
  const userStr = localStorage.getItem('user');
  const token = localStorage.getItem('admin_token');

  if (!userStr || !token) {
    navigate('/login');  // 未登录，跳转到登录页
    return;
  }

  const user = JSON.parse(userStr);
  if (user.role !== 'admin') {
    alert('无权限访问');
    navigate('/login');
    return;
  }

  setLoading(false);
}, [navigate]);
```

#### 2.5 登出功能 (frontend/src/components/Header.tsx:11-17)

**登出按钮**:
```typescript
const handleLogout = () => {
  if (confirm('确定要退出登录吗？')) {
    localStorage.removeItem('admin_token');
    localStorage.removeItem('user');
    navigate('/login');
  }
};
```

## 使用方法

### 首次登录

1. 访问 `http://localhost:8087/login`
2. 输入管理员密码（默认: `admin1237788`，在 `config/app.yaml` 中配置）
3. 点击登录

### 修改管理员密码

编辑 `config/app.yaml`:
```yaml
admin:
  password: "your_new_password"
```

或通过环境变量:
```bash
export ADMIN_PASSWORD="your_new_password"
./codex-relay.exe
```

### API 调用示例

**使用 curl**:
```bash
# 1. 登录获取 token
curl -X POST http://localhost:8087/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"password":"admin1237788"}'

# 响应: {"success":true,"token":"abc123...","user":{...}}

# 2. 使用 token 访问受保护接口
curl http://localhost:8087/api/admin/accounts \
  -H "Authorization: Bearer abc123..."

# 3. 登出
curl -X POST http://localhost:8087/api/auth/logout \
  -H "Authorization: Bearer abc123..."
```

**使用 Postman/ApiFox**:

1. 登录请求:
   - Method: `POST`
   - URL: `http://localhost:8087/api/auth/login`
   - Body (JSON):
     ```json
     {"password": "admin1237788"}
     ```

2. 复制响应中的 `token`

3. 访问受保护接口:
   - Method: `GET`
   - URL: `http://localhost:8087/api/admin/accounts`
   - Headers:
     ```
     Authorization: Bearer {复制的token}
     ```

## Token 管理

### Token 存储

- **存储位置**: Redis
- **Key 格式**: `admin_token:{token}`
- **Value**: `"admin"`
- **过期时间**: 24 小时

### Token 自动续期

目前不支持自动续期。Token 过期后需要重新登录。

如需实现自动续期，可以：
1. 在中间件验证 token 时重置过期时间
2. 前端定时刷新 token

### Token 安全

1. **传输安全**: 建议生产环境使用 HTTPS
2. **存储安全**: Token 存储在 `localStorage`，注意防止 XSS 攻击
3. **长度**: 64 位 hex 字符串，安全性足够

## 测试清单

- [ ] 未登录访问管理后台自动跳转到登录页
- [ ] 输入错误密码显示错误提示
- [ ] 输入正确密码登录成功，跳转到管理后台
- [ ] 登录后可以正常访问所有管理功能
- [ ] 点击"退出登录"按钮退出成功
- [ ] Token 过期后（24小时）自动跳转到登录页
- [ ] 使用过期/无效 token 访问 API 返回 401

## 完成状态

✅ 后端登录接口
✅ 后端登出接口
✅ 认证中间件
✅ 路由保护
✅ Token 生成和验证
✅ 前端登录页面
✅ 前端登出功能
✅ 请求/响应拦截器
✅ 路由保护
✅ 后端编译成功
✅ 前端编译成功

## 后续优化建议

1. **Token 刷新**: 实现 access token + refresh token 机制
2. **记住登录**: 支持 "记住我" 功能，延长 token 有效期
3. **多用户**: 支持多个管理员账号
4. **权限管理**: 支持不同角色和权限
5. **登录日志**: 记录登录历史和操作日志
6. **验证码**: 添加验证码防止暴力破解
7. **2FA**: 支持双因素认证
