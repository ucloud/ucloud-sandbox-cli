# Windows PowerShell 指南

在 Windows 中使用 `ucloud-sandbox-site` 连接、维护和部署站点空间时遵循本指南，同时继续遵守主 `SKILL.md` 的权限边界、凭证保护和部署要求。

## 目录

- [本地与远端边界](#本地与远端边界)
- [检查并安装 CLI](#检查并安装-cli)
- [设置站点凭证并验证连接](#设置站点凭证并验证连接)
- [常用文件和命令操作](#常用文件和命令操作)
- [打包并上传目录](#打包并上传目录)
- [执行多行远端命令](#执行多行远端命令)
- [故障处理](#故障处理)

## 本地与远端边界

- 本地使用 PowerShell 和 Windows `.exe`，不要执行主 `SKILL.md` 中的 Bash 安装、凭证注入或本地打包命令。
- `sandbox exec` 的命令在远端 Linux 沙箱中运行，继续使用 `/home/user/...`、`source`、`sudo -n` 等 Linux 语法。
- 本地路径使用 `C:\...`；远端路径使用 `/`。远端端点写成 `${SandboxId}:/path`，避免 PowerShell 把冒号解析成变量作用域。
- 把完整 `site_...` 视为凭证，不要输出、记录或写入配置文件。每次开启新的 PowerShell 调用时重新设置凭证。
- 执行需要公网的操作前，先按主 `SKILL.md` 申请并确认网络权限。

## 检查并安装 CLI

检查并卸载旧 npm 版；仅在命令不存在时调用独立 `install.ps1`。卸载需要管理员权限时，让用户在真实终端处理：

```powershell
if (Get-Command npm -ErrorAction SilentlyContinue) {
  npm list -g "@ucloud-sdks/ucloud-sandbox-cli" --depth=0 *> $null
  if ($LASTEXITCODE -eq 0) {
    npm uninstall -g "@ucloud-sdks/ucloud-sandbox-cli"
    if ($LASTEXITCODE -ne 0) { throw "Failed to uninstall the old npm CLI." }
  }
}

if (-not (Get-Command ucloud-sandbox-cli -CommandType Application -ErrorAction SilentlyContinue)) {
  [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
  $InstallerUrl = "https://raw.githubusercontent.com/Piwriw/ucloud-sandbox-cli/feat/windows/install.ps1"
  & ([scriptblock]::Create((Invoke-RestMethod -Uri $InstallerUrl -UseBasicParsing -ErrorAction Stop)))
}

ucloud-sandbox-cli version
if ($LASTEXITCODE -ne 0) { throw "ucloud-sandbox-cli verification failed." }
```

安装脚本默认安装到 `%LOCALAPPDATA%\Programs\ucloud-sandbox-cli`，并更新当前进程和用户 `PATH`。不要自动更新已经可正常运行的 CLI。

## 设置站点凭证并验证连接

只在当前 PowerShell 进程中设置完整站点 ID，并派生去掉 `site_` 前缀的沙箱 ID：

```powershell
$SiteId = "site_<sandbox-id>"

if (-not $SiteId.StartsWith("site_", [StringComparison]::Ordinal) -or $SiteId.Length -le 5) {
  throw "Invalid site ID. Expected site_<sandbox-id>."
}

$SandboxId = $SiteId.Substring(5)
$env:UCLOUD_SANDBOX_API_KEY = $SiteId

ucloud-sandbox-cli sandbox exec $SandboxId "printf 'SITE_CONNECTED\n'; pwd"
if ($LASTEXITCODE -ne 0) { throw "Site connection verification failed." }
```

不要用 `sandbox list` 验证连接。地域或 API 域名仍沿用已有 CLI 配置；需要临时指定时使用 `$env:UCLOUD_SANDBOX_REGION` 和 `$env:UCLOUD_SANDBOX_DOMAIN`，没有证据时不要擅自切换。

## 常用文件和命令操作

主 `SKILL.md` 示例中的 `$SANDBOX_ID` 在 PowerShell 中统一写成 `$SandboxId`：

```powershell
ucloud-sandbox-cli sandbox exec $SandboxId "pwd && ls -la /home/user"
ucloud-sandbox-cli fs ls $SandboxId /home/user --format json
ucloud-sandbox-cli fs mkdir $SandboxId /home/user/site

# 上传
ucloud-sandbox-cli fs cp "C:\work\index.html" "${SandboxId}:/home/user/site/index.html"

# 下载
ucloud-sandbox-cli fs cp "${SandboxId}:/home/user/site/service.log" "C:\work\service.log"
```

读取文件、删除文件、部署服务和查看环境变量时，继续遵守主 `SKILL.md` 的敏感信息与破坏性操作限制。

## 打包并上传目录

Windows 自带或已安装 `tar` 时，可以在本地 PowerShell 打包；排除凭证、依赖和版本控制目录：

```powershell
$LocalProjectDir = "C:\work\site"
$Archive = Join-Path ([IO.Path]::GetTempPath()) "site-release-$PID.tgz"

if (-not (Test-Path -LiteralPath $LocalProjectDir -PathType Container)) {
  throw "Local project directory does not exist: $LocalProjectDir"
}

try {
  tar --exclude=.git --exclude=.env --exclude=.env.* --exclude=.site.env --exclude=node_modules `
    -czf $Archive -C $LocalProjectDir .
  if ($LASTEXITCODE -ne 0) { throw "Failed to create deployment archive." }

  ucloud-sandbox-cli fs cp $Archive "${SandboxId}:/tmp/site-release.tgz"
  if ($LASTEXITCODE -ne 0) { throw "Failed to upload deployment archive." }

  ucloud-sandbox-cli sandbox exec $SandboxId `
    "mkdir -p /home/user/site && tar -xzf /tmp/site-release.tgz -C /home/user/site && rm -f /tmp/site-release.tgz"
  if ($LASTEXITCODE -ne 0) { throw "Failed to extract deployment archive." }
} finally {
  Remove-Item -LiteralPath $Archive -Force -ErrorAction SilentlyContinue
}
```

上传前先检查目标目录；不要无条件清空已有网站。

## 执行多行远端命令

包含 `$`、`$()`、引号或多行 Linux Shell 的远端命令使用单引号 here-string，并把 CRLF 转换为 LF：

```powershell
$RemoteCommand = @'
set -e
cd /home/user/site
set -a
source /home/user/.site.env
set +a
npm ci
npm run build
'@
$RemoteCommand = $RemoteCommand.Replace("`r`n", "`n")

ucloud-sandbox-cli sandbox exec $SandboxId $RemoteCommand
```

主 `SKILL.md` 中的环境变量脱敏、持久启动和服务诊断命令都按此模式传入；不要让 PowerShell 在本地提前展开远端 `$HOME`、`$PID` 或 `$()`。

## 故障处理

- 命令安装成功但找不到时，将安装目录加入当前进程 `PATH`：`$env:Path = "<安装目录>;$env:Path"`，并确认安装目录已写入用户 `PATH`。
- 出现 `Variable reference is not valid` 时，检查远端端点是否写成 `${SandboxId}:/path`。
- 远端多行命令出现 `\r` 或语法错误时，确认 here-string 已通过 `.Replace("`r`n", "`n")` 转换换行。
- 新的 PowerShell 调用提示缺少 API Key 时，重新设置 `$env:UCLOUD_SANDBOX_API_KEY = $SiteId`，不要把站点 ID 写入持久化配置。
