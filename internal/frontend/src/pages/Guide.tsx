import { useEffect } from "react";
import { useLocation } from "react-router-dom";

﻿
export default function Guide() {
  const location = useLocation();
  useEffect(() => {
    const id = location.hash?.slice(1);
    if (id) {
      const el = document.getElementById(id);
      if (el) {
        setTimeout(() => el.scrollIntoView({ behavior: "smooth", block: "start" }), 0);
      }
    }
  }, [location.hash]);

  return (
    <div className="min-h-screen bg-white">
      <div className="max-w-5xl mx-auto px-6 py-10">
        <h1 className="text-3xl font-bold mb-2">教程 · ClaudeCode · CodeX · Gemini 三合一</h1>
        <p className="text-gray-600 mb-8">迄今为止最先进的代码助手整合教程。本页为公开访问的静态指南（/guide），无需登录。</p>

        <section className="mt-6">
          <h2 id="overview" className="text-2xl font-semibold mb-3">概览</h2>
          <ul className="list-disc pl-6 space-y-2 text-gray-800">
            <li>Claude Code 是为编写代码而生的智能 Agent，可用自然语言高效实现想法。</li>
            <li>只需一杯咖啡的时间，Claude Code 就能帮你完成从理解到修改到提交的工作流。</li>
            <li>CodeX 已上线，口碑飙升，与 Claude Code（cc）额度通用。</li>
          </ul>
        </section>

        <section className="mt-8">
          <h2 className="text-xl font-semibold mb-3">支持的 IDE</h2>
          <ul className="list-disc pl-6 space-y-1 text-gray-800">
            <li>Visual Studio Code（包括 Cursor、Windsurf 等分支）</li>
            <li>JetBrains IDEs（PyCharm、WebStorm、IntelliJ、GoLand）</li>
          </ul>
        </section>

        <section className="mt-10">
          <h2 id="claude" className="text-2xl font-semibold mb-3">一、ClaudeCode（稳定性/性价比/口碑之王）</h2>
          <ul className="list-disc pl-6 space-y-2 text-gray-800">
            <li>纯正 Max 号池：拒绝第三方掺假，调用质量稳定。</li>
            <li>无需魔法：国内外直连，响应快、稳定不封号。</li>
            <li>（独家）缓存透明计费：记录清晰可查，降低不必要扣费。</li>
            <li>（独家）高命中缓存技术：减少 token 调用，令额度更耐用。</li>
            <li>（独家）兼容官网常见报错，错误率更低。</li>
            <li>（独家）支持 RooCode/Kilo Code/CherryStudio/ChatBox（详见下文 1.3）。</li>
            <li>支持镜像包一键安装登录；或使用官方包 + 环境变量配置。</li>
          </ul>
          <div className="mt-3 text-gray-700">
            <p>系统要求：</p>
            <ul className="list-disc pl-6">
              <li>操作系统：macOS 10.15+ / Ubuntu 20.04+ / Debian 10+ / Windows</li>
              <li>硬件：≥ 4GB RAM</li>
              <li>软件：Node.js 18+</li>
            </ul>
          </div>

          <div className="mt-6">
            <h3 className="text-xl font-semibold mb-2">1.1 官方包安装（推荐，二选一）</h3>
            <p className="text-gray-700">安装官方 Claude Code：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`npm install -g @anthropic-ai/claude-code
claude --version`}</code></pre>

            <h4 className="text-lg font-semibold mt-4 mb-2">环境变量配置（Windows / macOS / Linux）</h4>
            <p className="font-medium">Windows</p>
            <p className="text-gray-700 mt-1">方法1（永久）：编辑 <code>C:\\Users\\{`{用户名}` }\\.claude\\settings.json</code></p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "替换为您的API Key",
    "ANTHROPIC_BASE_URL": "https://keyswift.top/claude",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}`}</code></pre>
            <p className="text-gray-700 mt-2">注意：可选特价渠道 <code>https://keyswift.top/claude/aws</code></p>
            <p className="text-gray-700 mt-2">方法2（临时，仅当前终端）：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# PowerShell
$env:ANTHROPIC_BASE_URL="https://keyswift.top/claude"
$env:ANTHROPIC_AUTH_TOKEN="替换为您的API Key"

# CMD
set ANTHROPIC_BASE_URL=https://keyswift.top/claude
set ANTHROPIC_AUTH_TOKEN=替换为您的API Key`}</code></pre>
            <p className="text-gray-700 mt-2">方法3（永久，图形界面或 PowerShell）：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# PowerShell（用户级）
