---
name: ucloud-sandbox
description: 当用户需要在 Linux、macOS 或 Windows 中用 UCloud Sandbox CLI 操作沙箱服务时使用，包括安装或配置 ucloud-sandbox-cli、设置 API Key 和地域、创建/连接/执行/暂停/终止沙箱、浏览和管理沙箱文件、上传或下载文件、查看端口地址和监控指标、管理快照与模板，以及在 Claude Code、Codex、Gemini 等 Agent 中安装本技能。
---

# UCloud Sandbox CLI

使用 `ucloud-sandbox-cli` 管理 UCloud Sandbox 沙箱、快照和模板。优先用 CLI 完成操作；如果用户只是在询问命令，给出可复制的命令即可。

## 前置检查：公网权限

执行安装、更新或调用 UCloud Sandbox API 前，先确认当前环境允许访问公网。需要网络权限审批时先申请授权；未获授权时停止并说明原因。仅检查本地版本或提供命令说明时无需申请。

## 平台说明

在 Windows 或 PowerShell 环境中，执行安装、更新、认证配置、文件传输或其他 CLI 操作前，先完整阅读并遵循 [Windows PowerShell 指南](references/windows.md)。本页的 Bash 安装、更新和配置脚本仅适用于 Linux 和 macOS。

## 安装本技能

仅在用户要求“安装这个 skill/技能”时执行。Linux 和 macOS 把 `SKILL.md` 及其引用放到目标 Agent 的技能目录。可设置 `TARGET_AGENT=codex|claude|gemini|auto`，默认自动检测：

```bash
set -eu

SKILL_NAME="ucloud-sandbox"
SKILL_URL="https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/skills/ucloud-sandbox/SKILL.md"
WINDOWS_REFERENCE_URL="https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/skills/ucloud-sandbox/references/windows.md"
TARGET_AGENT="${TARGET_AGENT:-auto}"

case "$TARGET_AGENT" in
  codex)
    SKILL_DIR="${CODEX_HOME:-$HOME/.codex}/skills/$SKILL_NAME"
    ;;
  claude | claude-code)
    SKILL_DIR="$HOME/.claude/skills/$SKILL_NAME"
    ;;
  gemini)
    SKILL_DIR="$HOME/.gemini/skills/$SKILL_NAME"
    ;;
  auto)
    if [ -n "${CODEX_HOME:-}" ] || [ -d "$HOME/.codex" ]; then
      SKILL_DIR="${CODEX_HOME:-$HOME/.codex}/skills/$SKILL_NAME"
    elif [ -d "$HOME/.claude" ]; then
      SKILL_DIR="$HOME/.claude/skills/$SKILL_NAME"
    elif [ -d "$HOME/.gemini" ]; then
      SKILL_DIR="$HOME/.gemini/skills/$SKILL_NAME"
    else
      SKILL_DIR="$HOME/.codex/skills/$SKILL_NAME"
    fi
    ;;
  *)
    echo "TARGET_AGENT must be codex, claude, gemini, or auto" >&2
    exit 1
    ;;
esac

mkdir -p "$SKILL_DIR/references"
curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"
curl -fsSL "$WINDOWS_REFERENCE_URL" -o "$SKILL_DIR/references/windows.md"
echo "ucloud-sandbox skill installed to $SKILL_DIR"
```

Linux 和 macOS 常见目录：

- Codex: `TARGET_AGENT=codex`，目录为 `${CODEX_HOME:-$HOME/.codex}/skills/ucloud-sandbox`
- Claude Code: `TARGET_AGENT=claude`，目录为 `$HOME/.claude/skills/ucloud-sandbox`
- Gemini CLI: `TARGET_AGENT=gemini`，目录为 `$HOME/.gemini/skills/ucloud-sandbox`

## Step 0：确保 CLI 可用

每次准备执行真实操作前先检查。Linux 和 macOS 使用以下 Bash 流程。

### Linux 和 macOS

```bash
OLD_NPM_PACKAGE="@ucloud-sdks/ucloud-sandbox-cli"

if command -v npm >/dev/null 2>&1 && npm list -g "$OLD_NPM_PACKAGE" --depth=0 >/dev/null 2>&1; then
  echo "OLD_NPM_CLI_FOUND"
elif command -v ucloud-sandbox-cli >/dev/null 2>&1; then
  ucloud-sandbox-cli version
else
  echo "NOT_INSTALLED"
fi
```

