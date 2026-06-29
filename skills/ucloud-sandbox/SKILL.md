---
name: ucloud-sandbox
description: 当用户需要用 UCloud Sandbox CLI 操作沙箱服务时使用，包括安装或配置 ucloud-sandbox-cli、设置 API Key 和地域、创建/连接/执行/暂停/终止沙箱、查看沙箱列表/端口地址/监控指标、创建/列出/删除快照、初始化/构建/发布/删除模板，以及在 Claude Code、Codex、Gemini 等 Agent 中安装本技能。
---

# UCloud Sandbox CLI

使用 `ucloud-sandbox-cli` 管理 UCloud Sandbox 沙箱、快照和模板。优先用 CLI 完成操作；如果用户只是在询问命令，给出可复制的命令即可。

## 安装本技能

仅在用户要求“安装这个 skill/技能”时执行。把 `SKILL.md` 放到目标 Agent 的技能目录。可设置 `TARGET_AGENT=codex|claude|gemini|auto`，默认自动检测：

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

常见目录：

- Codex: `TARGET_AGENT=codex`，目录为 `${CODEX_HOME:-$HOME/.codex}/skills/ucloud-sandbox`
- Claude Code: `TARGET_AGENT=claude`，目录为 `$HOME/.claude/skills/ucloud-sandbox`
- Gemini CLI: `TARGET_AGENT=gemini`，目录为 `$HOME/.gemini/skills/ucloud-sandbox`

## Step 0：确保 CLI 可用

每次准备执行真实操作前先检查：

```bash
if ! command -v ucloud-sandbox-cli >/dev/null 2>&1; then
  echo "NOT_INSTALLED"
else
  ucloud-sandbox-cli version
fi
```

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

更新本技能：

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

更新 `ucloud-sandbox-cli` 到最新版本：

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

优先使用环境变量，适合临时会话和 CI：

```bash
export UCLOUD_SANDBOX_API_KEY="<api-key>"
export UCLOUD_SANDBOX_REGION="cn-wlcb"
```

持久化配置优先使用交互命令：

```bash
ucloud-sandbox-cli login
```

如果用户明确提供 API Key 和地域，并要求 Agent 直接生成配置，可写入 `~/.ucloud-sandbox-cli/config.json`。把下面的 `<api-key>` 和地域替换成用户提供值；不要打印 API Key，不要提交到 git。

```bash
mkdir -p "$HOME/.ucloud-sandbox-cli"
chmod 700 "$HOME/.ucloud-sandbox-cli"
cat > "$HOME/.ucloud-sandbox-cli/config.json" <<'EOF'
{
  "api_key": "<api-key>",
  "region": "cn-wlcb"
}
EOF
chmod 600 "$HOME/.ucloud-sandbox-cli/config.json"
```

环境变量优先级高于配置文件。切换地域：

```bash
ucloud-sandbox-cli region
```

退出登录并删除本地凭据：

```bash
ucloud-sandbox-cli logout
```

## Agent 操作原则

- 在 Claude Code、Codex、Gemini、CI 等非 TTY 环境中，创建或克隆沙箱时默认加 `--detach`，否则 CLI 会尝试进入交互终端。
- 需要解析列表时优先用 `--format json` 或 `-f json`。
- 执行破坏性命令前先确认用户意图：`sandbox kill`、`sandbox kill --all`、`snapshot delete`、`template delete`、`template publish --unpublish`。
- 不要在回复、日志或命令输出中泄露 API Key。
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
ucloud-sandbox-cli login
```

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
| `API key is required` | 运行 `ucloud-sandbox-cli login`，或设置 `UCLOUD_SANDBOX_API_KEY` |
| 命令安装成功但找不到 | 把安装目录加入 `PATH`，例如 `export PATH="$HOME/.local/bin:$PATH"` |
| 创建沙箱后卡在终端 | Agent/CI 中使用 `sandbox create ... --detach` |
| `template not found` | 运行 `template list --format json` 确认模板 ID/名称 |
| `sandbox not found` | 运行 `sandbox list --format json` 确认沙箱仍在运行 |
| metrics 时间参数不识别 | 使用 `--since 1h`、`--start "2026-06-23 12:00"` 这类格式 |

更多 CLI 用法参考：`https://astraflow.ucloud.cn/docs/agent-sandbox/product/cli`。
