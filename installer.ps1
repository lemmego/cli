# Determine the architecture
$arch = (Get-WmiObject Win32_OperatingSystem).OSArchitecture

# Function to download and install the binary
function Install-Lemmego {
    $downloadUrl = ""
    $destinationPath = "C:\Windows\System32\lemmego.exe"

    if ($arch -match "64-bit") {
        if ($env:PROCESSOR_ARCHITEW6432 -eq "ARM64") {
            $downloadUrl = "https://github.com/lemmego/cli/releases/download/v0.1.2/lemmego-v0.1.2-windows-arm64.exe"
        } else {
            $downloadUrl = "https://github.com/lemmego/cli/releases/download/v0.1.2/lemmego-v0.1.2-windows-amd64.exe"
        }
    } else {
        Write-Host "Unsupported architecture: $arch"
        exit 1
    }

    if (-not $downloadUrl) {
        Write-Host "Failed to determine the download URL for this platform."
        exit 1
    }

    Write-Host "Downloading: $downloadUrl"
    Invoke-WebRequest -Uri $downloadUrl -OutFile "lemmego.exe"

    # Move the file to the System32 directory
    Write-Host "Moving file to $destinationPath"
    if (-not (Test-Path $destinationPath)) {
        Move-Item -Path ".\lemmego.exe" -Destination $destinationPath
        if ($?) {
            Write-Host "Installation completed."
        } else {
            Write-Host "Failed to move the file to $destinationPath. Please check if you have administrator privileges."
        }
    } else {
        Write-Host "File already exists at $destinationPath. Skipping move."
    }
}

# Check if script is running with administrator privileges
if (-not ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "This script requires administrator privileges. Please run PowerShell as an administrator."
    exit 1
}

# Run the installation
Install-Lemmego