如果输出 `OLD_NPM_CLI_FOUND`，说明安装的是 v1.0 及以前的 npm 版旧 CLI。先卸载旧版，再安装新版二进制：

```bash
npm uninstall -g @ucloud-sdks/ucloud-sandbox-cli
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh
ucloud-sandbox-cli version
```

如果全局 npm 卸载需要权限或交互确认，提示用户在真实终端执行卸载命令。

如果未安装，使用官方安装脚本：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh
```

非交互或自动化安装：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y
```

安装到自定义目录：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -p "$HOME/.local/bin"
export PATH="$HOME/.local/bin:$PATH"
```

安装后运行：

```bash
ucloud-sandbox-cli version
```

如果提示命令不存在，说明安装目录不在 `PATH` 中。引导用户把安装目录加入 `PATH`。

## 按需更新 Skill 和 CLI

仅当用户明确要求“更新 skill/技能”或“更新 ucloud-sandbox-cli/命令行”时执行；不要在普通沙箱操作前自动更新。

Linux 和 macOS 更新本技能：

```bash
set -eu

SKILL_NAME="ucloud-sandbox"
SKILL_URL="https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/skills/ucloud-sandbox/SKILL.md"
WINDOWS_REFERENCE_URL="https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/skills/ucloud-sandbox/references/windows.md"
TARGET_AGENT="${TARGET_AGENT:-auto}"

case "$TARGET_AGENT" in
  codex)
    SKILL_DIR="${CODEX_HOME:-$HOME/.codex}/skills/$SKILL_NAME"
    ;;
  claude | claude-code)
    SKILL_DIR="$HOME/.claude/skills/$SKILL_NAME"
    ;;
  gemini)
    SKILL_DIR="$HOME/.gemini/skills/$SKILL_NAME"
    ;;
  auto)
    if [ -n "${CODEX_HOME:-}" ] || [ -d "$HOME/.codex" ]; then
      SKILL_DIR="${CODEX_HOME:-$HOME/.codex}/skills/$SKILL_NAME"
    elif [ -d "$HOME/.claude" ]; then
      SKILL_DIR="$HOME/.claude/skills/$SKILL_NAME"
    elif [ -d "$HOME/.gemini" ]; then
      SKILL_DIR="$HOME/.gemini/skills/$SKILL_NAME"
    else
      SKILL_DIR="$HOME/.codex/skills/$SKILL_NAME"
    fi
    ;;
  *)
    echo "TARGET_AGENT must be codex, claude, gemini, or auto" >&2
    exit 1
    ;;
esac

mkdir -p "$SKILL_DIR/references"
curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"
curl -fsSL "$WINDOWS_REFERENCE_URL" -o "$SKILL_DIR/references/windows.md"
echo "ucloud-sandbox skill updated at $SKILL_DIR"
```

Linux 和 macOS 更新 `ucloud-sandbox-cli` 到最新版本：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y
ucloud-sandbox-cli version
```

更新到指定版本：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y -v v1.2.3
ucloud-sandbox-cli version
```

如果原先安装在自定义目录，更新时继续传入同一个目录：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y -p "$HOME/.local/bin"
```

## Step 1：认证和地域配置

API Key 可从星图平台密钥管理获取：`https://astraflow.ucloud.cn/modelverse/api-keys`。常用地域包括 `cn-wlcb` 和 `us-ca`；不确定时询问用户。

持久化配置文件路径是 `~/.ucloud-sandbox-cli/config.json`。Linux 和 macOS 建议目录权限为 `700`、文件权限为 `600`。配置文件是 JSON，格式如下；展示或读取时必须隐藏 `api_key`：

```json
{
  "api_key": "<api-key>",
  "region": "cn-wlcb",
  "domain": "cn-wlcb.sandbox.ucloudai.com"
}
```

字段说明：

- `api_key`：用户的 UCloud Sandbox API Key，必须保密。
- `region`：地域，例如 `cn-wlcb` 或 `us-ca`。
- `domain`：可选，显式 API 域名；存在时优先于 `region` 推导出的默认域名。

不要让 Agent 执行 `ucloud-sandbox-cli login` 或 `ucloud-sandbox-cli region`。这两个命令需要真实交互式终端，Agent/CI 中通常会失败。

如果用户没有配置 API Key，不要让 Agent 询问或读取用户的 API Key 后写入配置文件。提示用户在真实终端中手动登录：

```bash
ucloud-sandbox-cli login
```

并说明 API Key 可从星图平台 Key 管理获取：`https://astraflow.ucloud.cn/modelverse/api-keys`。

