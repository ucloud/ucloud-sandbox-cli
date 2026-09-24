# UCloud Sandbox CLI

强大的命令行工具，用于本地管理沙箱生命周期、构建模板及执行运维任务。

UCloud Sandbox CLI 是开发者最常用的工具之一。它不仅可以帮助您快速初始化和构建沙箱模板，还可以实时查看沙箱监控指标以及执行批量操作。

## 安装指南

### 卸载旧的基于npm的CLI（可选）

在`v1.0`及以前的版本中，我们的CLI是基于npm构建和分发的，在后续版本中，我们改为了手动执行安装脚本。

如果您安装过`v1.0.x`的`ucloud-sandbox-cli`，请先使用下面的命令进行卸载：

```bash
npm uninstall -g @ucloud-sdks/ucloud-sandbox-cli
```

如果您之前从未安装过`ucloud-sandbox-cli`，或者没有使用npm安装过，可以跳过这一步。

### 安装 CLI

Linux 和 macOS 使用：

```bash
curl -sS https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.sh | sh
```

安装脚本会要求您确认安装路径（默认为`/usr/local/bin`），直接输入回车可以确认安装，或者您也可以手动输入安装路径。注意请确保安装路径在您的`$PATH`下面以可以直接使用命令行。

Windows 在 PowerShell 中使用：

```powershell
Invoke-RestMethod https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.ps1 | Invoke-Expression
```

Windows 安装脚本会自动选择 amd64 或 arm64 版本，默认安装到`%LOCALAPPDATA%\Programs\ucloud-sandbox-cli`，并将该目录加入用户`PATH`。

### 更新 CLI

从`v1.3.4`开始，CLI 可以自我更新，不需要重新执行安装脚本：

```bash
ucloud-sandbox-cli update
```

该命令会查询 GitHub 上的最新 Release，打印当前版本、最新版本和安装路径，等您输入`y`确认后，再下载与当前系统和架构匹配的二进制，校验 SHA256 并替换当前程序。

如果只想检查是否有新版本，不做任何改动，使用`--dry-run`：

```bash
ucloud-sandbox-cli update --dry-run
```

在脚本或 CI 中可以用`-y`跳过确认：

```bash
ucloud-sandbox-cli update -y
```

几点说明：

- 如果 CLI 安装在`/usr/local/bin`等需要管理员权限的目录，请使用`sudo ucloud-sandbox-cli update`。
- `v1.3.3`及以前的版本没有`update`命令，请重新执行上面的安装脚本来升级。
- 该命令使用 GitHub 的匿名 API 查询版本，如果遇到限流报错，可以设置`GITHUB_TOKEN`环境变量后重试。

## 身份认证与配置

认证与配置相关的命令都在 `auth` 下面。

### 环境注入

CLI 会优先读取环境变量中的 API Key、地域和其他配置。

```bash
export UCLOUD_SANDBOX_API_KEY=your_api_key
export UCLOUD_SANDBOX_REGION=region
```

如需使用 HTTP 而不是 HTTPS 连接控制面和沙箱，可通过环境变量启用 `insecure_http`：

```bash
export UCLOUD_SANDBOX_INSECURE_HTTP=true
```

也可以在 `~/.ucloud-sandbox-cli/config.json` 中持久化该配置：

```json
{
  "insecure_http": true
}
```

查看当前生效的配置（API Key 和镜像仓库密码会自动脱敏）：

```bash
ucloud-sandbox-cli auth config
```

直接用编辑器打开配置文件，编辑器取自 `$EDITOR`，未设置时用 `vim`，也可以在 `-e` 后面指定：

```bash
ucloud-sandbox-cli auth config -e
ucloud-sandbox-cli auth config -e nano
```

