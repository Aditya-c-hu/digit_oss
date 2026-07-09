# =============================================================================
# 03-start-infra.ps1
# Starts PostgreSQL + Kafka inside Docker Compose, waits for localhost ports.
# =============================================================================
[CmdletBinding()]
param()

$ErrorActionPreference = 'Stop'
$LogDir = "$PSScriptRoot\logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

function Wait-Port {
    param([string]$Host_,[int]$Port,[int]$TimeoutSec = 60)
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    while ((Get-Date) -lt $deadline) {
        try {
            $c = New-Object System.Net.Sockets.TcpClient
            $iar = $c.BeginConnect($Host_, $Port, $null, $null)
            if ($iar.AsyncWaitHandle.WaitOne(2000) -and $c.Connected) {
                $c.Close(); return $true
            }
            $c.Close()
        } catch {}
        Start-Sleep -Milliseconds 500
    }
    return $false
}

$Root = Split-Path -Parent $PSScriptRoot

# ---- Docker Compose Infra --------------------------------------------------
Write-Host "==> Starting PostgreSQL and Kafka via Docker Compose..." -ForegroundColor Cyan

Push-Location $Root
try {
    # Start postgres and kafka services in detached mode
    & docker compose up -d postgres kafka
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to run docker compose up -d postgres kafka"
    }
} finally {
    Pop-Location
}

# ---- Wait for PostgreSQL ---------------------------------------------------
Write-Host "==> Verifying PostgreSQL port (5433)..." -ForegroundColor Cyan
if (Wait-Port -Host_ 'localhost' -Port 5433 -TimeoutSec 30) {
    Write-Host "    Postgres up on 5433" -ForegroundColor Green
} else {
    Write-Host "    Postgres did not open 5433 in time" -ForegroundColor Red
    exit 1
}

# ---- Wait for Kafka --------------------------------------------------------
Write-Host "==> Verifying Kafka port (9092)..." -ForegroundColor Cyan
if (Wait-Port -Host_ 'localhost' -Port 9092 -TimeoutSec 60) {
    Write-Host "    Kafka up on 9092" -ForegroundColor Green
} else {
    Write-Host "    Kafka did not open 9092 in time" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "=== Infra ready: postgres:5433, kafka:9092 ===" -ForegroundColor Green
