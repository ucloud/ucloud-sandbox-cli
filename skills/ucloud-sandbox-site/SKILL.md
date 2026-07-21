---
name: ucloud-sandbox-site
description: 当用户提供以 `site_` 开头的 UCloud 站点空间 ID，并要求连接或操作站点空间时使用。适用于通过 ucloud-sandbox-cli 验证站点连接、执行命令、浏览与管理文件、上传或下载代码、读取站点环境变量，以及生成、构建、部署和排查运行在 80 端口的网站服务；同时遵守站点凭证的单沙箱权限边界和敏感变量脱敏要求。
---

# UCloud 站点空间

使用 `ucloud-sandbox-cli` 操作用户已经创建的站点空间。站点空间底层是一个沙箱；本技能只负责连接和维护该站点，不创建站点或沙箱。

## 核心概念

- 站点 ID 格式为 `site_<sandbox-id>`。例如 `site_iy1qen6gs2835o0udufdz` 对应沙箱 ID `iy1qen6gs2835o0udufdz`。
- 站点 ID 同时是一个受限 API Key。把完整站点 ID 写入 `UCLOUD_SANDBOX_API_KEY`，但向 CLI 传资源 ID 时使用去掉 `site_` 前缀后的沙箱 ID。
- 站点凭证只能操作它绑定的一个沙箱，不能列出或删除沙箱，也不能操作模板等其他资源。
- `/home/user/.site.env` 保存用户为网站配置的环境变量。它由站点空间管理，不要删除、覆盖或纳入部署包。
- 沙箱默认包含 Python 和 Node.js；确有需要时，可以通过 `sandbox exec` 调用 `apt` 安装其他依赖。
- 沙箱命令默认以 `user` 用户运行。`user` 已配置免密 sudo；绑定 80 端口以及其他需要 root 权限的操作要直接使用非交互的 `sudo -n`，不要先以普通用户试运行。

把站点 ID 视为凭证：不要在回复、日志、生成的代码或仓库文件中重复暴露它，也不要写入 `~/.ucloud-sandbox-cli/config.json`。仅在执行 CLI 的 shell 环境中临时设置。

## 权限和安全边界

站点操作只使用以下能力：

- `sandbox exec`：执行命令、检查环境、构建项目和管理服务。
- `sandbox host`：获取 80 端口的站点访问地址。
- `fs ls`、`fs cat`、`fs mkdir`、`fs cp`、`fs mv`、`fs rm`：管理该站点沙箱内的文件和目录。

不要尝试 `sandbox list`、`create`、`clone`、`kill`、`pause`，也不要执行快照、模板或其他管理命令。站点凭证返回无权限并不表示站点连接失败。

执行文件删除、覆盖或大范围移动前，先确认路径属于当前网站且操作符合用户意图。不要为了“清理部署目录”删除 `/home/user`、`/home/user/.site.env` 或来源不明的已有文件。

## 准备并验证 CLI

每次准备执行真实站点操作前，先检查是否存在旧 npm 版 CLI，并验证当前命令：

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

如果输出版本信息，CLI 验证通过，继续连接站点。如果输出 `OLD_NPM_CLI_FOUND`，说明安装的是 v1.0 及以前的 npm 旧版；先卸载旧版，再安装当前二进制版：

```bash
npm uninstall -g @ucloud-sdks/ucloud-sandbox-cli
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y
ucloud-sandbox-cli version
```

如果全局 npm 卸载需要管理员权限或交互确认，让用户在真实终端执行卸载命令，不要绕过权限限制。

如果输出 `NOT_INSTALLED`，使用官方安装脚本安装。Agent、CI 等自动化环境使用非交互模式：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y
ucloud-sandbox-cli version
```

需要让用户在真实终端交互安装时，也可以使用：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh
```

