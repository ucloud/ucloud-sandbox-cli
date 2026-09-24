---
name: ucloud-sandbox
description: 当用户需要在 Linux、macOS 或 Windows 中用 UCloud Sandbox CLI 操作沙箱服务时使用，包括安装、自我更新或配置 ucloud-sandbox-cli、设置 API Key 和地域、配置镜像仓库凭据、创建/连接/执行/暂停/复制/终止沙箱、管理持久化 Volume、创建沙箱时挂载 Volume、浏览和管理沙箱文件、上传或下载文件、查看端口地址、监控指标和日志、管理 Secret、快照与模板，以及在 Claude Code、Codex、Gemini 等 Agent 中安装本技能。
---

# UCloud Sandbox CLI

使用 `ucloud-sandbox-cli` 管理 UCloud Sandbox 沙箱、持久化 Volume、Secret、快照和模板。优先用 CLI 完成操作；如果用户只是在询问命令，给出可复制的命令即可。

## 命令总览

```
auth      login / logout / region / config / registry login / registry logout
sandbox   create / connect / exec / get / list / host / logs / metrics
          pause / fork / kill / fs (ls·cat·cp·mkdir·mv·rm)
snapshot  create / list / delete
volume    create / get / list / delete
secret    create / get / list / update / delete
template  init / build / get / list / logs / publish / delete / tag (assign·list·remove)
update    自我更新
version   版本信息
```

认证命令都在 `auth` 下，文件命令都在 `sandbox fs` 下。旧版的顶层 `login`/`logout`/`region`/`config`/`fs` 和 `sandbox clone` 均已不存在。

## 前置检查：公网权限

执行安装、更新或调用 UCloud Sandbox API 前，先确认当前环境允许访问公网。需要网络权限审批时先申请授权；未获授权时停止并说明原因。仅检查本地版本或提供命令说明时无需申请。

## 平台说明

在 Windows 或 PowerShell 环境中，执行安装、更新、认证配置、文件传输或其他 CLI 操作前，先完整阅读并遵循 [Windows PowerShell 指南](references/windows.md)。本页的 Bash 安装、更新和配置脚本仅适用于 Linux 和 macOS。

## 安装本技能

仅在用户要求“安装这个 skill/技能”时执行。Linux 和 macOS 把 `SKILL.md` 放到目标 Agent 的技能目录。可设置 `TARGET_AGENT=codex|claude|gemini|auto`，默认自动检测：

```bash
set -eu

SKILL_NAME="ucloud-sandbox"
SKILL_URL="https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/skills/ucloud-sandbox/SKILL.md"
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

mkdir -p "$SKILL_DIR"
curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"
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

mkdir -p "$SKILL_DIR"
curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"
echo "ucloud-sandbox skill updated at $SKILL_DIR"
```

从 `v1.3.4` 开始，CLI 自带 `update` 命令，可以自我更新到最新版本，不需要重新下载安装脚本。Agent 没有交互式终端，必须带 `-y` 跳过确认，否则命令会一直等待输入：

```bash
ucloud-sandbox-cli update -y
ucloud-sandbox-cli version
```

`update` 会查询 GitHub 最新 Release，下载与当前系统和架构匹配的二进制，校验 SHA256 后替换当前程序。只检查不安装时使用 `--dry-run`，它会打印当前版本、最新版本和下载地址：

```bash
ucloud-sandbox-cli update --dry-run
```

如果 CLI 安装在 `/usr/local/bin` 等需要管理员权限的目录，`update` 会报无权限写入。这时不要让 Agent 自行使用 sudo，提示用户在真实终端执行 `sudo ucloud-sandbox-cli update`。

如果查询版本时报 GitHub API 限流，让用户设置 `GITHUB_TOKEN` 环境变量后重试。

`v1.3.3` 及以前的版本没有 `update` 命令，需要更新到指定版本或改用其他安装目录时，仍然使用安装脚本：