[System.Environment]::SetEnvironmentVariable('ANTHROPIC_BASE_URL', 'https://keyswift.top/claude', 'User')
[System.Environment]::SetEnvironmentVariable('ANTHROPIC_AUTH_TOKEN', '替换为您的API Key', 'User')`}</code></pre>

            <p className="font-medium mt-4">macOS</p>
            <p className="text-gray-700 mt-1">方法1（推荐）：编辑 <code>~/.claude/settings.json</code></p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "替换为您的API Key",
    "ANTHROPIC_BASE_URL": "https://keyswift.top/claude",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}`}</code></pre>
            <p className="text-gray-700 mt-2">可选特价渠道：<code>https://keyswift.top/claude/aws</code></p>
            <p className="text-gray-700 mt-2">方法2（临时）：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`export ANTHROPIC_BASE_URL="https://keyswift.top/claude"
export ANTHROPIC_AUTH_TOKEN="替换为您的API Key"`}</code></pre>
            <p className="text-gray-700 mt-2">方法3（永久）：编辑 shell 配置并 source：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# bash（默认）
echo 'export ANTHROPIC_BASE_URL="https://keyswift.top/claude"' >> ~/.bash_profile
echo 'export ANTHROPIC_AUTH_TOKEN="替换为您的API Key"' >> ~/.bash_profile
source ~/.bash_profile

# zsh
echo 'export ANTHROPIC_BASE_URL="https://keyswift.top/claude"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="替换为您的API Key"' >> ~/.zshrc
source ~/.zshrc`}</code></pre>

            <p className="font-medium mt-4">Linux</p>
            <p className="text-gray-700 mt-1">方法1（临时）：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`export ANTHROPIC_BASE_URL="https://keyswift.top/claude"
export ANTHROPIC_AUTH_TOKEN="替换为您的API Key"`}</code></pre>
            <p className="text-gray-700 mt-2">方法2（永久）：编辑 shell 配置并 source：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# bash
echo 'export ANTHROPIC_BASE_URL="https://keyswift.top/claude"' >> ~/.bashrc
echo 'export ANTHROPIC_AUTH_TOKEN="替换为您的API Key"' >> ~/.bashrc
source ~/.bashrc

# zsh
echo 'export ANTHROPIC_BASE_URL="https://keyswift.top/claude"' >> ~/.zshrc
echo 'export ANTHROPIC_AUTH_TOKEN="替换为您的API Key"' >> ~/.zshrc
source ~/.zshrc`}</code></pre>
            <p className="text-gray-700 mt-2">方法3：配置 <code>~/.claude/settings.json</code>：</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`{
  "env": {
    "ANTHROPIC_AUTH_TOKEN": "替换为您的API Key",
    "ANTHROPIC_BASE_URL": "https://keyswift.top/claude",
    "CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC": 1
  },
  "permissions": {
    "allow": [],
    "deny": []
  }
}`}</code></pre>

            <h4 className="text-lg font-semibold mt-4 mb-2">通用验证</h4>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# macOS/Linux
echo $ANTHROPIC_BASE_URL
echo $ANTHROPIC_AUTH_TOKEN

# Windows PowerShell
echo $env:ANTHROPIC_BASE_URL
echo $env:ANTHROPIC_AUTH_TOKEN