如果用户明确希望使用环境变量，给出命令让用户自己在终端设置，适合临时会话和 CI：

```bash
export UCLOUD_SANDBOX_API_KEY="<api-key>"
export UCLOUD_SANDBOX_REGION="cn-wlcb"
```

切换持久化地域时，Agent 不执行 `ucloud-sandbox-cli region`，直接修改已有配置文件的 `region` 字段。修改前先确认配置文件存在；如果不存在，提示用户先在真实终端执行 `ucloud-sandbox-cli login`。

Linux 和 macOS：

```bash
CONFIG_FILE="$HOME/.ucloud-sandbox-cli/config.json"
NEW_REGION="cn-wlcb"

if [ ! -f "$CONFIG_FILE" ]; then
  echo "Config file not found. Please run 'ucloud-sandbox-cli login' in a real terminal first." >&2
  exit 1
fi

tmp="$(mktemp)"
jq --arg region "$NEW_REGION" '.region = $region' "$CONFIG_FILE" > "$tmp"
mv "$tmp" "$CONFIG_FILE"
chmod 600 "$HOME/.ucloud-sandbox-cli/config.json"
```

Linux 和 macOS 如果没有 `jq`，不要用易误伤 `api_key` 的字符串替换方案；提示用户安装 `jq`，或让用户在真实终端运行 `ucloud-sandbox-cli region` 自行切换。

需要读取配置确认地域或域名时，必须隐藏 `api_key`，不要 `cat ~/.ucloud-sandbox-cli/config.json`。优先只读取必要字段：

Linux 和 macOS：

```bash
jq -r '.region // empty' "$HOME/.ucloud-sandbox-cli/config.json"
jq -r '.domain // empty' "$HOME/.ucloud-sandbox-cli/config.json"
```

如果必须展示配置摘要，先脱敏：

```bash
jq '.api_key = if .api_key then "***hidden***" else . end' "$HOME/.ucloud-sandbox-cli/config.json"
```

Linux 和 macOS 没有 `jq` 时，使用不会输出真实密钥的方式：

```bash
sed -E 's/"api_key"[[:space:]]*:[[:space:]]*"[^"]*"/"api_key": "***hidden***"/' "$HOME/.ucloud-sandbox-cli/config.json"
```

退出登录并删除本地凭据：

```bash
ucloud-sandbox-cli logout
```

## Agent 操作原则

- 在 Claude Code、Codex、Gemini、CI 等非 TTY 环境中，创建或克隆沙箱时默认加 `--detach`，否则 CLI 会尝试进入交互终端。
- 不要执行 `ucloud-sandbox-cli login` 或 `ucloud-sandbox-cli region`；让用户在真实终端执行登录，Agent 只通过修改已有配置文件切换地域。
- 当本地没有 API Key 配置时，不要向用户索取 API Key 并代写配置；提示用户在真实终端运行 `ucloud-sandbox-cli login`，API Key 从星图平台 Key 管理获取。
- 需要解析列表时优先用 `--format json` 或 `-f json`。
- 执行破坏性命令前先确认用户意图：`sandbox kill`、`sandbox kill --all`、`fs rm`、`snapshot delete`、`template delete`、`template publish --unpublish`。
- 不要在回复、日志或命令输出中泄露 API Key；读取 `~/.ucloud-sandbox-cli/config.json` 时必须隐藏 `api_key`，只展示地域、域名等必要字段。
- 用户要打开交互式终端时，建议让用户在真实终端中运行 `sandbox connect`。

## Sandbox 常用操作

创建沙箱：

```bash
# Agent/CI 推荐：创建后不连接终端
ucloud-sandbox-cli sandbox create base --detach
ucloud-sandbox-cli sbx cr base --detach

# 指定超时时间，单位秒
ucloud-sandbox-cli sandbox create base --timeout 3600 --detach
```

常见模板：`base`、`code-interpreter-v1`、`desktop`，也可以使用用户自己的模板 ID 或名称。

列出沙箱：

```bash
ucloud-sandbox-cli sandbox list
ucloud-sandbox-cli sandbox list --state running
ucloud-sandbox-cli sandbox list --format json
```

连接已有沙箱：

```bash
ucloud-sandbox-cli sandbox connect <sandbox-id>
```

在沙箱中执行命令。命令参数要作为一个字符串传入：

```bash
ucloud-sandbox-cli sandbox exec <sandbox-id> "pwd && ls -la"
ucloud-sandbox-cli sandbox exec <sandbox-id> "python --version"
```

