---
name: ucloud-sandbox-site
description: 当用户提供以 `site_` 开头的 UCloud 站点空间连接 Key（格式为 `site_<sandbox-id>_<code>`），并要求连接、验证或操作站点空间时使用。适用于在 Linux、macOS 或 Windows 中通过 ucloud-sandbox-cli 校验连接 Key 并派生沙箱 ID、识别旧版 `site_<sandbox-id>` 格式并引导用户到星图站点空间页面重新获取连接语句、验证站点连接、识别访问码保护、执行命令、浏览与管理文件、上传或下载代码、读取站点环境变量，以及生成、构建、部署和排查运行在 80 端口的网站服务；同时遵守站点连接 Key、站点访问码等敏感信息的单沙箱权限边界和脱敏要求。
---

# UCloud 站点空间

使用 `ucloud-sandbox-cli` 操作用户已经创建的站点空间。站点空间底层是一个沙箱；本技能只负责连接和维护该站点，不创建站点或沙箱。

## 前置检查和平台

执行 CLI 安装或调用站点 API 前，先确认当前环境允许访问公网。需要网络权限审批时先申请授权；未获授权时停止并说明原因。

在 Windows 或 PowerShell 环境中，执行安装、连接、文件传输或部署前，先完整阅读并遵循 [Windows PowerShell 指南](references/windows.md)。本页的 Bash 安装、凭证注入和本地打包命令仅适用于 Linux 和 macOS；传给 `sandbox exec` 的命令仍在远端 Linux Shell 中执行。

## 核心概念