# Windows CMD
echo %ANTHROPIC_BASE_URL%
echo %ANTHROPIC_AUTH_TOKEN%`}</code></pre>

            <h4 className="text-lg font-semibold mt-4 mb-2">新增 API 渠道</h4>
            <ul className="list-disc pl-6 text-gray-800">
              <li><code>https://keyswift.top/claude/aws</code>（不限调用方式）</li>
              <li><code>https://keyswift.top/claude/droid</code>（仅支持 CC，可能不稳定）</li>
            </ul>
          </div>

          <div className="mt-8">
            <h3 className="text-xl font-semibold mb-2">1.2 镜像包安装（二选一，不太推荐）</h3>
            <p className="font-medium">macOS</p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# 1）全局安装 CLI
yarn -v || npm -v
npm install -g https://keyswift.top/install --registry=https://registry.npmmirror.com

# 2）进入你的项目后运行
cd your-project-folder
claude

# 3）弹出登录后即可使用`}</code></pre>
            <p className="font-medium mt-3">Windows（无需 WSL，1.0.51+）</p>
            <p className="text-gray-700">前置：安装 <a className="text-blue-600 underline" href="https://nodejs.org/" target="_blank" rel="noopener noreferrer">Node.js</a> 和 <a className="text-blue-600 underline" href="https://git-scm.com/downloads" target="_blank" rel="noopener noreferrer">Git Bash</a></p>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`npm install -g https://keyswift.top/install --registry=https://registry.npmmirror.com`}</code></pre>
          </div>

          <div className="mt-8">
            <h3 className="text-xl font-semibold mb-2">1.3 第三方客户端调用（已开放）</h3>
            <p className="font-medium">Roo Code / Kilo Code</p>
            <ul className="list-disc pl-6 text-gray-800">
              <li>供应商选择：Anthropic</li>
              <li>填写 API 密钥</li>
              <li>自定义基础 URL：<code>https://keyswift.top/claude</code></li>
            </ul>
            <p className="font-medium mt-3">Cherry Studio</p>
            <ul className="list-disc pl-6 text-gray-800">
              <li>新增供应商平台，类型选择 Anthropic</li>
              <li>填写 API 密钥，API 地址：<code>https://keyswift.top/claude</code></li>
              <li>在「管理」中拉取模型加入</li>
            </ul>
            <p className="font-medium mt-3">ChatBox</p>
            <ul className="list-disc pl-6 text-gray-800">
              <li>模型提供方：Claude</li>
              <li>填写 API 密钥</li>
              <li>API 地址：<code>https://keyswift.top/claude/v1</code></li>
            </ul>
            <p className="font-medium mt-3">VS Code</p>
            <ul className="list-disc pl-6 text-gray-800">
              <li>安装官方 Claude Code 插件</li>
              <li>强制登录方案（如需）：创建 <code>~/.claude/config.json</code> 内容：</li>
            </ul>
            <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`{
  "primaryApiKey": "fox"
}`}</code></pre>
          </div>
        </section>

        <section className="mt-12">
          <h2 id="codex" className="text-2xl font-semibold mb-3">二、CodeX 安装教程（与 CC 额度通用）</h2>
          <h3 className="text-xl font-semibold mt-2 mb-2">2.1 安装</h3>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`npm install -g @openai/codex`}</code></pre>

          <h3 className="text-xl font-semibold mt-4 mb-2">2.2 配置</h3>
          <p className="font-medium">编辑/创建 <code>~/.codex/config.toml</code></p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`model_provider = "fox"
model = "gpt-5"
model_reasoning_effort = "high"
disable_response_storage = true

[model_providers.fox]
name = "fox"
base_url = "https://keyswift.top/codex/v1"
wire_api = "responses"
requires_openai_auth = true`}</code></pre>
          <p className="font-medium mt-3">编辑/创建 <code>~/.codex/auth.json</code></p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`{
  "OPENAI_API_KEY": "替换为您的API Key"
}`}</code></pre>
          <p className="text-gray-700 mt-2">切换新模型：</p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`codex -m gpt-5-codex`}</code></pre>

          <h3 className="text-xl font-semibold mt-4 mb-2">2.3 在 VS Code / Cherry 使用</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>VS Code：安装官方插件，按上方配置即可。</li>
            <li>Cherry：已兼容老版 <code>/v1/chat/completions</code>，可供 Cherry / Roo / Kilo 调用。</li>
          </ul>
        
          <h3 className="text-xl font-semibold mt-4 mb-2">2.4 命令白名单（可选，推荐）</h3>
          <p className="text-gray-700">Windows 用户：编辑/创建 <code>C:\Users\quanliangwei\.codex\rules\default.rules</code>，将以下规则追加到文件末尾：</p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# 通用读取与搜索命令（最核心，覆盖大部分日常操作）