API key可以从星图平台的[密钥管理](https://astraflow.ucloud.cn/modelverse/api-keys)获取。

可用地域可以参考：[切换地域](https://astraflow.ucloud.cn/docs/agent-sandbox/product/region)。

### 持久化认证

配置持久化认证：

```bash
# 这个命令会要求您输入API key并选择默认地域
ucloud-sandbox-cli auth login
```

删除持久化认证：

```bash
ucloud-sandbox-cli auth logout
```

> 在持久化认证生效的情况下，仍然可以使用环境变量来替换API key和地域。

### 切换地域

快速选择并切换地域：

```bash
# 会列出当前可用的地域供您选择
ucloud-sandbox-cli auth region
```

### 镜像仓库凭据

构建模板时如果 base image 来自私有仓库，需要先登录该仓库。凭据按仓库域名保存，可以同时配置多个仓库：

```bash
# 省略域名时使用 docker.io
ucloud-sandbox-cli auth registry login
ucloud-sandbox-cli auth registry login uhub.service.ucloud.cn
```

命令会提示输入用户名和密码（密码输入时不回显）。删除某个仓库的凭据：

```bash
ucloud-sandbox-cli auth registry logout uhub.service.ucloud.cn
```

`template build` 会解析 Dockerfile 里 `FROM` 的镜像，按其仓库域名自动选用对应凭据；没有匹配条目时不带鉴权，公共镜像无需配置。

凭据在配置文件中的形状如下，也可以通过 `UCLOUD_SANDBOX_REGISTRIES` 环境变量以同样的 JSON 传入：

```json
{
  "registries": {
    "uhub.service.ucloud.cn": {
      "username": "<username>",
      "password": "<password>"
    }
  }
}
```

## 沙箱运行管理

### 创建与连接

快速创建沙箱并进入交互式终端：

```bash
# 使用内置模板创建沙箱
ucloud-sandbox-cli sandbox create [template]
# 简写：ucloud-sandbox-cli sbx cr [template]
```

**内置模板：**

```bash
# 代码解释器 - 预装 Python 和数据科学库
ucloud-sandbox-cli sandbox create code-interpreter-v1
 
# 桌面环境 - 支持图形化应用和浏览器
ucloud-sandbox-cli sandbox create desktop
 
# 基础环境 - 轻量级 Linux 环境
ucloud-sandbox-cli sandbox create base
```

创建沙箱时可以按 Volume 名称挂载一个或多个持久化 Volume：

```bash
ucloud-sandbox-cli sandbox create base --mount <volume-name>:/data
ucloud-sandbox-cli sandbox create base \
  --mount <volume-name-1>:/data \
  --mount <volume-name-2>:/cache
```

开启自动暂停/自动恢复后，沙箱在超时被暂停后，下次访问时会自动恢复运行：

```bash
ucloud-sandbox-cli sandbox create base --auto-pause --auto-resume --detach
```

> 创建成功后，CLI 会自动连接终端，您可以像操作本地 Shell 一样执行命令。按`Ctrl+D`或输入`exit`退出连接(沙箱继续运行)。

### 连接现有沙箱

重新连接到已运行的沙箱实例：

```bash
ucloud-sandbox-cli sandbox connect <sandbox-id>
# 简写：ucloud-sandbox-cli sbx connect <sandbox-id>
```

### 列表查询

查看名下所有活跃（运行或暂停）的沙箱实例：

```bash
ucloud-sandbox-cli sandbox list
# 简写：ucloud-sandbox-cli sandbox ls
```

### 执行命令

不进入交互终端，直接在沙箱里跑一条命令：

```bash
ucloud-sandbox-cli sandbox exec <sandbox-id> "python --version"

# CLI 自己的参数要写在沙箱 ID 前面
ucloud-sandbox-cli sandbox exec -u root <sandbox-id> "ls -la /root"
```

### 端口地址

拿到沙箱内某个端口对外的访问地址：

```bash
ucloud-sandbox-cli sandbox host <sandbox-id> 3000

# 输出完整 URL 而不是 host:port
ucloud-sandbox-cli sandbox host <sandbox-id> 3000 --url
```

### 暂停与复制

暂停沙箱并保留状态，下次连接时恢复：

```bash
ucloud-sandbox-cli sandbox pause <sandbox-id>

# 只保留文件系统、不保留内存（恢复时冷启动）
ucloud-sandbox-cli sandbox pause <sandbox-id> --memory=false
```

从一个运行中的沙箱复制出若干个副本，所有副本共用一次快照：

```bash
ucloud-sandbox-cli sandbox fork <sandbox-id>
ucloud-sandbox-cli sandbox fork <sandbox-id> --count 3 --timeout 3600
```

### 强制关停 (Kill)

立即释放沙箱资源：

```bash
# 关停特定 ID
ucloud-sandbox-cli sandbox kill <sandbox-id>

# 关停所有活跃沙箱，会先要求确认
ucloud-sandbox-cli sandbox kill --all

# 按状态过滤，并限制本次最多关停多少个
ucloud-sandbox-cli sandbox kill --all --state running --limit 10 -y
```

### 监控与日志

实时洞察沙箱运行状态：

```bash
# 查看资源占用指标 (CPU/RAM/Disk)
ucloud-sandbox-cli sandbox metrics <sandbox-id>

# 持续查看指标
ucloud-sandbox-cli sandbox metrics <sandbox-id> -w

# 指定时间区间，参数是 Unix 时间戳（秒）
ucloud-sandbox-cli sandbox metrics <sandbox-id> --start $(date -d '1 hour ago' +%s)
```

查看沙箱日志：

```bash
ucloud-sandbox-cli sandbox logs <sandbox-id>
ucloud-sandbox-cli sandbox logs <sandbox-id> --level warn
ucloud-sandbox-cli sandbox logs <sandbox-id> -f
```

## 文件管理

文件相关命令在 `sandbox fs` 下面，每个命令都支持 `-u/--user` 指定以哪个用户执行：

```bash
# 浏览
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user
ucloud-sandbox-cli sandbox fs ls <sandbox-id> /home/user -d 2 -f json

# 读取
ucloud-sandbox-cli sandbox fs cat <sandbox-id> /home/user/app.py

# 创建目录、移动、删除
ucloud-sandbox-cli sandbox fs mkdir <sandbox-id> /home/user/app
ucloud-sandbox-cli sandbox fs mv <sandbox-id> /home/user/old.txt /home/user/new.txt
ucloud-sandbox-cli sandbox fs rm <sandbox-id> /home/user/obsolete.txt
```

上传和下载使用 `fs cp`，远端一侧写成 `<sandbox-id>:<path>`，源和目标里必须恰好有一个远端：

```bash
# 上传
ucloud-sandbox-cli sandbox fs cp ./index.html <sandbox-id>:/home/user/app/index.html

# 上传到目录并保留文件名（远端路径以 / 结尾）
ucloud-sandbox-cli sandbox fs cp ./index.html <sandbox-id>:/home/user/app/

# 下载
ucloud-sandbox-cli sandbox fs cp <sandbox-id>:/home/user/app/out.txt ./out.txt
```

## Snapshot 管理

把沙箱当前状态保存为快照：

```bash
ucloud-sandbox-cli snapshot create <sandbox-id>

# 指定名称；名称已存在时会作为新的一次构建挂到该快照下
ucloud-sandbox-cli snapshot create <sandbox-id> -n nightly
```

列出和删除快照：

```bash
ucloud-sandbox-cli snapshot list
ucloud-sandbox-cli snapshot list -s <sandbox-id>
ucloud-sandbox-cli snapshot delete <snapshot-id...>
```

快照 ID 可以直接当作模板传给 `sandbox create`，从快照启动新沙箱：

```bash
ucloud-sandbox-cli sandbox create <snapshot-id>
```

## Secret 管理

Secret 的值是只写的，创建后不会再被返回：

```bash
# 不带 --value 时会提示输入，输入过程不回显
ucloud-sandbox-cli secret create openai-key
ucloud-sandbox-cli secret create openai-key --value "<value>"

# 从标准输入读取，适合多行内容
cat private.pem | ucloud-sandbox-cli secret create deploy-key --value-stdin
```

更新、查看和删除：

```bash
ucloud-sandbox-cli secret update openai-key
ucloud-sandbox-cli secret get openai-key
ucloud-sandbox-cli secret list
ucloud-sandbox-cli secret delete openai-key
```

在创建沙箱时通过 `--network-rules` 引用 Secret，平台会在出站请求中替换为真实值：

```bash
ucloud-sandbox-cli sandbox create base \
  --network-rules '{"api.example.com": {"X-API-KEY": "${openai-key}"}}'
```

## Volume 管理

创建持久化 Volume：

```bash
ucloud-sandbox-cli vol create <name>
```

列出 Volume：

```bash
ucloud-sandbox-cli vol list
ucloud-sandbox-cli vol list --format json
```

查看单个 Volume：

```bash
ucloud-sandbox-cli vol get <volume-id>
```

删除一个或多个 Volume：

```bash
ucloud-sandbox-cli vol delete <volume-id...>
```

## 模板构建管理

### 初始化模板项目

创建一个标准化的模板开发目录：

```bash
ucloud-sandbox-cli tpl init my-custom-env --cpu-count <cpu> --memory-mb <memory>
cd my-custom-env
```

生成的 `ucloud-template.json` 记录模板名、CPU、内存和 Dockerfile 文件名，`template build` 在命令行没给对应参数时会从这里取值。

### 构建模板

在上面的`my-custom-env`里面，您可以看到`template.dockerfile`文件，您需要编辑这个文件，输入`RUN`命令以定义构建模板需要的命令。

构建模板：

```bash
ucloud-sandbox-cli tpl build my-custom-env
```

常用构建参数：

```bash
# 指定资源；内存必须是偶数 MB
ucloud-sandbox-cli tpl build my-custom-env --cpu-count 2 --memory-mb 2048

# 跳过缓存，强制重跑每一步
ucloud-sandbox-cli tpl build my-custom-env --no-cache

# 打标签，并在构建完成后直接公开
ucloud-sandbox-cli tpl build my-custom-env -t v1 -t latest --publish

# 指定启动命令和就绪探针，两者必须一起给
ucloud-sandbox-cli tpl build my-custom-env \
  --cmd "python app.py" \
  --ready-cmd "curl -f http://localhost:8000/health"
```

可使用 `--level` 控制构建日志的最低等级（`debug`、`info`、`warn` 或 `error`），默认为 `info`：

```bash
ucloud-sandbox-cli tpl build my-custom-env --level debug
```

> 构建上下文就是 Dockerfile 所在的目录，`COPY` 的源路径相对于该目录解析。`--dockerfile` 指向该目录之外时命令会直接报错，而不是用错误的文件构建。

### 查看模板与构建日志

```bash
ucloud-sandbox-cli tpl list
ucloud-sandbox-cli tpl get <template-id>
ucloud-sandbox-cli tpl logs <template-id> <build-id>
```

### 发布模板

默认情况下，模板只能由您当前的项目访问，如果您需要其他人也能使用模板，需要公开：

```bash
ucloud-sandbox-cli tpl publish <template-id>
```

取消公开：

```bash
ucloud-sandbox-cli tpl publish --unpublish <template-id>
```

### 模板标签

标签指向模板的某一次构建，可以用来固定版本：

```bash
ucloud-sandbox-cli tpl tag list my-custom-env
ucloud-sandbox-cli tpl tag assign my-custom-env:latest v1
ucloud-sandbox-cli tpl tag remove my-custom-env v1
```

### 删除模板

```bash
ucloud-sandbox-cli tpl delete <template-id>

# 从列表中交互选择
ucloud-sandbox-cli tpl delete --select
```

## 输出格式与分页

所有列表类命令（`sandbox list`、`volume list`、`secret list`、`snapshot list`、`template list`、`template tag list`、`sandbox fs ls`）都支持同一组参数：

```bash
-f, --format string   输出格式，pretty 或 json（默认 pretty）
-p, --page int        页码（默认 1）
-l, --limit int       每页条数，0 表示全部
```

需要在脚本里解析结果时用 `-f json`。

## 典型工作流示例

1. 准备环境：`ucloud-sandbox-cli auth login`
2. 创建模板：`ucloud-sandbox-cli template init` -> 编写 `template.dockerfile` -> `ucloud-sandbox-cli template build`
3. 业务接入：在 SDK 中使用 `Sandbox.create(template='my-agent-env')`
4. 资源回收：`ucloud-sandbox-cli sandbox kill --all`
