# Windows PowerShell 指南

在 Windows 中安装、更新、配置或调用 `ucloud-sandbox-cli` 时使用本指南。

## 目录

- [基本规则](#基本规则)
- [安装或更新本技能](#安装或更新本技能)
- [检查并安装 CLI](#检查并安装-cli)
- [更新 CLI](#更新-cli)
- [认证和配置](#认证和配置)
- [PowerShell 调用](#powershell-调用)
- [故障处理](#故障处理)

## 基本规则

- 在本地使用 PowerShell 和 Windows `.exe`，不要执行 Bash 安装命令；沙箱内仍使用 Linux Shell。
- 使用 `$env:NAME` 设置环境变量，本地路径使用 `C:\...`，远端路径仍使用 `/`。
- 执行需要公网的操作前，遵守主 `SKILL.md` 的公网权限检查。

## 安装或更新本技能

仅在用户明确要求安装或更新技能时执行。把 `$TargetAgent` 设置为 `codex`、`claude` 或 `gemini`：

```powershell
$TargetAgent = "codex"
$CodexHome = if ($env:CODEX_HOME) { $env:CODEX_HOME } else { Join-Path $HOME ".codex" }
$SkillRoot = switch ($TargetAgent) {
  "codex"  { Join-Path $CodexHome "skills" }
  "claude" { Join-Path $HOME ".claude\skills" }
  "gemini" { Join-Path $HOME ".gemini\skills" }
  default   { throw "TargetAgent must be codex, claude, or gemini." }
}
$SkillDir = Join-Path $SkillRoot "ucloud-sandbox"
$ReferencesDir = Join-Path $SkillDir "references"
$BaseUrl = "https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/skills/ucloud-sandbox"

[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
New-Item -ItemType Directory -Force -Path $SkillDir, $ReferencesDir | Out-Null
Invoke-WebRequest -Uri "$BaseUrl/SKILL.md" -OutFile (Join-Path $SkillDir "SKILL.md") -UseBasicParsing -ErrorAction Stop
Invoke-WebRequest -Uri "$BaseUrl/references/windows.md" -OutFile (Join-Path $ReferencesDir "windows.md") -UseBasicParsing -ErrorAction Stop
"ucloud-sandbox skill installed or updated at $SkillDir"
```

## 检查并安装 CLI

检查并卸载旧 npm 版；仅在命令不存在时调用仓库根目录的独立 `install.ps1`。卸载需要管理员权限时，让用户在真实终端处理：

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
  & ([scriptblock]::Create((Invoke-RestMethod -Uri "https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.ps1" -UseBasicParsing -ErrorAction Stop)))
}

ucloud-sandbox-cli version
if ($LASTEXITCODE -ne 0) { throw "ucloud-sandbox-cli verification failed." }
```

安装脚本默认安装到 `%LOCALAPPDATA%\Programs\ucloud-sandbox-cli`，并更新当前进程和用户 `PATH`。

## 更新 CLI

仅在用户明确要求更新 CLI 时执行。

从 `v1.3.4` 开始，CLI 自带 `update` 命令，可以自我更新到最新版本。PowerShell 中同样没有交互式确认，必须带 `-y`：

```powershell
ucloud-sandbox-cli update -y
if ($LASTEXITCODE -ne 0) { throw "ucloud-sandbox-cli update failed." }
ucloud-sandbox-cli version
```

`update` 会下载与当前架构匹配的 ZIP，校验 SHA256 后替换当前程序。只检查不安装时使用 `ucloud-sandbox-cli update --dry-run`。默认安装目录 `%LOCALAPPDATA%\Programs\ucloud-sandbox-cli` 属于当前用户，不需要管理员权限；如果 CLI 被装到 `Program Files` 等位置，让用户在管理员 PowerShell 中执行更新。查询版本报 GitHub API 限流时，让用户设置 `GITHUB_TOKEN` 环境变量后重试。

`v1.3.3` 及以前的版本没有 `update` 命令，需要指定版本或沿用自定义目录时，仍然调用安装脚本。更新到 latest 时保持 `$InstallArgs` 为空；指定版本或沿用自定义目录时设置相应字段，可以同时设置：

```powershell
$InstallArgs = @{
  # Version = "v1.2.3"
  # InstallDir = Join-Path $HOME "bin"
}

[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
$InstallerUrl = "https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.ps1"
& ([scriptblock]::Create((Invoke-RestMethod -Uri $InstallerUrl -UseBasicParsing -ErrorAction Stop))) @InstallArgs
ucloud-sandbox-cli version
if ($LASTEXITCODE -ne 0) { throw "ucloud-sandbox-cli verification failed." }
```

## 认证和配置

配置文件位于 `~/.ucloud-sandbox-cli/config.json`，使用用户配置目录的 ACL。不要输出真实 `api_key`，也不要输出 `registries` 里任何仓库的 `password`。

查看配置优先用 CLI，它输出 JSON 并已经把 `api_key` 和每个仓库的 `password` 替换成 `****`：

```powershell
ucloud-sandbox-cli auth config
```

临时使用环境变量时，让用户在自己的终端设置：

```powershell
$env:UCLOUD_SANDBOX_API_KEY = "<api-key>"
$env:UCLOUD_SANDBOX_REGION = "cn-wlcb"
$env:UCLOUD_SANDBOX_DOMAIN = "cn-wlcb.sandbox.ucloudai.com"
```

私有仓库凭据按仓库域名保存在配置的 `registries` 下，让用户在真实终端执行 `ucloud-sandbox-cli auth registry login <domain>` 配置；需要在 CI 中注入时用同形状的 JSON：

```powershell
$env:UCLOUD_SANDBOX_REGISTRIES = '{"uhub.service.ucloud.cn":{"username":"<username>","password":"<password>"}}'
```

切换持久化地域时使用结构化 JSON API，不要输出 `$Config`。已有标准地域 `domain` 时同步更新，因为它优先于 `region`；检测到自定义域名时停止并先向用户确认：

```powershell
$ConfigFile = Join-Path $HOME ".ucloud-sandbox-cli\config.json"
$NewRegion = "cn-wlcb"
$NewDomain = "$NewRegion.sandbox.ucloudai.com"

if (-not (Test-Path -LiteralPath $ConfigFile -PathType Leaf)) {
  throw "Config file not found. Run 'ucloud-sandbox-cli auth login' in a real terminal first."
}

$Config = Get-Content -LiteralPath $ConfigFile -Raw -ErrorAction Stop | ConvertFrom-Json
$Config | Add-Member -NotePropertyName "region" -NotePropertyValue $NewRegion -Force
$ExistingDomain = $Config.PSObject.Properties["domain"]
if ($ExistingDomain -and -not [string]::IsNullOrWhiteSpace([string]$ExistingDomain.Value)) {
  if ([string]$ExistingDomain.Value -notmatch '^[a-z0-9-]+\.sandbox\.ucloudai\.com$') {
    throw "Config uses a custom domain. Confirm it before changing the region."
  }
  $ExistingDomain.Value = $NewDomain
}
$Json = $Config | ConvertTo-Json -Depth 10
$Utf8NoBom = New-Object System.Text.UTF8Encoding($false)
$TempFile = Join-Path ([IO.Path]::GetDirectoryName($ConfigFile)) "config.$PID.tmp"

try {
  [IO.File]::WriteAllText($TempFile, $Json, $Utf8NoBom)
  [IO.File]::Replace($TempFile, $ConfigFile, $null)
} finally {
  Remove-Item -LiteralPath $TempFile -Force -ErrorAction SilentlyContinue
}
```

只读取地域和域名时，不要输出整个配置：

```powershell
$ConfigFile = Join-Path $HOME ".ucloud-sandbox-cli\config.json"
$Config = Get-Content -LiteralPath $ConfigFile -Raw -ErrorAction Stop | ConvertFrom-Json
$Region = $Config.PSObject.Properties["region"]
$Domain = $Config.PSObject.Properties["domain"]
if ($Region) { "region=$($Region.Value)" } else { "region=" }
if ($Domain) { "domain=$($Domain.Value)" } else { "domain=" }
```

必须自己读文件展示摘要时先脱敏，`api_key` 和每个仓库的 `password` 都要遮住（优先改用 `ucloud-sandbox-cli auth config`，它已经做了这件事）：

```powershell
$ConfigFile = Join-Path $HOME ".ucloud-sandbox-cli\config.json"
$Summary = Get-Content -LiteralPath $ConfigFile -Raw -ErrorAction Stop | ConvertFrom-Json
$ApiKey = $Summary.PSObject.Properties["api_key"]
if ($ApiKey) { $ApiKey.Value = "***hidden***" }
$Registries = $Summary.PSObject.Properties["registries"]
if ($Registries -and $Registries.Value) {
  foreach ($Entry in $Registries.Value.PSObject.Properties) {
    $Password = $Entry.Value.PSObject.Properties["password"]
    if ($Password) { $Password.Value = "***hidden***" }
  }
}
$Summary | ConvertTo-Json -Depth 10
```

## PowerShell 调用

- 变量后紧跟远端端点或 Volume 挂载参数的冒号时，写成 `${SandboxId}:/path` 或 `${VolumeName}:/path`，避免 PowerShell 把冒号解析为变量作用域。
- 包含 `$`、`$()` 或多行 Shell 的远端命令使用单引号 here-string，并把 CRLF 转换为 LF。

```powershell
$SandboxId = "<sandbox-id>"
$VolumeName = "workspace"
ucloud-sandbox-cli sandbox exec $SandboxId "pwd && ls -la"
ucloud-sandbox-cli sandbox fs cp "C:\work\index.html" "${SandboxId}:/home/user/app/index.html"
ucloud-sandbox-cli sandbox create base --mount "${VolumeName}:/data" --detach

$RemoteCommand = @'
printf 'HOME=%s\n' "$HOME"
'@
$RemoteCommand = $RemoteCommand.Replace("`r`n", "`n")
ucloud-sandbox-cli sandbox exec $SandboxId $RemoteCommand
```

## 故障处理

命令安装成功但找不到时，将安装目录加入当前进程 `PATH`，并确认安装目录已经写入用户 `PATH`：

```powershell
$env:Path = "<安装目录>;$env:Path"
```
