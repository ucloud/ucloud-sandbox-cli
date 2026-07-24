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

仅在用户明确要求更新 CLI 时执行。更新到 latest 时保持 `$InstallArgs` 为空；指定版本或沿用自定义目录时设置相应字段，可以同时设置：

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

配置文件位于 `~/.ucloud-sandbox-cli/config.json`，使用用户配置目录的 ACL。不要输出真实 `api_key`。

临时使用环境变量时，让用户在自己的终端设置：

```powershell
$env:UCLOUD_SANDBOX_API_KEY = "<api-key>"
$env:UCLOUD_SANDBOX_REGION = "cn-wlcb"
$env:UCLOUD_SANDBOX_DOMAIN = "cn-wlcb.sandbox.ucloudai.com"
```

切换持久化地域时使用结构化 JSON API，不要输出 `$Config`。已有标准地域 `domain` 时同步更新，因为它优先于 `region`；检测到自定义域名时停止并先向用户确认：

```powershell
$ConfigFile = Join-Path $HOME ".ucloud-sandbox-cli\config.json"
$NewRegion = "cn-wlcb"
$NewDomain = "$NewRegion.sandbox.ucloudai.com"

if (-not (Test-Path -LiteralPath $ConfigFile -PathType Leaf)) {
  throw "Config file not found. Run 'ucloud-sandbox-cli login' in a real terminal first."
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

必须展示配置摘要时先脱敏：

```powershell
$ConfigFile = Join-Path $HOME ".ucloud-sandbox-cli\config.json"
$Summary = Get-Content -LiteralPath $ConfigFile -Raw -ErrorAction Stop | ConvertFrom-Json
$ApiKey = $Summary.PSObject.Properties["api_key"]
if ($ApiKey) { $ApiKey.Value = "***hidden***" }
$Summary | ConvertTo-Json -Depth 10
```

## PowerShell 调用

- 变量后紧跟远端端点冒号时写成 `${SandboxId}:/path`，避免 PowerShell 把冒号解析为变量作用域。
- 包含 `$`、`$()` 或多行 Shell 的远端命令使用单引号 here-string，并把 CRLF 转换为 LF。

```powershell
$SandboxId = "<sandbox-id>"
ucloud-sandbox-cli sandbox exec $SandboxId "pwd && ls -la"
ucloud-sandbox-cli fs cp "C:\work\index.html" "${SandboxId}:/home/user/app/index.html"

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
