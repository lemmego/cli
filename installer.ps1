# Installs the Lemmego CLI on Windows.
#
#   irm https://raw.githubusercontent.com/lemmego/cli/refs/heads/main/installer.ps1 | iex
#
# The version is resolved from the latest GitHub release, so this script does
# not need editing when a new version ships. Override it to pin:
#
#   $env:LEMMEGO_VERSION = "v0.1.45"
#   $env:LEMMEGO_INSTALL_DIR = "$env:LOCALAPPDATA\Programs\lemmego"

$ErrorActionPreference = "Stop"

$repo = "lemmego/cli"
$installDir = if ($env:LEMMEGO_INSTALL_DIR) { $env:LEMMEGO_INSTALL_DIR } else { "C:\Windows\System32" }
$destination = Join-Path $installDir "lemmego.exe"

# Resolve the architecture of the process' host, not the shell.
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    "AMD64" { "amd64" }
    "ARM64" { "arm64" }
    "x86"   { if ($env:PROCESSOR_ARCHITEW6432 -eq "ARM64") { "arm64" } else { "amd64" } }
    default { $null }
}
if (-not $arch) {
    Write-Error "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE"
    exit 1
}

$version = $env:LEMMEGO_VERSION
if (-not $version) {
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$repo/releases/latest" -UseBasicParsing
        $version = $release.tag_name
    } catch {
        Write-Error "Could not determine the latest version. Set `$env:LEMMEGO_VERSION to install a specific one."
        exit 1
    }
}

$asset = "lemmego-$version-windows-$arch.exe"
$baseUrl = "https://github.com/$repo/releases/download/$version"
$tempFile = Join-Path ([System.IO.Path]::GetTempPath()) $asset

Write-Host "Downloading $asset"
try {
    Invoke-WebRequest -Uri "$baseUrl/$asset" -OutFile $tempFile -UseBasicParsing
} catch {
    Write-Error "Download failed: $baseUrl/$asset"
    exit 1
}

# Verify against the checksum published alongside the binary.
try {
    $expected = (Invoke-WebRequest -Uri "$baseUrl/$asset.md5" -UseBasicParsing).Content.Trim()
    $actual = (Get-FileHash -Path $tempFile -Algorithm MD5).Hash.ToLower()
    if ($actual -ne $expected.ToLower()) {
        Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
        Write-Error "Checksum mismatch for $asset`n  expected $expected`n  actual   $actual"
        exit 1
    }
} catch [System.Net.WebException] {
    Write-Host "No checksum published for $asset; skipping verification."
}

Write-Host "Installing to $destination"
try {
    New-Item -ItemType Directory -Path $installDir -Force | Out-Null
    # Force overwrites an existing binary. The previous version of this script
    # skipped the move when the file already existed, so re-running it never
    # upgraded an installation.
    Move-Item -Path $tempFile -Destination $destination -Force
} catch {
    Remove-Item $tempFile -Force -ErrorAction SilentlyContinue
    Write-Error "Failed to write $destination. Run this from an elevated prompt, or set `$env:LEMMEGO_INSTALL_DIR to a directory you own."
    exit 1
}

Write-Host "Installation completed."
& $destination --version