获取沙箱端口访问地址：

```bash
ucloud-sandbox-cli sandbox host <sandbox-id> 3000
```

查看资源指标：

```bash
ucloud-sandbox-cli sandbox metrics <sandbox-id>
ucloud-sandbox-cli sandbox metrics <sandbox-id> --since 1h
ucloud-sandbox-cli sandbox metrics <sandbox-id> --raw
ucloud-sandbox-cli sandbox metrics <sandbox-id> --watch
```

暂停、克隆、终止：

```bash
ucloud-sandbox-cli sandbox pause <sandbox-id>
ucloud-sandbox-cli sandbox clone <sandbox-id> --detach
ucloud-sandbox-cli sandbox kill <sandbox-id>
ucloud-sandbox-cli sandbox kill --all --state running
```

## 文件系统常用操作

文件系统命令统一使用 `ucloud-sandbox-cli fs`。除 `cp` 外，第一个参数都是沙箱 ID；路径可以是沙箱内的绝对路径或相对路径。

### 浏览文件

列出当前默认目录、指定目录或单个文件：

```bash
ucloud-sandbox-cli fs ls <sandbox-id>
ucloud-sandbox-cli fs ls <sandbox-id> /home/user
ucloud-sandbox-cli fs ls <sandbox-id> /home/user/app/package.json
```

需要稳定解析结果时使用 JSON 格式：

```bash
ucloud-sandbox-cli fs ls <sandbox-id> /home/user/app --format json
ucloud-sandbox-cli fs ls <sandbox-id> /home/user/app -f json
```

### 读取文件

把远端文件内容输出到标准输出：

```bash
ucloud-sandbox-cli fs cat <sandbox-id> /home/user/app/package.json
```

`fs cat` 不会自动脱敏。读取 `.env`、凭证、配置文件或其他可能包含密钥的文件前，先确认确有必要；不要把敏感内容直接回显给用户或写入日志。二进制文件和需要保存到本地的文件使用 `fs cp` 下载，不要使用 `fs cat`。

### 创建目录

```bash
ucloud-sandbox-cli fs mkdir <sandbox-id> /home/user/app
```

目录已存在时命令仍然成功，并输出 `Directory already exists`。`fs mkdir` 一次接收一个目录；创建多层目录时从已有父目录开始逐层执行。确需递归创建时，可以使用 `sandbox exec <sandbox-id> "mkdir -p <dir>"`，但必须先校验路径，避免拼接未经验证的用户输入。

### 上传和下载文件

`fs cp` 的用法是 `fs cp <src-path> <dest-path>`。远端端点必须写成 `<sandbox-id>:<path>`，源和目标中必须恰好有一个远端端点；不支持远端到远端复制，也不递归复制目录。

上传单个文件：

```bash
ucloud-sandbox-cli fs cp ./index.html <sandbox-id>:/home/user/app/index.html
```

上传到远端目录并保留本地文件名时，远端目标必须以 `/` 结尾：

```bash
ucloud-sandbox-cli fs cp ./index.html <sandbox-id>:/home/user/app/
```

下载文件：

```bash
ucloud-sandbox-cli fs cp <sandbox-id>:/home/user/app/output.txt ./output.txt
```

如果本地目标是已经存在的目录，CLI 会保留远端文件名：

```bash
ucloud-sandbox-cli fs cp <sandbox-id>:/home/user/app/output.txt ./downloads/
```

传输目录时，先用 `tar` 等工具打包成单个文件，上传或下载后再解包。打包部署内容时排除 `.git`、`.env`、API Key、依赖缓存和其他不应传输的敏感或冗余文件。

### 移动和重命名

在同一个沙箱内移动或重命名路径：

```bash
ucloud-sandbox-cli fs mv <sandbox-id> /home/user/app/old.txt /home/user/app/new.txt
```

`fs mv` 不能跨沙箱移动，也不能在本地与沙箱之间移动；这两类场景使用 `fs cp`，确认复制成功后再按用户意图决定是否删除源文件。

### 删除文件或目录

```bash
ucloud-sandbox-cli fs rm <sandbox-id> /home/user/app/obsolete.txt
```

`fs rm` 是破坏性操作且没有交互确认。执行前确认沙箱 ID 和准确路径，必要时先用 `fs ls` 检查目标；不要删除根目录、用户主目录、来源不明的目录或未经用户授权的数据，也不要把未经验证的用户输入直接作为删除路径。

