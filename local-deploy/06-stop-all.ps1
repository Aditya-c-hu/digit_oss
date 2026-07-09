# =============================================================================
# 06-stop-all.ps1
# Kills services launched by 05-start-services.ps1, then Kafka + ZooKeeper.
# Postgres service stays Running (managed by Windows). Pass -StopPg to stop it.
# =============================================================================
[CmdletBinding()]
param(
    [switch]$StopPg,
    [string]$PgService = 'postgresql-x64-15'
)

$LogDir = "$PSScriptRoot\logs"
$pidsFile = "$LogDir\pids.json"

if (Test-Path $pidsFile) {
    $records = Get-Content $pidsFile -Raw | ConvertFrom-Json
    foreach ($r in $records) {
        try {
            Stop-Process -Id $r.pid -Force -ErrorAction Stop
            Write-Host "Killed $($r.name) (pid $($r.pid))"
        } catch {
            Write-Host "  $($r.name) (pid $($r.pid)) already gone" -ForegroundColor DarkGray
        }
    }
    Remove-Item $pidsFile -Force
}

# Sweep stray child processes
Write-Host "==> Killing any remaining java/node/ws-services/ws-calculator processes..."
Get-Process java,node,ws-services,ws-calculator -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

# Kafka + ZK
Write-Host "==> Killing Kafka + ZooKeeper (Java processes already swept)..."

if ($StopPg) {
    Write-Host "==> Stopping PostgreSQL service..."
    Stop-Service $PgService -Force -ErrorAction SilentlyContinue
}

Write-Host "=== Stopped. ===" -ForegroundColor Green