若默认安装目录不可写，安装到用户目录并把它加入当前 shell 的 `PATH`：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh -s -- -y -p "$HOME/.local/bin"
export PATH="$HOME/.local/bin:$PATH"
ucloud-sandbox-cli version
```

安装后必须以 `ucloud-sandbox-cli version` 成功作为 CLI 验证标准。安装脚本成功但命令仍不存在时，定位实际安装目录并加入 `PATH`；不要在尚未验证 CLI 时继续站点认证或部署。

本节只安装和验证 `ucloud-sandbox-cli`。不要自动更新已经可正常运行的 CLI，也不要在本技能中安装或更新 `ucloud-sandbox-site` skill 本身。

## 连接站点

### 1. 获取并校验站点 ID

如果用户还没有提供站点 ID，只向用户索取 `site_...` 格式的站点 ID；不要索取普通 UCloud API Key。先按照上一节完成 CLI 安装和验证。

在同一个 shell 调用中设置凭证并派生沙箱 ID：

```bash
SITE_ID='site_<sandbox-id>'

case "$SITE_ID" in
  site_?*) ;;
  *) echo "站点 ID 格式无效，应为 site_<sandbox-id>" >&2; exit 1 ;;
esac

SANDBOX_ID="${SITE_ID#site_}"
export UCLOUD_SANDBOX_API_KEY="$SITE_ID"
```

环境变量只对当前 shell 进程及其子进程有效。Agent 每次开启新的 shell 调用时，都要重新注入 `UCLOUD_SANDBOX_API_KEY` 并派生 `SANDBOX_ID`，不要假设上一次 `export` 仍然有效。

站点所在地域或 API 域名仍由已有 CLI 配置以及 `UCLOUD_SANDBOX_REGION`、`UCLOUD_SANDBOX_DOMAIN` 决定。没有证据时不要擅自切换；连接失败且怀疑地域不符时，向用户确认站点地域。

### 2. 执行无副作用的连接验证

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" "printf 'SITE_CONNECTED\\n'; pwd"
```

只要这个 `exec` 成功，就认为 AI 已连接站点，可以继续检查文件、生成代码和部署。不要用 `sandbox list` 验证连接。

## 常用命令

以下示例均假定当前 shell 已正确设置 `UCLOUD_SANDBOX_API_KEY` 和 `SANDBOX_ID`。`sandbox` 可以缩写为 `sbx`。

### 执行命令

命令内容必须作为一个完整字符串传给 `exec`：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" "pwd && ls -la /home/user"
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" "python3 --version && node --version && npm --version"
```

普通项目操作不要使用 sudo，包括上传或修改 `/home/user` 下的代码、安装项目依赖、执行构建和读取 `.site.env`。否则会产生 root 所有的项目文件，妨碍后续更新。

以下操作通常需要 root 权限，首次执行就使用 `sudo -n`：

- 监听 80 等小于 1024 的特权端口，以及停止由 root 启动的服务进程。
- 使用 `apt-get` 安装系统依赖。
- 使用 `systemctl`、`service` 或修改系统级服务配置。
- 写入 `/etc`、`/usr`、`/usr/local`、`/var` 等系统目录，或调整不属于 `user` 的文件权限和所有者。

不要在本地对整个 `ucloud-sandbox-cli` 命令使用 sudo；只在传给 `sandbox exec` 的远端命令中提升确实需要的部分。不要使用交互式 `sudo`，需要提权时先用 `sudo -n true` 验证免密权限；失败则报告问题，不要等待密码输入。

仅在项目确实需要额外系统包时安装：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" "sudo -n apt-get update && sudo -n apt-get install -y <package>"
```

### 浏览和读取文件

```bash
ucloud-sandbox-cli fs ls "$SANDBOX_ID" /home/user
ucloud-sandbox-cli fs ls "$SANDBOX_ID" /home/user --format json
ucloud-sandbox-cli fs cat "$SANDBOX_ID" /home/user/site/index.html
```

只对确认不含敏感信息的普通文件使用 `fs cat`。不要对 `/home/user/.site.env` 使用 `fs cat`。

### 创建目录

```bash
ucloud-sandbox-cli fs mkdir "$SANDBOX_ID" /home/user/site
```

目录已存在时命令仍然成功，并提示 `Directory already exists`。创建多层目录时，从已有的父目录开始逐层调用 `fs mkdir`；如果需要一次创建完整目录树，可以通过 `sandbox exec` 执行经过校验的 `mkdir -p`。

### 上传和下载文件

`fs cp` 的远端端点格式是 `<sandbox-id>:<path>`，源和目标中必须恰好有一个远端端点。它一次只复制一个文件：

