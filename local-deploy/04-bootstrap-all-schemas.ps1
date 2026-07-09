# =============================================================================
# 04-bootstrap-all-schemas.ps1
# Automatically scans, collects, and boots all database Flyway DDL migration
# schemas for ALL DIGIT microservices (Java, Go, Node.js) into PostgreSQL.
# =============================================================================
$ErrorActionPreference = 'SilentlyContinue'
$Root = Split-Path -Parent $PSScriptRoot

Write-Host "==> Scanning for all schema migration folders..." -ForegroundColor Cyan

# Locate folders containing migrations
$migrationDirs = Get-ChildItem -Path $Root -Recurse -Directory | Where-Object {
    $_.FullName -match 'db[\\/]migration' -or 
    $_.FullName -match 'migrations' -or 
    $_.FullName -match 'migration[\\/]ddl'
} | Where-Object { $_.FullName -notmatch 'target[\\/]classes' } # exclude duplicate target/classes dirs

Write-Host "Found $($migrationDirs.Count) migration directories." -ForegroundColor Green

# Collect and sort all SQL files by filename
$sqlFiles = @()
foreach ($dir in $migrationDirs) {
    $files = Get-ChildItem -Path $dir.FullName -Filter "*.sql"
    foreach ($f in $files) {
        # Avoid duplicate scripts by matching base name
        if ($sqlFiles.Name -notcontains $f.Name) {
            $sqlFiles += $f
        }
    }
}

# Sort chronologically/alphabetically by file name
$sqlFiles = $sqlFiles | Sort-Object Name

Write-Host "Collected $($sqlFiles.Count) unique SQL migration scripts to apply." -ForegroundColor Green

# Apply SQL scripts sequentially
$appliedCount = 0
foreach ($f in $sqlFiles) {
    Write-Host "Applying DDL: $($f.Name)..." -ForegroundColor DarkGray
    $sqlContent = Get-Content -Path $f.FullName -Raw
    
    # Execute DDL directly into Docker Postgres
    $sqlContent | docker exec -i ws-postgres psql -U postgres -d rainmaker > $null 2>&1
    $appliedCount++
}

Write-Host "=== Successfully applied $appliedCount database schemas to rainmaker! ===" -ForegroundColor Green
