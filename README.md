# Codex Relay (Go)

一个用于拦截并转发“newcli ↔ codex”流量的 WebSocket 中继服务器。通过共享 `token` 将两端成对配对，原样转发帧，同时打印/记录拦截到的消息，让本机作为中间服务器供其它电脑连接使用。

- 协议：WebSocket（文本/二进制帧均可），原样双向转发
- 角色：`role=newcli`（上游） 与 `role=codex`（下游）
- 配对：同一个 `token` 的两个角色自动成对
- 域名示例：`a806698083.ticp.io`

## 运行说明（最简）

- 在本机上启动中继（默认监听 `:8080`）：

```powershell
# 进入 C:\\work\\my-codex
cd C:\\work\\my-codex

# 首次建议拉取依赖并构建
go mod tidy
go build -o relay.exe ./

# 启动（默认 :8080）
./relay.exe
```

- newcli 侧连接（上游）：
  `ws://a806698083.ticp.io:8080/ws?role=newcli&token=YOUR_TOKEN`
- codex 侧连接（下游）：
  `ws://a806698083.ticp.io:8080/ws?role=codex&token=YOUR_TOKEN`

当两端使用相同的 `token` 连接到 `/ws` 后，即自动配对并双向转发。控制台会打印连接事件与每条消息的方向、类型、长度和预览内容。

提示：如果“下游电脑”只需换 token 即可使用服务，请让对方仅替换 `YOUR_TOKEN`，其余保持不变。

## 运行说明（自定义端口/日志/白名单）

你可以通过参数或配置文件进行定制：

- 参数方式：

```powershell
# 自定义监听端口
./relay.exe -addr ":8080"

# 指定配置文件
./relay.exe -config "C:\\Users\\quanliangwei\\.codex\\relay.json"
```

- 配置文件方式：程序会在启动时尝试加载：
  `C:\\Users\\quanliangwei\\.codex\\relay.json`

示例配置：

```json
{
  "listenAddr": ":8080",
  "allowedTokens": ["YOUR_TOKEN"],
  "logDir": "C:\\\Users\\\quanliangwei\\\\.codex\\\\logs",
  "pairTimeoutSeconds": 0,
  "maxMessageLogBytes": 2048
}
```

字段说明：
- `listenAddr`：监听地址，默认 `:8080`
- `allowedTokens`：非空则启用 token 白名单；空数组或缺省表示允许任意 token
- `logDir`：日志目录；为空则仅打印到控制台
- `pairTimeoutSeconds`：配对超时（0 表示关闭）
- `maxMessageLogBytes`：日志中消息预览字节数上限

## 客户端连接规范

- newcli（上游）：`ws://a806698083.ticp.io:8080/ws?role=newcli&token=YOUR_TOKEN`
- codex（下游）：`ws://a806698083.ticp.io:8080/ws?role=codex&token=YOUR_TOKEN`
- 注意：两端必须使用同一个 `token` 才能配对成功。每个 `token` 最多维持一对连接（一个 newcli + 一个 codex）。

## 健康检查

- `GET http://a806698083.ticp.io:8080/healthz` 返回 `200 ok` 表示服务正常。

## 域名与网络建议

- 确保 `a806698083.ticp.io` 指向你的主机并转发到监听端口（如 8080）。
- Windows 防火墙需允许入站 8080 端口（或你自定义的端口）。
- 内网环境可使用内网穿透或反向代理将域名流量转发到本机。

## 拦截与日志

- 服务端会打印：连接/断开事件、每条消息的方向（in/out）、类型（text/bin）、长度、预览内容（默认截断至 2KB）。
- 若设置了 `logDir`，日志也会落地到 `relay.log` 文件中。

## 目录结构

- `main.go`：程序入口、HTTP 路由（`/ws`、`/healthz`）
- `internal/hub/hub.go`：按 token 管理配对与双向转发
- `internal/config/config.go`：加载配置（默认 `C:\\Users\\<用户名>\\.codex\\relay.json`）

## 说明与限制

- 这是一个“协议无关”的 WS 转发 demo：只转发 WebSocket 帧，不理解上层业务协议。
- 对端未连接时消息会被丢弃（未做队列/重发）。
- 未内置复杂鉴权；可用 `allowedTokens` 作为简易访问控制。

---

如需我帮你本机构建并测试运行（拉取依赖与编译），告诉我即可。