```bash
# 上传
ucloud-sandbox-cli fs cp ./index.html "$SANDBOX_ID:/home/user/site/index.html"

# 下载
ucloud-sandbox-cli fs cp "$SANDBOX_ID:/home/user/site/service.log" ./service.log
```

上传目录时，先在本地打包，再上传并在站点中解压。排除 `.env`、凭证、依赖目录和其他不应部署的本地文件：

```bash
LOCAL_PROJECT_DIR='./site'
tar \
  --exclude='.git' \
  --exclude='.env' \
  --exclude='.env.*' \
  --exclude='node_modules' \
  -czf /tmp/site-release.tgz -C "$LOCAL_PROJECT_DIR" .

ucloud-sandbox-cli fs cp /tmp/site-release.tgz "$SANDBOX_ID:/tmp/site-release.tgz"
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "mkdir -p /home/user/site && tar -xzf /tmp/site-release.tgz -C /home/user/site && rm -f /tmp/site-release.tgz"
```

上传前先检查目标目录。若已有网站，优先使用独立发布目录或只覆盖本次变更的文件，避免旧文件与新构建产物混杂，也不要无条件清空目录。

### 移动和删除文件

```bash
ucloud-sandbox-cli fs mv "$SANDBOX_ID" /home/user/site/old.html /home/user/site/index.html
ucloud-sandbox-cli fs rm "$SANDBOX_ID" /home/user/site/obsolete.html
```

`fs rm` 是破坏性操作。执行前确认准确路径；需要删除目录或批量文件时，不要把未校验的用户输入拼入 `rm -rf`。

## 安全读取站点环境变量

用户询问已配置的环境变量时，可以告知变量是否存在，并展示非敏感变量。变量名只要不区分大小写地包含 `API_KEY` 或 `KEY`，就绝不能输出它的值；只说明该变量存在或显示 `<已隐藏>`。

不要直接运行以下可能泄露凭证的命令：

- `fs cat ... /home/user/.site.env`
- `cat /home/user/.site.env`
- 未过滤的 `env`、`printenv`、`set` 或 `export -p`
- 在加载环境变量时启用 `set -x`