prefix_rule(pattern=["powershell.exe", "-Command", "rg"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Select-String"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Get-ChildItem"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Get-Content"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "sed -n"], decision="allow")

# 通用行号范围打印（常见于查看文件片段）
prefix_rule(pattern=["powershell.exe", "-Command", "$start="], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "for($i="], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "$lines["], decision="allow")  # 数组索引访问

# 通用内容替换与修改（谨慎使用，允许常见替换操作）
prefix_rule(pattern=["powershell.exe", "-Command", "-replace"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Set-Content -Path"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Set-Content -Value"], decision="allow")

# git 常用查看命令
prefix_rule(pattern=["powershell.exe", "-Command", "git status"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "git diff"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "git log"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "git show"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "git grep"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "git ls-files"], decision="allow")

# 打开文件夹或编辑器（通用）
prefix_rule(pattern=["powershell.exe", "-Command", "code "], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "explorer.exe"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Start-Process explorer.exe"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Start-Process -FilePath explorer.exe"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "cmd /c start"], decision="allow")

# 其他常用命令
prefix_rule(pattern=["powershell.exe", "-Command", "go build"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "go mod tidy"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "cmd /c dir"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "cmd /c type"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "cmd /c findstr"], decision="allow")

# 允许检查 rg 是否存在（常见条件判断）
prefix_rule(pattern=["powershell.exe", "-Command", "if (Get-Command rg"], decision="allow")

# 允许列出当前目录或递归文件列表（不带具体路径）
prefix_rule(pattern=["powershell.exe", "-Command", "Get-ChildItem -Recurse"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Get-ChildItem -Force"], decision="allow")
prefix_rule(pattern=["powershell.exe", "-Command", "Select-String -Path"], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "Select-String -LiteralPath"], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$content = Get-Content -Path"], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$content= Get-Content -Path"], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$content = Get-Content -LiteralPath"], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$content= Get-Content -LiteralPath"], decision="allow")


prefix_rule(pattern=["powershell.exe", "-Command", "$p ="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$p="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$path ="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$path="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$c ="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$c="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ $p ="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ $p="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ $path ="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ $path="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ $c ="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ $c="], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command"], decision="allow")

prefix_rule(pattern=["powershell", "-Command"], decision="allow")

prefix_rule(pattern=["pwsh.exe", "-Command"], decision="allow")


prefix_rule(pattern=["powershell.exe"], decision="allow")

prefix_rule(pattern=["powershell"], decision="allow")

prefix_rule(pattern=["pwsh.exe"], decision="allow")

prefix_rule(pattern=["cmd"], decision="allow")