```bash
# 更新到最新版本
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y

# 更新到指定版本
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y -v v1.3.4

# 原先安装在自定义目录时，继续传入同一个目录
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y -p "$HOME/.local/bin"
```

## Step 1：认证和地域配置

API Key 可从星图平台密钥管理获取：`https://astraflow.ucloud.cn/modelverse/api-keys`。常用地域包括 `cn-wlcb`、`cn-sh` 和 `us-ca`；不确定时询问用户。

持久化配置文件路径是 `~/.ucloud-sandbox-cli/config.json`。Linux 和 macOS 建议目录权限为 `700`、文件权限为 `600`。配置文件是 JSON，格式如下；展示或读取时必须隐藏 `api_key`：

```json
{
  "api_key": "<api-key>",
  "region": "cn-wlcb",
  "domain": "cn-wlcb.sandbox.ucloudai.com",
  "insecure_http": false,
  "registries": {
    "uhub.service.ucloud.cn": {
      "username": "<username>",
      "password": "<password>"
    }
  }
}
```

字段说明：

- `api_key`：用户的 UCloud Sandbox API Key，必须保密。
- `region`：地域，例如 `cn-wlcb`、`cn-sh` 或 `us-ca`。
- `domain`：可选，显式 API 域名；存在时优先于 `region` 推导出的默认域名。
- `insecure_http`：可选，用 HTTP 而不是 HTTPS 连接控制面和沙箱。
- `registries`：构建模板时拉取私有 base image 用的凭据，按仓库域名索引。其中的 `password` 与 `api_key` 同等敏感，必须保密。

不要让 Agent 执行 `ucloud-sandbox-cli auth login`、`ucloud-sandbox-cli auth region` 或 `ucloud-sandbox-cli auth registry login`。这三个命令需要真实交互式终端，Agent/CI 中通常会失败。

读取配置时优先使用 `ucloud-sandbox-cli auth config`，它输出 JSON 并自动把 `api_key` 和每个仓库的 `password` 换成 `****`，不需要再手工脱敏：

```bash
ucloud-sandbox-cli auth config
```

如果用户没有配置 API Key，不要让 Agent 询问或读取用户的 API Key 后写入配置文件。提示用户在真实终端中手动登录：

```bash
ucloud-sandbox-cli auth login
```

并说明 API Key 可从星图平台 Key 管理获取：`https://astraflow.ucloud.cn/modelverse/api-keys`。

如果用户明确希望使用环境变量，给出命令让用户自己在终端设置，适合临时会话和 CI：

```bash
export UCLOUD_SANDBOX_API_KEY="<api-key>"
export UCLOUD_SANDBOX_REGION="cn-wlcb"
```

切换持久化地域时，Agent 不执行 `ucloud-sandbox-cli auth region`，直接修改已有配置文件的 `region` 字段。修改前先确认配置文件存在；如果不存在，提示用户先在真实终端执行 `ucloud-sandbox-cli auth login`。

Linux 和 macOS：

```bash
CONFIG_FILE="$HOME/.ucloud-sandbox-cli/config.json"
NEW_REGION="cn-wlcb"

if [ ! -f "$CONFIG_FILE" ]; then
  echo "Config file not found. Please run 'ucloud-sandbox-cli auth login' in a real terminal first." >&2
  exit 1
fi

tmp="$(mktemp)"
jq --arg region "$NEW_REGION" '.region = $region' "$CONFIG_FILE" > "$tmp"
mv "$tmp" "$CONFIG_FILE"
chmod 600 "$HOME/.ucloud-sandbox-cli/config.json"
```

