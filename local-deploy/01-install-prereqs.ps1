# =============================================================================
# 01-install-prereqs.ps1
# Installs JDK 8, Maven, PostgreSQL 15, and downloads Kafka 3.7 to C:\kafka.
# Run as Administrator. Re-run is safe (idempotent).
# =============================================================================
[CmdletBinding()]
param(
    [string]$KafkaDir = "C:\kafka",
    [string]$KafkaVersion = "3.7.0",
    [string]$ScalaVersion = "2.13"
)

$ErrorActionPreference = 'Stop'

function Test-Admin {
    $id = [Security.Principal.WindowsIdentity]::GetCurrent()
    $p  = New-Object Security.Principal.WindowsPrincipal($id)
    return $p.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

if (-not (Test-Admin)) {
    Write-Host "Must run as Administrator. Right-click PowerShell -> Run as Administrator." -ForegroundColor Red
    exit 1
}

Write-Host "==> Checking winget..."
$null = winget --version

# ---- JDK 8 -----------------------------------------------------------------
Write-Host "==> Installing Eclipse Temurin JDK 8..."
winget install --id EclipseAdoptium.Temurin.8.JDK -e --silent --accept-source-agreements --accept-package-agreements
# JAVA_HOME setup
$jdk = Get-ChildItem 'C:\Program Files\Eclipse Adoptium\' -Directory -ErrorAction SilentlyContinue |
       Where-Object { $_.Name -like 'jdk-8*' } | Select-Object -First 1
if ($jdk) {
    [Environment]::SetEnvironmentVariable('JAVA_HOME', $jdk.FullName, 'Machine')
    Write-Host "    JAVA_HOME set to $($jdk.FullName)"
}

# ---- Maven -----------------------------------------------------------------
Write-Host "==> Installing Apache Maven..."
winget install --id Apache.Maven -e --silent --accept-source-agreements --accept-package-agreements

# ---- PostgreSQL 15 ---------------------------------------------------------
# Skipped as per user request (running in Docker Compose)

# ---- Kafka 3.7 (manual download — no winget package) -----------------------
# Skipped as per user request (running in Docker Compose)

Write-Host ""
Write-Host "=============================================================" -ForegroundColor Green
Write-Host " JDK 8 & Maven install complete." -ForegroundColor Green
Write-Host " CLOSE this PowerShell and OPEN a new one so PATH refreshes." -ForegroundColor Yellow
Write-Host " Then verify: java -version ; mvn -v" -ForegroundColor Yellow
Write-Host "=============================================================" -ForegroundColor Green
