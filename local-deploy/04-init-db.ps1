# =============================================================================
# 04-init-db.ps1
# Creates the rainmaker DB and applies DDL schema migrations inside Docker container.
# Resolves relative SQL inclusions (\ir) on the host and pipes them directly.
# =============================================================================
[CmdletBinding()]
param(
    [string]$PgUser = 'postgres',
    [string]$DbName = 'rainmaker'
)

$ErrorActionPreference = 'Stop'
$Root = Split-Path -Parent $PSScriptRoot

Write-Host "==> Checking if database '$DbName' exists inside Container..." -ForegroundColor Cyan
$exists = (docker exec -i ws-postgres psql -U $PgUser -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$DbName'") 2>$null
if ($exists -eq '1') {
    Write-Host "    Database '$DbName' already exists." -ForegroundColor Green
} else {
    Write-Host "==> Creating database '$DbName' inside Container..." -ForegroundColor Cyan
    & docker exec -i ws-postgres psql -U $PgUser -d postgres -v ON_ERROR_STOP=1 -c "CREATE DATABASE $DbName;"
}

$initSql = "$Root\db\init.sql"
if (Test-Path $initSql) {
    Write-Host "==> Resolving relative imports in $initSql and applying DDL..." -ForegroundColor Cyan
    $fullSql = ""
    $lines = Get-Content $initSql
    foreach ($line in $lines) {
        if ($line.Trim() -like '\ir *') {
            $relPath = $line.Trim().Substring(4).Trim()
            # Resolve path relative to the db/ folder
            $targetPath = [System.IO.Path]::GetFullPath([System.IO.Path]::Combine("$Root\db", $relPath))
            if (Test-Path $targetPath) {
                Write-Host "    Inlining migration DDL: $relPath" -ForegroundColor DarkGray
                $fullSql += "`n-- --- Inlined from $relPath ---`n"
                $fullSql += Get-Content $targetPath -Raw
                $fullSql += "`n"
            } else {
                Write-Host "    WARNING: Could not find target migration file: $targetPath" -ForegroundColor Yellow
            }
        } else {
            $fullSql += $line + "`n"
        }
    }

    # Stream the consolidated SQL directly into postgres inside the container
    $fullSql | docker exec -i ws-postgres psql -U $PgUser -d $DbName -v ON_ERROR_STOP=1 | Out-Null
    Write-Host "    Database schemas perfectly applied and bootstrapped!" -ForegroundColor Green
} else {
    Write-Host "    No db\init.sql found - skipping schema bootstrap." -ForegroundColor Yellow
}

Write-Host "=== DB ready: jdbc:postgresql://localhost:5433/$DbName ===" -ForegroundColor Green