Linux 和 macOS 如果没有 `jq`，不要用易误伤 `api_key` 的字符串替换方案；提示用户安装 `jq`，或让用户在真实终端运行 `ucloud-sandbox-cli auth region` 自行切换。

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
ucloud-sandbox-cli auth logout
```

## Agent 操作原则

- 在 Claude Code、Codex、Gemini、CI 等非 TTY 环境中，创建沙箱时默认加 `--detach`，否则 CLI 会尝试进入交互终端。
- 不要执行 `ucloud-sandbox-cli auth login`、`auth region` 或 `auth registry login`；让用户在真实终端执行，Agent 只通过修改已有配置文件切换地域。
- 当本地没有 API Key 配置时，不要向用户索取 API Key 并代写配置；提示用户在真实终端运行 `ucloud-sandbox-cli auth login`，API Key 从星图平台 Key 管理获取。
- 需要解析列表时优先用 `--format json` 或 `-f json`。
- 需要确认的命令必须带 `-y`，否则会在非 TTY 环境下一直等待输入：`sandbox kill --all`、`template delete`、`template publish`、`template tag remove`、`update`。
- 执行破坏性命令前先确认用户意图：`sandbox kill`、`sandbox kill --all`、`volume delete`、`secret delete`、`sandbox fs rm`、`snapshot delete`、`template delete`、`template publish --unpublish`、`auth logout`、`auth registry logout`。
- `sandbox kill --all` 配合 `--limit` 限制单次影响面，并用 `--state`、`--template`、`--metadata` 收窄范围。
- 不要在回复、日志或命令输出中泄露 API Key 和镜像仓库密码；查看配置用 `ucloud-sandbox-cli auth config`（自动脱敏），不要 `cat ~/.ucloud-sandbox-cli/config.json`。
- Secret 的值是只写的，`secret get` 只返回元数据。创建或更新 Secret 时不要把值写在命令行里（会进入 shell 历史和进程列表），用 `--value-stdin` 从标准输入传入。
- 用户要打开交互式终端时，建议让用户在真实终端中运行 `sandbox connect`。
- `sandbox logs -f` 和 `sandbox metrics -w` 会持续阻塞，Agent/CI 中不要使用。

## Sandbox 常用操作

创建沙箱：

```bash
# Agent/CI 推荐：创建后不连接终端
ucloud-sandbox-cli sandbox create base --detach
ucloud-sandbox-cli sbx cr base --detach

# 指定超时时间，单位秒
ucloud-sandbox-cli sandbox create base --timeout 3600 --detach

# 按名称挂载一个持久化 Volume
ucloud-sandbox-cli sandbox create base --mount workspace:/data --detach

# 重复 --mount 可挂载多个 Volume
ucloud-sandbox-cli sandbox create base \
  --mount workspace:/data \
  --mount model-cache:/cache \
  --detach
```

常见模板：`base`、`code-interpreter-v1`、`desktop`，也可以使用用户自己的模板 ID 或名称。

`--mount` 的格式是 `<volume-name>:<absolute-path>`。左侧必须是 Volume 名称而不是 Volume ID，右侧必须是沙箱内的绝对路径；需要多个挂载时重复传入 `--mount`。

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

查看沙箱详情：

```bash
ucloud-sandbox-cli sandbox get <sandbox-id>
```

获取沙箱端口访问地址：

```bash
ucloud-sandbox-cli sandbox host <sandbox-id> 3000
ucloud-sandbox-cli sandbox host <sandbox-id> 3000 --url
```

查看资源指标。`--start` 和 `--end` 接收 Unix 时间戳（秒），不接受 `1h` 这类相对写法：

```bash
ucloud-sandbox-cli sandbox metrics <sandbox-id>
ucloud-sandbox-cli sandbox metrics <sandbox-id> --start "$(date -d '1 hour ago' +%s)"
ucloud-sandbox-cli sandbox metrics <sandbox-id> --raw
ucloud-sandbox-cli sandbox metrics <sandbox-id> --watch
```

查看沙箱日志。`--follow` 会一直阻塞到沙箱停止，Agent/CI 中不要使用：

```bash
ucloud-sandbox-cli sandbox logs <sandbox-id>
ucloud-sandbox-cli sandbox logs <sandbox-id> --level warn
ucloud-sandbox-cli sandbox logs <sandbox-id> --search "panic"
```

暂停、复制、终止：

```bash
ucloud-sandbox-cli sandbox pause <sandbox-id>

# 只保留文件系统，恢复时冷启动
ucloud-sandbox-cli sandbox pause <sandbox-id> --memory=false