## Snapshot 常用操作

从沙箱创建快照：

```bash
ucloud-sandbox-cli snapshot create <sandbox-id>
ucloud-sandbox-cli snap cr <sandbox-id>
```

列出快照：

```bash
ucloud-sandbox-cli snapshot list
ucloud-sandbox-cli snapshot list --sandbox-id <sandbox-id>
ucloud-sandbox-cli snapshot list --format json
```

删除快照：

```bash
ucloud-sandbox-cli snapshot delete <snapshot-id>
ucloud-sandbox-cli snapshot delete <snapshot-id-1> <snapshot-id-2>
```

从快照启动新沙箱时，把快照 ID 当作模板传给 `sandbox create`：

```bash
ucloud-sandbox-cli sandbox create <snapshot-id> --detach
```

## Template 常用操作

初始化模板目录：

```bash
ucloud-sandbox-cli template init my-template
ucloud-sandbox-cli tpl init my-template --cpu 2 --memory 1024 --from base
cd my-template
```

编辑生成的 `template.dockerfile`，加入模板构建需要的 `RUN`、`COPY` 等步骤。注意：当前 CLI 的 `template build` 命令以服务端模板构建流程为准；如果用户依赖复杂 Dockerfile 语法，先运行 `ucloud-sandbox-cli template build --help` 并按当前版本支持能力调整。

构建模板：

```bash
ucloud-sandbox-cli template build my-template
ucloud-sandbox-cli tpl build my-template --cpu-count 2 --memory-mb 1024
ucloud-sandbox-cli tpl build my-template --no-cache
ucloud-sandbox-cli tpl build my-template --publish
```

如果提供启动命令和就绪探针，`--cmd` 与 `--ready-cmd` 必须一起使用：

```bash
ucloud-sandbox-cli tpl build my-template \
  --cmd "python app.py" \
  --ready-cmd "curl -f http://localhost:8000/health"
```

列出模板：

```bash
ucloud-sandbox-cli template list
ucloud-sandbox-cli template list --format json
```

发布或取消发布模板：

```bash
ucloud-sandbox-cli template publish <template-id>
ucloud-sandbox-cli template publish --unpublish <template-id>
```

删除模板：

```bash
ucloud-sandbox-cli template delete <template-id>
ucloud-sandbox-cli template delete --select
```

注意：模板名称只能包含小写字母、数字、短横线和下划线，且不能以短横线或下划线开头/结尾。内存值必须是偶数 MB。

## 常见工作流

准备环境：

```bash
ucloud-sandbox-cli version
```

如果未登录，提示用户在真实终端执行 `ucloud-sandbox-cli login`，不要由 Agent 执行该交互命令。

创建沙箱并执行命令：

```bash
ucloud-sandbox-cli sandbox create base --detach
ucloud-sandbox-cli sandbox list --format json
ucloud-sandbox-cli sandbox exec <sandbox-id> "echo hello from sandbox"
ucloud-sandbox-cli sandbox kill <sandbox-id>
```

保存沙箱状态并复用：

```bash
ucloud-sandbox-cli snapshot create <sandbox-id>
ucloud-sandbox-cli sandbox create <snapshot-id> --detach
```

构建自定义模板：

```bash
ucloud-sandbox-cli template init my-agent-env
cd my-agent-env
# edit template.dockerfile
ucloud-sandbox-cli template build my-agent-env
ucloud-sandbox-cli sandbox create <template-id-or-name> --detach
```

## 故障处理

| 现象 | 处理 |
| --- | --- |
| `API key is required` | 提示用户在真实终端运行 `ucloud-sandbox-cli login`，或由用户自行设置 `UCLOUD_SANDBOX_API_KEY`；API Key 从星图平台 Key 管理获取 |
| 命令安装成功但找不到 | Linux/macOS 使用 `export PATH="$HOME/.local/bin:$PATH"`；Windows 参见 [Windows 故障处理](references/windows.md#故障处理) |
| 创建沙箱后卡在终端 | Agent/CI 中使用 `sandbox create ... --detach` |
| `template not found` | 运行 `template list --format json` 确认模板 ID/名称 |
| `sandbox not found` | 运行 `sandbox list --format json` 确认沙箱仍在运行 |
| metrics 时间参数不识别 | 使用 `--since 1h`、`--start "2026-06-23 12:00"` 这类格式 |

更多 CLI 用法参考：`https://astraflow.ucloud.cn/docs/agent-sandbox/product/cli`。
