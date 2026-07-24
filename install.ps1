<#
.SYNOPSIS
Downloads and installs UCloud Sandbox CLI on Windows.

.PARAMETER Version
Release tag to install. Defaults to the latest release.

.PARAMETER InstallDir
Directory where ucloud-sandbox-cli.exe is installed.

.PARAMETER BaseUrl
Base URL of the GitHub releases page.

.EXAMPLE
Invoke-RestMethod https://raw.githubusercontent.com/ucloud/ucloud-sandbox-cli/main/install.ps1 | Invoke-Expression

.EXAMPLE
.\install.ps1 -Version v1.2.3
#>

[CmdletBinding()]
param(
    [ValidateNotNullOrEmpty()]
    [string]$Version = "latest",

    [string]$InstallDir = "",

    [ValidateNotNullOrEmpty()]
    [string]$BaseUrl = "https://github.com/ucloud/ucloud-sandbox-cli/releases"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$BinaryName = "ucloud-sandbox-cli"
$DocsUrl = "https://astraflow.ucloud.cn/docs/agent-sandbox/product/cli"

function Write-Info {
    param([string]$Message)
    Write-Host "> $Message"
}

function Write-Success {
    param([string]$Message)
    Write-Host "+ $Message" -ForegroundColor Green
}

function Get-WindowsArchitecture {
    $machineArchitecture = if ($env:PROCESSOR_ARCHITEW6432) {
        $env:PROCESSOR_ARCHITEW6432
    } else {
        $env:PROCESSOR_ARCHITECTURE
    }

    if ([string]::IsNullOrWhiteSpace($machineArchitecture)) {
        throw "Unable to detect the Windows architecture."
    }

    switch ($machineArchitecture.ToUpperInvariant()) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { throw "Unsupported Windows architecture: $machineArchitecture. Supported architectures are: amd64, arm64." }
    }
}

function Add-InstallDirToPath {
    param([string]$Directory)

    $normalizedDirectory = [Environment]::ExpandEnvironmentVariables($Directory).TrimEnd("\")
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $userEntries = @($userPath -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $isInUserPath = $userEntries | Where-Object {
        [Environment]::ExpandEnvironmentVariables($_).Trim().TrimEnd("\") -ieq $normalizedDirectory
    }

    if (-not $isInUserPath) {
        $newUserPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
            $Directory
        } else {
            "$Directory;$userPath"
        }
        [Environment]::SetEnvironmentVariable("Path", $newUserPath, "User")
        Write-Info "Added $Directory to the user PATH."
    }

    $processEntries = @($env:Path -split ";" | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    $isInProcessPath = $processEntries | Where-Object {
        [Environment]::ExpandEnvironmentVariables($_).Trim().TrimEnd("\") -ieq $normalizedDirectory
    }
    if (-not $isInProcessPath) {
        $env:Path = "$Directory;$env:Path"
    }
}

if ($env:OS -ne "Windows_NT") {
    throw "install.ps1 only supports Windows. Use install.sh on Linux or macOS."
}

if ([string]::IsNullOrWhiteSpace($InstallDir)) {
    if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) {
        throw "LOCALAPPDATA is not set. Pass -InstallDir with a writable installation directory."
    }
    $InstallDir = Join-Path $env:LOCALAPPDATA "Programs\ucloud-sandbox-cli"
}
$InstallDir = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($InstallDir)

$architecture = Get-WindowsArchitecture
$assetName = "${BinaryName}_windows_${architecture}.zip"
$releaseBaseUrl = $BaseUrl.TrimEnd("/")
$downloadUrl = if ($Version -eq "latest") {
    "$releaseBaseUrl/latest/download/$assetName"
} else {
    "$releaseBaseUrl/download/$Version/$assetName"
}
$targetBinary = Join-Path $InstallDir "$BinaryName.exe"
$tempDir = Join-Path ([IO.Path]::GetTempPath()) "$BinaryName-install-$([Guid]::NewGuid().ToString('N'))"
$archivePath = Join-Path $tempDir $assetName
$extractDir = Join-Path $tempDir "extract"

Write-Host ""
Write-Info "Welcome to the $BinaryName installer."
Write-Info "Installer configuration:"
Write-Info "  Version: $Version"
Write-Info "  OS: windows"
Write-Info "  Arch: $architecture"
Write-Info "  Install dir: $InstallDir"
Write-Info "  Download URL: $downloadUrl"
Write-Host ""

try {
    New-Item -ItemType Directory -Force -Path $tempDir, $extractDir, $InstallDir | Out-Null

    # Windows PowerShell 5.1 can otherwise negotiate an obsolete TLS version.
    [Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

    Write-Info "Downloading $BinaryName..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $archivePath -UseBasicParsing

    Write-Info "Installing $BinaryName to $InstallDir..."
    Expand-Archive -LiteralPath $archivePath -DestinationPath $extractDir -Force
    $extractedBinary = Join-Path $extractDir "$BinaryName.exe"
    if (-not (Test-Path -LiteralPath $extractedBinary -PathType Leaf)) {
        throw "Archive did not contain $BinaryName.exe."
    }

    Copy-Item -LiteralPath $extractedBinary -Destination $targetBinary -Force
    Unblock-File -LiteralPath $targetBinary -ErrorAction SilentlyContinue
    Add-InstallDirToPath -Directory $InstallDir

    Write-Success "$BinaryName installed successfully."
    Write-Host ""
    Write-Info "Installed binary version:"
    & $targetBinary version
    if ($LASTEXITCODE -ne 0) {
        throw "$BinaryName version exited with code $LASTEXITCODE."
    }
} finally {
    if (Test-Path -LiteralPath $tempDir) {
        Remove-Item -LiteralPath $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

Write-Host ""
Write-Info "Documentation: $DocsUrl"
Write-Info "Run '$BinaryName login' first, then start using the CLI."