# fork 从运行中的沙箱复制副本，取代了旧版的 sandbox clone
ucloud-sandbox-cli sandbox fork <sandbox-id>
ucloud-sandbox-cli sandbox fork <sandbox-id> --count 3

ucloud-sandbox-cli sandbox kill <sandbox-id>

# --all 默认会要求确认，Agent/CI 中必须带 -y，并建议用 --limit 限制影响面
ucloud-sandbox-cli sandbox kill --all --state running --limit 10 -y
```

## Volume 常用操作

创建持久化 Volume；参数是 Volume 名称：

```bash
ucloud-sandbox-cli volume create workspace
ucloud-sandbox-cli vol cr workspace
```

列出 Volume。需要稳定解析名称和 ID 时使用 JSON 格式：

```bash
ucloud-sandbox-cli volume list
ucloud-sandbox-cli volume list --format json
ucloud-sandbox-cli vol ls -f json
```

删除 Volume 时使用 `volume list` 返回的 Volume ID，而不是名称。删除属于破坏性操作，执行前确认目标 ID；可以一次删除多个：

```bash
ucloud-sandbox-cli volume delete <volume-id>
ucloud-sandbox-cli vol dl <volume-id-1> <volume-id-2>
```

当前 CLI 只提供 Volume 的创建、列表、删除和沙箱挂载，不提供直接操作 Volume 内文件或目录的命令。需要访问内容时，先创建沙箱并用 Volume 名称挂载，再通过挂载路径执行 `sandbox exec` 或 `fs` 命令：

```bash
ucloud-sandbox-cli sandbox create base --mount workspace:/data --detach
ucloud-sandbox-cli sandbox exec <sandbox-id> "ls -la /data"
```

## Secret 常用操作

Secret 的值是只写的：创建后任何接口都不会再返回它，`secret get` 只给出元数据。

创建 Secret 时不要把值写在命令行参数里，它会进入 shell 历史和本机的进程列表。Agent/CI 中用 `--value-stdin` 从标准输入传入：

```bash
printf '%s' "$SECRET_VALUE" | ucloud-sandbox-cli secret create openai-key --value-stdin
cat private.pem | ucloud-sandbox-cli secret create deploy-key --value-stdin
```

不带 `--value` 也不带 `--value-stdin` 时命令会提示输入（不回显），这需要真实终端，Agent 中会失败。

查看、更新和删除。更新会新增一个版本并成为默认版本，旧版本保留：

```bash
ucloud-sandbox-cli secret list
ucloud-sandbox-cli secret list --format json
ucloud-sandbox-cli secret get openai-key
printf '%s' "$NEW_VALUE" | ucloud-sandbox-cli secret update openai-key --value-stdin
ucloud-sandbox-cli secret delete openai-key
```

Secret 名称只能包含字母、数字、下划线和短横线。

在沙箱中使用 Secret：创建沙箱时用 `--network-rules` 声明对某个域名的出站请求要加哪些请求头，值里用 `${secret-name}` 引用，平台在出站时替换为真实值，沙箱内部看不到它：

```bash
ucloud-sandbox-cli sandbox create base \
  --network-rules '{"api.example.com": {"X-API-KEY": "${openai-key}"}}' \
  --detach