prefix_rule(pattern=["cmd.exe"], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$ "], decision="allow")

prefix_rule(pattern=["powershell.exe", "-Command", "$"], decision="allow")

prefix_rule(pattern=["powershell", "-Command", "$ "], decision="allow")

prefix_rule(pattern=["powershell", "-Command", "$"], decision="allow")

prefix_rule(pattern=["pwsh.exe", "-Command", "$ "], decision="allow")

prefix_rule(pattern=["pwsh.exe", "-Command", "$"], decision="allow")


prefix_rule(pattern=["rg"], decision="allow")

prefix_rule(pattern=["$ "], decision="allow")

prefix_rule(pattern=["$"], decision="allow")

prefix_rule(pattern=["Select-String"], decision="allow")

prefix_rule(pattern=["Get-Content"], decision="allow")

prefix_rule(pattern=["Set-Content"], decision="allow")
`}</code></pre></section>

        <section className="mt-12">
          <h2 id="gemini" className="text-2xl font-semibold mb-3">三、Gemini CLI 安装教程</h2>
          <h3 className="text-xl font-semibold mt-2 mb-2">3.1 安装</h3>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`npm install -g @google/gemini-cli`}</code></pre>
          <h3 className="text-xl font-semibold mt-4 mb-2">3.2 配置</h3>
          <p className="font-medium">编辑 <code>~/.gemini/.env</code></p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`GOOGLE_GEMINI_BASE_URL=https://keyswift.top/gemini
GEMINI_API_KEY=你的APIKey
GEMINI_MODEL=gemini-3-pro-preview`}</code></pre>
          <p className="font-medium mt-3">编辑/创建 <code>~/.gemini/settings.json</code></p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`{
  "ide": { "enabled": true },
  "security": { "auth": { "selectedType": "gemini-api-key" } }
}`}</code></pre>
          <p className="text-gray-700 mt-2">若出现 401：在 CLI 输入 <code>/auth</code>，按提示填入 Key。</p>
        </section>

        <section className="mt-12">
          <h2 id="docs" className="text-2xl font-semibold mb-3">四、Claude Code 官方中文文档</h2>
          <a className="text-blue-600 underline" href="https://docs.anthropic.com/zh-CN/docs/claude-code/quickstart" target="_blank" rel="noopener noreferrer">https://docs.anthropic.com/zh-CN/docs/claude-code/quickstart</a>
        </section>

        <section className="mt-12">
          <h2 id="features" className="text-2xl font-semibold mb-3">五、Claude Code 功能与用法</h2>
          <h3 className="text-xl font-semibold mt-2 mb-2">5.1 交互方式</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>交互模式：运行 <code>claude</code> 启动 REPL 会话</li>
            <li>单次模式：<code>claude -p "查询"</code> 快速命令</li>
          </ul>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# 启动交互模式
claude

# 以初始查询启动
claude "解释这个项目"

# 运行单个命令并退出
claude -p "这个函数做什么？"

# 处理管道内容
cat logs.txt | claude -p "分析这些错误"`}</code></pre>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.2 IDE 集成</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>VS Code / JetBrains 官方插件可直接联动，在终端唤起 Claude Code 后自动或手动安装。</li>
            <li>JetBrains 插件：<a className="text-blue-600 underline" href="https://plugins.jetbrains.com/plugin/22707-claude-code-beta-" target="_blank" rel="noopener noreferrer">Claude Code [Beta] - IntelliJ IDEs Plugin</a></li>
            <li>如需手动选择 IDE，可在 Claude 会话中使用命令：<code>/ide</code></li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.3 连接 Cursor（示例）</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>方法一：直接安装相关插件</li>
            <li>方法二：基于 WSL（在 Cursor 中连接 Ubuntu 终端运行 Claude Code，并可视化代码改动）</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.4 模型与上下文</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>推荐使用 Claude 4 Sonnet（默认），体验接近 Opus，费用约 1/5。</li>
            <li>长上下文建议使用 <code>/compact [instructions]</code> 进行压缩，节省额度。</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.5 对话恢复</h3>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# 立即恢复最近会话
claude --continue