需要查看配置摘要时，在沙箱内逐行解析并脱敏：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" 'awk '\''
  /^[[:space:]]*(#|$)/ { next }
  {
    line=$0
    sub(/^[[:space:]]*export[[:space:]]+/, "", line)
    eq=index(line, "=")
    if (!eq) next
    name=substr(line, 1, eq-1)
    gsub(/^[[:space:]]+|[[:space:]]+$/, "", name)
    if (name !~ /^[A-Za-z_][A-Za-z0-9_]*$/) next
    if (toupper(name) ~ /API_KEY|KEY/) print name "=<已隐藏>"
    else print line
  }
'\'' /home/user/.site.env'
```

如果只需确认某个敏感变量是否存在，验证变量名是合法 shell 标识符后，仅返回“存在”或“不存在”，不要返回值。若 `.site.env` 不存在，报告事实并询问用户，不要自行创建空文件替代它。

构建或启动用户网站时，通过下面的模式加载全部变量，不打印内容：

```bash
set -a
source /home/user/.site.env
set +a
```

## 生成和部署网站

根据用户需求选择合适的前端技术栈。先浏览现有文件和项目配置，再在本地生成或修改代码并上传；不要无故替换用户已有框架。部署时遵守以下硬性要求：

1. 服务必须监听 `0.0.0.0:80`，不能只监听 `127.0.0.1`，也不能改用 3000、5173、8080 等端口交付。沙箱默认用户是 `user`，绑定 80 端口时必须从第一次启动就使用 `sudo -n`。
2. 构建和启动服务时都要 `source /home/user/.site.env`；加载过程不得输出变量内容。
3. 服务必须脱离 `sandbox exec` 持久运行。使用 `nohup` 或站点中已有的服务管理器，并重定向标准输入、标准输出和标准错误。
4. 保存 PID 和日志。root 服务的存活检查与停止也要使用 `sudo -n kill`；重启时只终止自己记录的旧 PID，不使用宽泛的 `pkill node`、`killall` 等命令影响其他进程。
5. 只有在站点内访问 `http://127.0.0.1:80` 成功后，才调用 `sandbox host` 并向用户宣布部署成功。

### 构建示例

Node 项目按项目自己的锁文件和脚本构建；下面仅是常见模式：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "cd /home/user/site && set -a && source /home/user/.site.env && set +a && npm ci && npm run build"
```

如果没有 `package-lock.json`，按项目实际包管理器和锁文件选择命令，不要机械执行 `npm ci`。

不要用 sudo 执行 `npm`、`pnpm`、`yarn` 或构建脚本；只在启动最终的 80 端口服务时提权。

### 持久启动示例

对于构建产物位于 `/home/user/site/dist` 的纯静态网站，可以使用已安装的 Python 持久运行：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" '
  set -e
  PID_FILE=/home/user/site/service.pid
  LOG_FILE=/home/user/site/service.log

  sudo -n true

  if [ -s "$PID_FILE" ]; then
    old_pid=$(cat "$PID_FILE")
    if sudo -n kill -0 "$old_pid" 2>/dev/null; then
      sudo -n kill "$old_pid"
    fi
  fi

  sudo -n touch "$PID_FILE" "$LOG_FILE"
  sudo -n chown user:user "$PID_FILE" "$LOG_FILE"
  sudo -n bash -lc "
    set -e
    set -a
    source /home/user/.site.env
    set +a
    nohup python3 -m http.server 80 --bind 0.0.0.0 --directory /home/user/site/dist \
      >/home/user/site/service.log 2>&1 </dev/null &
    echo \$! >/home/user/site/service.pid
  "
'
```

启动前通过 sudo 创建日志和 PID 文件，再把所有者归还给 `user`，既能兼容先前试错留下的 root 文件，也能避免后续维护需要一直提权。必须在 root 启动器内部记录 `$!`，确保 PID 指向真正的网站服务，而不是外层 `sudo` 包装进程。

对于 SSR、Node 或其他动态服务，保留同样的 `sudo -n`、PID、日志、后台运行和环境变量加载模式，把 `exec python3 ...` 替换成项目的生产启动命令，并显式设置或传入 `HOST=0.0.0.0`、`PORT=80`。不要把开发服务器当作默认的生产部署方案。

### 验证并返回地址

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "curl -fsS --max-time 10 http://127.0.0.1:80/ >/dev/null && echo SITE_HTTP_OK"

ucloud-sandbox-cli sandbox host "$SANDBOX_ID" 80
```

把 `sandbox host` 输出的实际地址原样告知用户，不要猜测或拼接域名。同时简要说明已部署内容、监听端口和验证结果。若验证失败，先检查进程、80 端口和服务日志：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  'if [ -f /home/user/site/service.pid ]; then pid=$(cat /home/user/site/service.pid); sudo -n ps -p "$pid" -o pid=,stat=,cmd=; fi; sudo -n ss -ltnp | grep '\'':80'\'' || true; tail -n 100 /home/user/site/service.log'
```

上例用单引号包住远端命令，使 `$()` 在沙箱中展开。改写命令或改变引号层级时，不要让本地 shell 提前执行远端表达式。

## 故障处理

| 现象 | 处理 |
| --- | --- |
| `exec` 提示无权限 | 确认完整 `site_...` 被用作 API Key，去掉前缀的值被用作沙箱 ID；不要改用 `sandbox list` 测试 |
| 提示找不到沙箱 | 检查是否误把完整站点 ID 当成沙箱 ID，并向用户确认站点地域 |
| `.site.env` 不存在 | 报告缺失并询问用户；不要擅自创建或用本地 `.env` 覆盖 |
| 启动 80 端口时报 `Permission denied` | 确认启动命令从第一次执行就使用 `sudo -n`，不要先用普通用户尝试绑定 80 端口 |
| `sudo` 等待密码或提示需要终端 | 改用 `sudo -n`；若 `sudo -n true` 失败，报告免密 sudo 配置异常，不要尝试输入密码 |
| `exec` 在启动服务后不返回 | 确认服务已后台运行，并把 stdin、stdout、stderr 全部重定向 |
| 站点内访问 80 端口失败 | 检查 PID、日志、启动命令和监听地址，确认服务监听 `0.0.0.0:80` |
| `sandbox host` 有输出但页面打不开 | 不要据此宣布成功；先在站点内用 `curl` 验证，再检查进程和日志 |