```

## 文件系统常用操作

文件系统命令统一使用 `ucloud-sandbox-cli sandbox fs`（旧版的顶层 `fs` 已经不存在）。除 `cp` 外，第一个参数都是沙箱 ID；路径可以是沙箱内的绝对路径或相对路径。

每个 `fs` 子命令都支持 `-u/--user` 指定以哪个用户执行。相对路径和权限都按该用户解析，需要访问 root 拥有的文件时加 `-u root`：

```bash
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /root -u root
```

### 浏览文件

列出当前默认目录、指定目录或单个文件：

```bash
ucloud-sandbox-cli sandbox fs ls <sandbox-id>
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user/app/package.json
```

需要稳定解析结果时使用 JSON 格式：

```bash
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user/app --format json
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user/app -f json
```

`-d/--depth` 控制向下递归几层，默认 1（只列出目录本身的条目）：

```bash
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user/app -d 3 -f json
```

### 读取文件

把远端文件内容输出到标准输出：

```bash
ucloud-sandbox-cli sandbox fs cat <sandbox-id> /home/user/app/package.json
```

`fs cat` 不会自动脱敏。读取 `.env`、凭证、配置文件或其他可能包含密钥的文件前，先确认确有必要；不要把敏感内容直接回显给用户或写入日志。二进制文件和需要保存到本地的文件使用 `fs cp` 下载，不要使用 `fs cat`。

### 创建目录

```bash
ucloud-sandbox-cli sandbox fs mkdir <sandbox-id> /home/user/app
```

目录已存在时命令仍然成功，并输出 `Directory already exists`。`fs mkdir` 一次接收一个目录；创建多层目录时从已有父目录开始逐层执行。确需递归创建时，可以使用 `sandbox exec <sandbox-id> "mkdir -p <dir>"`，但必须先校验路径，避免拼接未经验证的用户输入。

### 上传和下载文件

`fs cp` 的用法是 `fs cp <src-path> <dest-path>`。远端端点必须写成 `<sandbox-id>:<path>`，源和目标中必须恰好有一个远端端点；不支持远端到远端复制，也不递归复制目录。

上传单个文件：

```bash
ucloud-sandbox-cli sandbox fs cp ./index.html <sandbox-id>:/home/user/app/index.html
```

上传到远端目录并保留本地文件名时，远端目标必须以 `/` 结尾：

```bash
ucloud-sandbox-cli sandbox fs cp ./index.html <sandbox-id>:/home/user/app/
```

下载文件：

```bash
ucloud-sandbox-cli sandbox fs cp <sandbox-id>:/home/user/app/output.txt ./output.txt
```

如果本地目标是已经存在的目录，CLI 会保留远端文件名：

```bash
ucloud-sandbox-cli sandbox fs cp <sandbox-id>:/home/user/app/output.txt ./downloads/
```

传输目录时，先用 `tar` 等工具打包成单个文件，上传或下载后再解包。打包部署内容时排除 `.git`、`.env`、API Key、依赖缓存和其他不应传输的敏感或冗余文件。

### 移动和重命名

在同一个沙箱内移动或重命名路径：

```bash
ucloud-sandbox-cli sandbox fs mv <sandbox-id> /home/user/app/old.txt /home/user/app/new.txt
```

`fs mv` 不能跨沙箱移动，也不能在本地与沙箱之间移动；这两类场景使用 `fs cp`，确认复制成功后再按用户意图决定是否删除源文件。

### 删除文件或目录

```bash
ucloud-sandbox-cli sandbox fs rm <sandbox-id> /home/user/app/obsolete.txt
```

`fs rm` 是破坏性操作且没有交互确认。执行前确认沙箱 ID 和准确路径，必要时先用 `fs ls` 检查目标；不要删除根目录、用户主目录、来源不明的目录或未经用户授权的数据，也不要把未经验证的用户输入直接作为删除路径。

## Snapshot 常用操作

从沙箱创建快照：

```bash
ucloud-sandbox-cli snapshot create <sandbox-id>
ucloud-sandbox-cli snap cr <sandbox-id>

# 指定名称；名称已存在时作为新的一次构建挂到该快照下，而不是新建一个
ucloud-sandbox-cli snapshot create <sandbox-id> -n nightly
```

列出快照：

```bash
ucloud-sandbox-cli snapshot list
ucloud-sandbox-cli snapshot list -s <sandbox-id>
ucloud-sandbox-cli snapshot list -n nightly
ucloud-sandbox-cli snapshot list --format json
```

删除快照。这会连同该快照的所有构建一起删除：

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
ucloud-sandbox-cli tpl init my-template --cpu-count 2 --memory-mb 1024
cd my-template
```