# 交互式选择恢复
claude --resume`}</code></pre>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.6 处理图像</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>拖拽到窗口或粘贴（macOS），或提供路径：<code>分析这个图像：/path/to/image.png</code></li>
            <li>示例需求：错误截图诊断、UI 元素描述、生成匹配 CSS/HTML 结构。</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.7 深入思考</h3>
          <p className="text-gray-700">用自然语言要求其进行更深入的架构/安全/边界分析（更耗额度）。</p>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.8 记忆（CLAUDE.md）</h3>
          <p className="text-gray-700">用 <code>/init</code> 初始化 CLAUDE.md，记录命令、约定、架构与偏好，便于团队共享与个性化。</p>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.9 CI 与自动化</h3>
          <p className="text-gray-700">非交互模式便于脚本/流水线使用：</p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`claude -p "使用最新更改更新 README" --allowedTools "Bash(git diff:*)" "Bash(git log:*)" Write --disallowedTools ..`}</code></pre>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.10 模型上下文协议（MCP）</h3>
          <p className="text-gray-700">开放协议，支持访问外部工具和数据源；Claude Code 可作为 MCP 客户端/服务器。参考官方文档了解配置。</p>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.11 Git 工作树并行会话</h3>
          <p className="text-gray-700">在隔离目录中并行进行多个任务：</p>
          <pre className="mt-2 bg-gray-900 text-gray-100 p-3 rounded overflow-auto"><code>{`# 创建带新分支的工作树
git worktree add ../project-feature-a -b feature-a
# 使用现有分支创建
git worktree add ../project-bugfix bugfix-123

# 在各自目录运行
cd ../project-feature-a && claude
cd ../project-bugfix && claude

# 管理
git worktree list
git worktree remove ../project-feature-a`}</code></pre>
          <ul className="list-disc pl-6 text-gray-800 mt-2">
            <li>每个工作树互不影响，共享历史与远程。</li>
            <li>注意为每个工作树初始化依赖环境。</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.12 自然语言范例</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>识别未文档化代码：在 auth 模块中查找缺少 JSDoc 的函数</li>
            <li>生成文档：为 auth.js 未文档化函数添加 JSDoc</li>
            <li>理解代码：支付处理做什么？权限检查在哪？缓存层如何工作？</li>
            <li>智能编辑：表单校验、日志重构为新 API、修复竞态条件</li>
            <li>测试/安全：运行测试并修复失败、查找修复漏洞、解释失败原因</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.13 常见斜杠命令</h3>
          <p className="text-gray-700">/bug、/clear、/compact、/config、/cost、/doctor、/help、/init、/login、/logout、/memory、/pr_comments、/review、/status、/terminal-setup、/vim 等</p>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.14 快捷键与多行输入</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li># 开头快速添加记忆（系统会提示选择记忆文件）</li>
            <li>多行命令：输入 <code>\</code> 后回车，或配置 Option+Enter / Shift+Enter</li>
            <li>终端设置：Mac Terminal 勾选“将 Option 用作 Meta”；iTerm2/VSCode 将 Option 设为 “Esc+”</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.15 Vim 模式（可 /vim 开启或 /config 配置）</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>模式：Esc、i/I、a/A、o/O</li>
            <li>导航：h/j/k/l，w/e/b，0/$/^，gg/G</li>
            <li>编辑：x，dw/de/db/dd/D，cw/ce/cb/cc/C，.（重复）</li>
          </ul>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.16 常见错误</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>400 invalid_request_error：请求格式或内容问题</li>
            <li>401 authentication_error：API Key 问题</li>
            <li>403 permission_error：无权限</li>
            <li>404 not_found_error：资源不存在</li>
            <li>413 request_too_large：请求过大，建议 <code>/compact</code></li>
            <li>429 rate_limit_error：触发限流</li>
            <li>500 api_error：服务内部错误</li>
            <li>529 overloaded_error：API 过载。建议逐步增加流量并保持稳定模式</li>
          </ul>
          <p className="text-gray-700 mt-2">SSE 流式响应可能在 200 后仍抛错，需额外处理。</p>

          <h3 className="text-xl font-semibold mt-6 mb-2">5.17 更多高级功能与安全</h3>
          <ul className="list-disc pl-6 text-gray-800">
            <li>类 Unix 工具用法、自定义斜杠命令、<code>$ARGUMENTS</code> 参数等</li>
            <li>高级设置、命令行参数、本地/共享/用户设置</li>
            <li>权限与安全：参考官方文档“管理权限和安全”</li>
          </ul>
        </section>
      </div>
    </div>
  );
}

