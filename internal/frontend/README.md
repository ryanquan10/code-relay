# 账户管理后台

基于 React + TypeScript + Vite 构建的账户管理系统，支持货源管理、账号管理和使用量查看。

## 功能特性

- **货源管理**: 管理发卡来源，支持添加、编辑、删除
- **账号管理**: 管理账户信息，支持批量导入账号和token
- **使用量查看**: 查看token使用量和消费统计
- **简单认证**: 使用固定密码登录(测试环境)

## 技术栈

- React 18
- TypeScript
- Vite
- TailwindCSS
- React Router
- Axios

## 开发指南

### 安装依赖

```bash
npm install
```

### 启动开发服务器

```bash
npm run dev
```

访问 http://localhost:3000

### 构建生产版本

```bash
npm run build
```

构建产物将输出到 `dist` 目录。

## 默认登录密码

```
admin123
```

## API 配置

API 请求会代理到 `http://localhost:8082`，可以在 `vite.config.ts` 中修改。

## 批量导入格式

账号批量导入支持以下格式(每行一个):

```
email,token,balance
email|token|balance
```

示例:
```
user1@example.com,sk-token1,10.00
user2@example.com,sk-token2,20.00
```

余额字段可选，默认为 0。