`--from` 不给时使用平台在当前地域的 base 镜像。生成的 `ucloud-template.json` 记录模板名、CPU、内存和 Dockerfile 文件名，`template build` 在命令行没给对应参数时从这里取值。

编辑生成的 `template.dockerfile`，加入模板构建需要的 `RUN`、`COPY` 等步骤。注意：当前 CLI 的 `template build` 命令以服务端模板构建流程为准；如果用户依赖复杂 Dockerfile 语法，先运行 `ucloud-sandbox-cli template build --help` 并按当前版本支持能力调整。

构建上下文就是 Dockerfile 所在目录，`COPY` 的源路径相对该目录解析。`--dockerfile` 指向该目录之外（包括子目录）时命令会直接报错，不要试图用绝对路径绕开。

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

base image 来自私有仓库时，先让用户在真实终端执行 `ucloud-sandbox-cli auth registry login <domain>` 配置凭据。`build` 会解析 `FROM` 的镜像，按其仓库域名自动选用对应凭据；命令行不再接收用户名和密码。

列出模板与查看构建日志：

```bash
ucloud-sandbox-cli template list
ucloud-sandbox-cli template list --format json
ucloud-sandbox-cli template get <template-id>
ucloud-sandbox-cli template logs <template-id> <build-id>
```

`template get` 只返回第一页构建记录，需要更多时用 `-l/--limit` 放大页大小。

管理指向构建的标签：

```bash
ucloud-sandbox-cli template tag list my-template
ucloud-sandbox-cli template tag assign my-template:latest v1
ucloud-sandbox-cli template tag remove my-template v1 -y
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

如果未登录，提示用户在真实终端执行 `ucloud-sandbox-cli auth login`，不要由 Agent 执行该交互命令。

创建沙箱并执行命令：

```bash
ucloud-sandbox-cli sandbox create base --detach
ucloud-sandbox-cli sandbox list --format json
ucloud-sandbox-cli sandbox exec <sandbox-id> "echo hello from sandbox"
ucloud-sandbox-cli sandbox kill <sandbox-id>
```

创建并挂载持久化 Volume：

```bash
ucloud-sandbox-cli volume create workspace
ucloud-sandbox-cli sandbox create base --mount workspace:/data --detach
ucloud-sandbox-cli sandbox exec <sandbox-id> "touch /data/example.txt && ls -la /data"
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
| `API key is required` | 提示用户在真实终端运行 `ucloud-sandbox-cli auth login`，或由用户自行设置 `UCLOUD_SANDBOX_API_KEY`；API Key 从星图平台 Key 管理获取 |
| 命令安装成功但找不到 | Linux/macOS 使用 `export PATH="$HOME/.local/bin:$PATH"`；Windows 参见 [Windows 故障处理](references/windows.md#故障处理) |
| 创建沙箱后卡在终端 | Agent/CI 中使用 `sandbox create ... --detach` |
| `template not found` | 运行 `template list --format json` 确认模板 ID/名称 |
| `sandbox not found` | 运行 `sandbox list --format json` 确认沙箱仍在运行 |
| 挂载 Volume 失败或 Volume 不存在 | 运行 `volume list --format json` 确认 Volume 名称；`--mount` 使用名称，不使用 Volume ID |
| metrics 时间参数不识别 | `--start`/`--end` 只接收 Unix 时间戳（秒），例如 `--start "$(date -d '1 hour ago' +%s)"` |
| `fs`、`login`、`region`、`config`、`clone` 提示未知命令 | 这些命令已经挪位：`sandbox fs`、`auth login`、`auth region`、`auth config`；`clone` 由 `sandbox fork` 取代 |
| 拉取私有 base image 失败 | 让用户在真实终端执行 `auth registry login <domain>`；用 `auth config` 确认 `registries` 里有该域名 |
| 命令卡住不返回 | 多半在等确认或在持续输出：带 `-y`，并避免 `sandbox logs -f`、`sandbox metrics -w` |

更多 CLI 用法参考：`https://astraflow.ucloud.cn/docs/agent-sandbox/product/cli`。