- 站点连接 Key 格式为 `site_<sandbox-id>_<code>`，`<code>` 是站点空间随机生成的连接码，用于防止他人仅凭沙箱 ID 就连接到站点。例如 `site_iy1qen6gs2835o0udufdz_7f3a9c2e` 对应沙箱 ID `iy1qen6gs2835o0udufdz`。
- 沙箱 ID 是 `site_` 前缀之后、第一个 `_` 之前的部分。连接码只是 Key 的组成部分，不要单独传给 CLI，也不要拼进沙箱 ID。
- 站点连接 Key 同时是一个受限 API Key。把完整 Key 写入 `UCLOUD_SANDBOX_API_KEY`，但向 CLI 传资源 ID 时只使用派生出的沙箱 ID。
- 旧格式 `site_<sandbox-id>`（不带 `_<code>`）已不再可用。用户提供旧格式时，不要尝试补全、猜测或省略连接码，也不要改用普通 API Key，直接请用户到星图控制台的[站点空间页面](https://astraflow.ucloud.cn/docs/modelverse/console/station-site)重新复制新的站点连接语句。
- 站点凭证只能操作它绑定的一个沙箱，不能列出或删除沙箱，也不能操作模板等其他资源。
- `/home/user/.site.env` 保存用户为网站配置的环境变量。它由站点空间管理，不要删除、覆盖或纳入部署包。
- 沙箱默认包含 Python 和 Node.js；确有需要时，可以通过 `sandbox exec` 调用 `apt` 安装其他依赖。
- 沙箱命令默认以 `user` 用户运行。`user` 已配置免密 sudo；绑定 80 端口以及其他需要 root 权限的操作要直接使用非交互的 `sudo -n`，不要先以普通用户试运行。

把站点连接 Key 视为凭证：不要在回复、日志、生成的代码或仓库文件中重复暴露它（包括其中的连接码），也不要写入 `~/.ucloud-sandbox-cli/config.json`。仅在执行 CLI 的 shell 环境中临时设置。派生出的沙箱 ID 不含连接码，可以正常出现在命令中。

## 权限和安全边界

站点操作只使用以下能力：

- `sandbox exec`：执行命令、检查环境、构建项目和管理服务。
- `sandbox host`：获取 80 端口的站点访问地址。
- `sandbox fs ls`、`sandbox fs cat`、`sandbox fs mkdir`、`sandbox fs cp`、`sandbox fs mv`、`sandbox fs rm`：管理该站点沙箱内的文件和目录。本节以下把这一组简称为 `fs ...`。

不要尝试 `sandbox list`、`create`、`fork`、`kill`、`pause`，也不要执行快照、Secret、Volume、模板或其他管理命令。站点凭证返回无权限并不表示站点连接失败。

执行文件删除、覆盖或大范围移动前，先确认路径属于当前网站且操作符合用户意图。不要为了“清理部署目录”删除 `/home/user`、`/home/user/.site.env` 或来源不明的已有文件。

## 准备并验证 CLI

最低支持版本是 `v1.3.2`。`v1.3.1` 开始让控制面、RPC、文件和流式请求统一继承 `HTTP_PROXY`、`HTTPS_PROXY` 与 `NO_PROXY`；`v1.3.2` 进一步要求官方安装器校验 Release SHA256。更早版本在受限网络中可能误报 DNS 失败，或缺少安装完整性校验。

先执行一次 `ucloud-sandbox-cli version`。AstraFlow 的本地 Agent `PATH` 包含真实用户的通用 CLI 目录（Linux/macOS 为 `~/.local/bin`），不是每个会话的私有 `$HOME`，也不是 AstraFlow 的 `Application Support` 目录。已满足最低版本时直接继续，不要更新。

如果命令不存在或低于 `v1.3.2`，只把官方安装命令安装到真实用户的 `~/.local/bin`。这一步必须作为一个完整命令申请一次主机执行权限；获批后不要再为安装器内部的下载、临时目录或写文件分别申请权限：

```bash
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/ucloud-sandbox-cli.XXXXXXXX")" && \
trap 'rm -rf "$TMP_DIR"' EXIT && \
curl -fsSLo "$TMP_DIR/install.sh" \
  https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh && \
sh "$TMP_DIR/install.sh" -y -p "$HOME/.local/bin" && \
"$HOME/.local/bin/ucloud-sandbox-cli" version
```

不要先尝试 `/usr/local/bin`，不要使用 sudo，不要安装到 Agent 会话的私有 `$HOME`，不要写入或硬编码 AstraFlow 的 `sandbox-workspaces` / `Application Support` 路径，也不要用 `find $HOME` 搜索 CLI。安装完成后回到 Default 沙箱，按命令名 `ucloud-sandbox-cli` 使用；真实用户目录跨会话和重启复用。

Windows 按 [Windows PowerShell 指南](references/windows.md) 使用用户级安装目录；同样把安装与版本验证合并成一次获批的主机命令。

仅当用户明确要求更新 CLI 时，如果已安装的版本是 `v1.3.4` 或更高，直接用自带的 `update` 命令，不必重新下载安装脚本。这里没有交互式终端，必须带 `-y` 跳过确认：

```bash
ucloud-sandbox-cli update -y && ucloud-sandbox-cli version
```

`update` 会校验 Release SHA256 后替换当前程序，保持原安装目录不变。低于 `v1.3.4` 的版本没有该命令，按上面的安装流程重装。查询版本报 GitHub API 限流时，说明原因并让用户稍后重试或设置 `GITHUB_TOKEN`。

本节只准备 `ucloud-sandbox-cli`，不要安装或更新本 Skill 本身。

### 判断 Skill 是否最新

当需要确认 Skill 是否为最新版本时，比较已安装 CLI 版本与 GitHub 最新 release；两者一致（或 Skill 相关文件无落后提交）即视为 Skill 已是最新：

```bash
ucloud-sandbox-cli version
curl -fsSL https://api.github.com/repos/ucloud/ucloud-sandbox-cli/releases/latest | grep tag_name
```
如果不是最新的通过`npx skills add ucloud/ucloud-sandbox-cli -s astraflow-api -s ucloud-sandbox-site`提示用户进行更新skill

## 连接站点

### 1. 获取并校验站点连接 Key

如果用户还没有提供站点连接 Key，只向用户索取 `site_<sandbox-id>_<code>` 格式的连接 Key（即站点空间页面上的站点连接语句）；不要索取普通 UCloud API Key，也不要只要沙箱 ID。先按照上一节完成 CLI 安装和验证。

Linux 和 macOS 在同一个 shell 调用中校验 Key、设置凭证并派生沙箱 ID：

```bash
SITE_KEY='site_<sandbox-id>_<code>'

case "$SITE_KEY" in
  site_*_*) ;;
  site_?*)
    echo "检测到旧版站点 ID 格式 site_<sandbox-id>，已不再支持。请到星图控制台的站点空间页面重新获取站点连接语句（site_<sandbox-id>_<code>）。" >&2
    exit 1 ;;
  *)
    echo "站点连接 Key 格式无效，应为 site_<sandbox-id>_<code>" >&2
    exit 1 ;;
esac

SITE_BODY="${SITE_KEY#site_}"
SANDBOX_ID="${SITE_BODY%%_*}"
SITE_CODE="${SITE_BODY#*_}"

if [ -z "$SANDBOX_ID" ] || [ -z "$SITE_CODE" ]; then
  echo "站点连接 Key 缺少沙箱 ID 或连接码，请到星图控制台的站点空间页面重新获取站点连接语句。" >&2
  exit 1
fi

export UCLOUD_SANDBOX_API_KEY="$SITE_KEY"
```

`SANDBOX_ID` 只取第一个 `_` 之前的部分，连接码留在 `UCLOUD_SANDBOX_API_KEY` 中；不要把 `SITE_CODE` 或完整 Key 传给任何 CLI 子命令的资源 ID 参数。

校验失败时立即停止，不要继续尝试 `sandbox exec`。用户提供旧格式时，只回复需要到站点空间页面重新获取连接语句，不要输出用户给出的旧 ID 之外的推测值。

环境变量只对当前 shell 进程及其子进程有效。Agent 每次开启新的 shell 调用时，都要重新注入 `UCLOUD_SANDBOX_API_KEY` 并派生 `SANDBOX_ID`，不要假设上一次 `export` 仍然有效。

站点所在地域或 API 域名仍由已有 CLI 配置以及 `UCLOUD_SANDBOX_REGION`、`UCLOUD_SANDBOX_DOMAIN` 决定。没有证据时不要擅自切换；连接失败且怀疑地域不符时，向用户确认站点地域。

### 2. 在一次调用中验证连接并检查初始目录

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "printf 'SITE_CONNECTED\\n'; pwd; printf 'SITE_HOME\\n'; ls -la /home/user"
```

只要这个 `exec` 成功，就认为 AI 已连接站点并已完成首次目录检查，可以继续生成代码和部署。不要把连接验证、`pwd` 和初始 `ls` 拆成多个调用，也不要用 `sandbox list` 验证连接。

### 3. 识别站点访问码保护

CLI 连接验证与公共 URL 访问是两件事：AI 可以通过 CLI 连接沙箱，但配置了访问码的公共站点不能被 AI 直接验证。

访问 `sandbox host` 或用户自定义域名时，使用不携带任何访问码的请求检查 HTTP 状态和响应体。若返回 `401` 且响应体包含 `Site protected`，判定为站点已开启访问码保护，而不是服务故障。不要猜测、绕过或重试访问码，也不要要求客户把访问码发送给 AI；访问码是敏感内容，不得写入命令、日志、代码或回复。

此时可以保留站点内 `http://127.0.0.1:80` 的验证结果，但公共页面验证必须交给客户：把 `sandbox host` 返回的地址交给客户，让客户在浏览器中自行输入访问码。向客户说明 AI 无法在访问码开启时完成公共页面验证。

如果客户确实需要 AI 在开发阶段验证公共页面，请客户先在站点空间控制台临时关闭访问码，完成验证后再重新开启。配置方法参考[站点空间文档](https://astraflow.ucloud.cn/docs/modelverse/console/station-site)。即使客户主动提供访问码，也不要让 AI 代为使用或保存访问码。

### 4. 识别公共访问被拒绝（403）

公共 URL 返回 `403` 表示站点网关拒绝了这次访问，不是网站服务故障，也不是 CLI 连接失败。判定前先读取响应体，按下面三种情况区分处理，不要重试、绕过或更换出口 IP：

| 响应体特征 | 原因 | 处理 |
| --- | --- | --- |
| 包含 `This site does not accept requests from your IP address` | 访问来源 IP 不被允许 | 请客户到星图控制台的站点空间页面检查该站点的 IP 白名单和黑名单配置，确认当前访问来源是否被放行 |
| 包含 `Domain not allowed` | 站点配置了自定义域名，只允许通过该域名访问 | CLI 无法获取站点配置的自定义域名，AI 不要猜测或拼接域名；把公共页面验证交给客户，请客户用自己在星图控制台配置的自定义域名访问 |
| 包含 `Port not allowed` | 访问了 80 以外的端口 | 站点只开放 80 端口。改回 80 端口访问，并确认服务监听 `0.0.0.0:80`，不要通过其他端口对外交付 |

IP 名单和自定义域名都属于站点侧配置，AI 无法通过 CLI 查询或修改，只能引导客户在星图控制台调整或改用自定义域名访问。此时可以保留站点内 `http://127.0.0.1:80` 的验证结果，但不要宣布公共页面验证通过。

### 5. 返回统一连接提示

完成连接验证（成功或失败）后，按本节的固定模板向用户返回提示。不要省略字段、改动措辞或用其他格式替代；模板中的 `<sandbox-id>`、`<region>` 用实际值替换。

连接成功（第 2 步的 `exec` 验证通过）时返回：

```
已成功连接站点，可以开始开发。
- 沙箱 ID：<sandbox-id>
- 地域：<region>
- 工作目录：/home/user
- 连接技能：ucloud-sandbox-site（<技能状态>）

使用指引
- 开发站点：用自然语言描述你的想法，Agent 会帮你开发。
- 保存数据：站点已预装 PostgreSQL，默认未启动。需要时告诉 Agent：“使用预装数据库保存数据。”
- 查看效果：开发完成后，回到「站点空间」，点击「访问」即可查看，无需单独发布。
```

连接失败（连接 Key 校验通过但 `exec` 验证失败）时返回：

```
连接失败，暂时无法访问站点。
- 沙箱 ID：<sandbox-id>
- 地域：<region>
- 失败原因：<根据实际返回信息填写>
- 连接技能：ucloud-sandbox-site（<技能状态>）

排查指引
- 更新技能：旧版技能可能导致连接失败，请重新安装最新版后重试。
- 核对连接信息：回到「站点空间」，重新复制完整连接语句后交给 Agent。
- 仍无法连接：将以上信息及错误提示提供给技术支持，协助排查。
```

各字段来源：

- 沙箱 ID：从连接 Key 派生的沙箱 ID，不含连接码。
- 地域：当前 CLI 配置或 `UCLOUD_SANDBOX_REGION` 中的地域，可通过 `ucloud-sandbox-cli auth config` 确认。
- 工作目录：第 2 步验证命令中 `pwd` 的输出，正常为 `/home/user`。
- 失败原因：摘自 CLI 的实际错误输出，可以概括但不得虚构，也不要包含完整连接 Key 或连接码。
- 连接技能状态三选一：会话或平台已确认技能为最新版时用“已是最新”；已确认技能不是最新时用“请重新安装最新版，否则可能影响新功能使用”（成功模板）或“请重新安装最新版，否则可能影响连接及新功能使用”（失败模板）；无法完成检查时用“未能检查更新，请确认是否为最新版”。不要编造检查结果。

连接 Key 为旧格式或校验失败时不套用失败模板，按“获取并校验站点连接 Key”的要求回复。连接成功后继续执行站点操作时，不需要重复返回成功模板。

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

### PostgreSQL 15 与 pgvector（仅适用于 site 模板）

基于 `site` 系列模板（规格模板命名格式为 `site-{CPU核数}c-{内存GiB数}g`）构建的站点内置 PostgreSQL 15、客户端和 pgvector。先检查实际环境，不要仅凭站点连接 Key 判断数据库已安装或已启动：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "whoami && command -v start-postgres && command -v stop-postgres && psql --version"
```

默认 Linux 用户应为 `user`。`base` 模板不包含这些数据库组件；若命令缺失，先确认站点模板和实际安装情况，不要用站点凭证创建或替换沙箱。

#### 启动、验证和停止

- `start-postgres` 内部执行 `sudo -n systemctl start postgresql-vector.service`，由其依赖启动 `postgresql@15-main.service`，调用者无需再包一层 sudo。
- `postgresql-vector.service` 是以 `postgres` 运行的 oneshot 初始化单元：通过 `enable-pgvector` 按需创建非超级用户角色 `user`、由该角色拥有的 `ucloud` 数据库，并在 `ucloud` 中执行 `CREATE EXTENSION IF NOT EXISTS vector`。
- 初始化单元使用 `RemainAfterExit=yes`，显示 `active (exited)` 正常；重复启动已活动的单元不会重新执行初始化 SQL，也不会修正已有角色或数据库的属性。

在同一沙箱中启动数据库：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" "start-postgres"
```

通过 Unix Socket 检查服务、连接身份和向量类型。`pg_isready` 只能检查服务就绪，后面的 SQL 成功才证明应用身份可以连接并使用 pgvector：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "sudo -n systemctl is-active postgresql@15-main.service && \
   sudo -n systemctl is-active postgresql-vector.service && \
   pg_isready -h /var/run/postgresql -p 5432 -U user -d ucloud && \
   psql -X -w -v ON_ERROR_STOP=1 -h /var/run/postgresql -p 5432 -U user -d ucloud \
     -c \"SELECT current_user, current_database(), '[1,2,3]'::vector;\""
```

预期查询返回 `user`、`ucloud` 和 `[1,2,3]`。需要停止数据库时执行：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" "stop-postgres"
```

`stop-postgres` 内部停止 `postgresql@15-main.service`，初始化单元随依赖关系一起停止；它不会删除数据库。不要把停止数据库作为连接验证的收尾步骤，以免中断网站。

#### 应用连接配置

模板默认使用本地 Unix Socket 目录 `/var/run/postgresql`、端口 `5432`、数据库 `ucloud` 和角色 `user`。`pg_hba.conf` 对本地 Socket 使用 `peer` 认证，对 IPv4、IPv6 的所有 TCP 连接（含回环地址）使用 `reject`；模板同时关闭了 PostgreSQL SSL。

- 应用和迁移命令必须在同一沙箱内以 Linux 用户 `user` 运行。peer 核对操作系统身份，仅在配置中填写数据库用户名 `user` 并不能让 root 进程通过认证。
- 显式指定 Socket 目录，不使用 `127.0.0.1` 或 `localhost`。不要假定省略 `host` 就会使用 Socket，例如 Node.js 的 `pg` 默认连接 `localhost`。
- 指定 Socket 目录后，`port=5432` 选择的是 `.s.PGSQL.5432` Socket 文件，并不代表走 TCP。此连接不需要数据库密码，也不要要求 SSL 或放开 `pg_hba.conf` 来修复连接。
- 不要使用超级用户 `postgres` 运行业务应用；必要的管理员操作使用 `sudo -n -u postgres psql`。

应用连接示例：

| 客户端 | 连接示例 |
| --- | --- |
| Python / psycopg | `psycopg.connect("host=/var/run/postgresql port=5432 dbname=ucloud user=user")` |
| Node.js / pg | `new Pool({ host: '/var/run/postgresql', port: 5432, database: 'ucloud', user: 'user' })` |
| Go / pgx 或 lib/pq | `host=/var/run/postgresql port=5432 dbname=ucloud user=user sslmode=disable` |

#### 同时满足 peer 认证和 80 端口要求

使用数据库的动态网站不能套用后文的 root 静态服务启动示例。可通过 systemd 的 `User=user` 与 `AmbientCapabilities=CAP_NET_BIND_SERVICE` 让应用以 `user` 身份绑定 `0.0.0.0:80`，使用 `sudo -n` 管理服务。此时进程、PID 和日志由 systemd 管理，无需另存 PID 文件；环境变量加载和 HTTP 验证仍按部署章节执行。

#### 数据库排错

| 现象 | 处理 |
| --- | --- |
| Socket 不存在或连接被拒绝 | 检查 `start-postgres` 结果、`systemctl status postgresql@15-main.service postgresql-vector.service` 和 `/var/run/postgresql`；需要日志时使用 `sudo -n journalctl -u postgresql@15-main.service -u postgresql-vector.service -n 100 --no-pager`，输出前检查是否包含敏感内容 |
| `pg_hba.conf rejects connection`，或日志出现 `127.0.0.1` / `::1` | 客户端走了 TCP；显式把 `host` 改成 Socket 目录，并检查环境变量或连接 URL 是否覆盖配置 |
| `Peer authentication failed` 或 `role "root" does not exist` | 检查实际应用或迁移进程的 Linux 用户与数据库角色，改为 `user`；不要通过 `trust`、密码或放开 TCP 绕过 peer |
| 缺少 `ucloud`、`user` 或 `vector` 类型 | 检查初始化单元状态和日志，并确认连接的是 `ucloud`；初始化脚本只在该数据库中启用 `vector` |

### 浏览和读取文件

```bash
ucloud-sandbox-cli sandbox fs ls "$SANDBOX_ID" /home/user
ucloud-sandbox-cli sandbox fs ls "$SANDBOX_ID" /home/user --format json
ucloud-sandbox-cli sandbox fs cat "$SANDBOX_ID" /home/user/site/index.html
```

只对确认不含敏感信息的普通文件使用 `fs cat`。不要对 `/home/user/.site.env` 使用 `fs cat`。

### 创建目录

```bash
ucloud-sandbox-cli sandbox fs mkdir "$SANDBOX_ID" /home/user/site
```

目录已存在时命令仍然成功，并提示 `Directory already exists`。创建多层目录时，从已有的父目录开始逐层调用 `fs mkdir`；如果需要一次创建完整目录树，可以通过 `sandbox exec` 执行经过校验的 `mkdir -p`。

### 上传和下载文件

`fs cp` 的远端端点格式是 `<sandbox-id>:<path>`，源和目标中必须恰好有一个远端端点。它一次只复制一个文件：

```bash
# 上传
ucloud-sandbox-cli sandbox fs cp ./index.html "$SANDBOX_ID:/home/user/site/index.html"

# 下载
ucloud-sandbox-cli sandbox fs cp "$SANDBOX_ID:/home/user/site/service.log" ./service.log
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

ucloud-sandbox-cli sandbox fs cp /tmp/site-release.tgz "$SANDBOX_ID:/tmp/site-release.tgz"
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "mkdir -p /home/user/site && tar -xzf /tmp/site-release.tgz -C /home/user/site && rm -f /tmp/site-release.tgz"
```

上传前先检查目标目录。若已有网站，优先使用独立发布目录或只覆盖本次变更的文件，避免旧文件与新构建产物混杂，也不要无条件清空目录。

### 移动和删除文件

```bash
ucloud-sandbox-cli sandbox fs mv "$SANDBOX_ID" /home/user/site/old.html /home/user/site/index.html
ucloud-sandbox-cli sandbox fs rm "$SANDBOX_ID" /home/user/site/obsolete.html
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

对于不使用本地 PostgreSQL 的 SSR、Node 或其他动态服务，保留同样的 `sudo -n`、PID、日志、后台运行和环境变量加载模式，把 `exec python3 ...` 替换成项目的生产启动命令，并显式设置或传入 `HOST=0.0.0.0`、`PORT=80`。不要把开发服务器当作默认的生产部署方案。

### 验证并返回地址

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  "curl -fsS --max-time 10 http://127.0.0.1:80/ >/dev/null && echo SITE_HTTP_OK"

ucloud-sandbox-cli sandbox host "$SANDBOX_ID" 80
```

把 `sandbox host` 输出的实际地址原样告知用户，不要猜测或拼接域名。同时简要说明已部署内容、监听端口和验证结果。

验证公共 URL 时同时读取 HTTP 状态码和响应体，先区分网关拒绝和服务故障，再决定是否排查服务：

- `401` 且响应体包含 `Site protected`：访问码保护，按“识别站点访问码保护”处理。
- `403`：访问被网关拒绝，按“识别公共访问被拒绝（403）”对照响应体判断是 IP 名单、自定义域名还是端口限制。

这两类情况都不表示网站服务异常，也不影响站点内 HTTP 验证结果，但都不能据此宣布公共页面验证成功。若验证失败且不属于上述两类，先检查进程、80 端口和服务日志：

```bash
ucloud-sandbox-cli sandbox exec "$SANDBOX_ID" \
  'if [ -f /home/user/site/service.pid ]; then pid=$(cat /home/user/site/service.pid); sudo -n ps -p "$pid" -o pid=,stat=,cmd=; fi; sudo -n ss -ltnp | grep '\'':80'\'' || true; tail -n 100 /home/user/site/service.log'
```

上例用单引号包住远端命令，使 `$()` 在沙箱中展开。改写命令或改变引号层级时，不要让本地 shell 提前执行远端表达式。

## 故障处理

| 现象 | 处理 |
| --- | --- |
| 用户提供的是旧格式 `site_<sandbox-id>` | 不要尝试连接或补全连接码；告知格式已更新，请用户到星图控制台的站点空间页面重新复制站点连接语句 |
| `exec` 提示无权限或鉴权失败 | 确认完整 `site_<sandbox-id>_<code>` 被用作 API Key（连接码没有被截断），派生出的沙箱 ID 被用作资源 ID；仍失败时请用户确认连接语句是否已在控制台重新生成；不要改用 `sandbox list` 测试 |
| 提示找不到沙箱 | 检查是否误把完整连接 Key 或带连接码的字符串当成沙箱 ID，沙箱 ID 只取第一个 `_` 之前的部分，并向用户确认站点地域 |
| `.site.env` 不存在 | 报告缺失并询问用户；不要擅自创建或用本地 `.env` 覆盖 |
| 启动 80 端口时报 `Permission denied` | 确认启动命令从第一次执行就使用 `sudo -n`，不要先用普通用户尝试绑定 80 端口 |
| `sudo` 等待密码或提示需要终端 | 改用 `sudo -n`；若 `sudo -n true` 失败，报告免密 sudo 配置异常，不要尝试输入密码 |
| `exec` 在启动服务后不返回 | 确认服务已后台运行，并把 stdin、stdout、stderr 全部重定向 |
| 站点内访问 80 端口失败 | 检查 PID、日志、启动命令和监听地址，确认服务监听 `0.0.0.0:80` |
| 公共 URL 返回 `401` 且响应体包含 `Site protected` | 这是访问码保护，不是服务故障；不要索取或使用访问码，交给客户自行输入验证；如需 AI 验证，先让客户临时关闭访问码 |
| 公共 URL 返回 `403` 且响应体包含 `This site does not accept requests from your IP address` | 访问来源 IP 被拒绝；请客户在星图控制台的站点空间页面检查该站点的 IP 白名单和黑名单配置，不要重试或更换出口 IP |
| 公共 URL 返回 `403` 且响应体包含 `Domain not allowed` | 站点配置了自定义域名，只能通过该域名访问；CLI 无法获取该域名，不要猜测或拼接，请客户用自己配置的自定义域名访问并验证 |
| 公共 URL 返回 `403` 且响应体包含 `Port not allowed` | 站点只开放 80 端口；改回 80 端口访问，并确认服务监听 `0.0.0.0:80` |
| `sandbox host` 有输出但页面打不开 | 不要据此宣布成功；先在站点内用 `curl` 验证，再检查进程和日志 |
| CLI 低于 `v1.3.2` 且出现 DNS/代理错误 | 按“准备并验证 CLI”只申请一次主机安装命令，将 CLI 升级到真实用户目录；回到 Default 沙箱重试一次，不要切到主机执行站点命令 |
| CLI 已是 `v1.3.2` 或更高但仍出现 DNS/代理错误 | 保持 Default 沙箱；如平台要求，只申请一次当前会话的网络授权并重试一次。仍失败则报告域名、端口和原始错误，不要继续申请主机执行权限 |
