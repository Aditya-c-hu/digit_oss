# =============================================================================
# 02-build-all.ps1
# Builds: 25 Java services (mvn package), 2 Go binaries, pdf-service (npm).
# Logs: local-deploy\logs\build-<svc>.log
# Re-run any time. Skips services that already have a target/*.jar.
# =============================================================================
[CmdletBinding()]
param(
    [switch]$Force,            # rebuild even if jar exists
    [switch]$ContinueOnError   # don't stop on first failure
)

$ErrorActionPreference = if ($ContinueOnError) { 'Continue' } else { 'Stop' }

$Root      = Split-Path -Parent $PSScriptRoot
$LogDir    = "$PSScriptRoot\logs"
$Settings  = "$Root\maven-settings.xml"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

$javaSvcs = @(
    'egov-mdms-service','egov-idgen','egov-persister','egov-filestore','egov-user',
    'egov-workflow-v2','egov-location','egov-localization','egov-accesscontrol',
    'egov-common-masters','egov-enc-service','egov-indexer','egov-notification-mail',
    'egov-notification-sms','egov-otp','egov-pg-service','egov-searcher',
    'egov-url-shortening','tenant','user-otp',
    'property-services','pt-calculator-v2',
    'billing-service','collection-services','egov-apportion-service'
)

$failed = @()

# ---- Java ------------------------------------------------------------------
foreach ($svc in $javaSvcs) {
    $svcDir = Join-Path $Root $svc
    if (-not (Test-Path "$svcDir\pom.xml")) {
        Write-Host "[skip] $svc - no pom.xml" -ForegroundColor DarkYellow
        continue
    }
    $existingJar = Get-ChildItem "$svcDir\target\*.jar" -ErrorAction SilentlyContinue |
                   Where-Object { $_.Name -notmatch '\.original$' } | Select-Object -First 1
    if ($existingJar -and -not $Force) {
        Write-Host "[skip] $svc (already built: $($existingJar.Name))" -ForegroundColor DarkGray
        continue
    }

    $log = "$LogDir\build-$svc.log"
    Write-Host "[mvn] $svc -> $log" -ForegroundColor Cyan
    Push-Location $svcDir
    try {
        & mvn -U -s $Settings -f pom.xml clean package -DskipTests -B *> $log
        if ($LASTEXITCODE -ne 0) {
            Write-Host "    FAILED - see $log" -ForegroundColor Red
            $failed += $svc
            if (-not $ContinueOnError) { throw "mvn failed for $svc" }
        } else {
            Write-Host "    OK" -ForegroundColor Green
        }
    } finally {
        Pop-Location
    }
}

# ---- Go --------------------------------------------------------------------
foreach ($svc in @('ws-services','ws-calculator')) {
    $svcDir = Join-Path $Root $svc
    $outBin = "$svcDir\bin\$svc.exe"
    if ((Test-Path $outBin) -and -not $Force) {
        Write-Host "[skip] $svc (already built)" -ForegroundColor DarkGray
        continue
    }
    $log = "$LogDir\build-$svc.log"
    Write-Host "[go]  $svc -> $log" -ForegroundColor Cyan
    New-Item -ItemType Directory -Force -Path "$svcDir\bin" | Out-Null
    Push-Location $svcDir
    try {
        & go mod tidy *> $log
        & go build -buildvcs=false -ldflags='-s -w' -o $outBin "./cmd/$svc" *>> $log
        if ($LASTEXITCODE -ne 0) {
            Write-Host "    FAILED - see $log" -ForegroundColor Red
            $failed += $svc
            if (-not $ContinueOnError) { throw "go build failed for $svc" }
        } else {
            Write-Host "    OK ($outBin)" -ForegroundColor Green
        }
    } finally {
        Pop-Location
    }
}

# ---- Node (pdf-service) ----------------------------------------------------
$pdfDir = "$Root\pdf-service"
if (Test-Path "$pdfDir\package.json") {
    $log = "$LogDir\build-pdf-service.log"
    Write-Host "[npm] pdf-service -> $log" -ForegroundColor Cyan
    Push-Location $pdfDir
    try {
        & npm install --no-audit --no-fund *> $log
        if (Test-Path "$pdfDir\node_modules") {
            Write-Host "    OK" -ForegroundColor Green
        } else {
            Write-Host "    FAILED - see $log" -ForegroundColor Red
            $failed += 'pdf-service'
        }
    } finally {
        Pop-Location
    }
}

Write-Host ""
if ($failed.Count -eq 0) {
    Write-Host "=== BUILD OK - all 28 services compiled ===" -ForegroundColor Green
} else {
    Write-Host "=== BUILD COMPLETED WITH $($failed.Count) FAILURES ===" -ForegroundColor Yellow
    $failed | ForEach-Object { Write-Host "  - $_" -ForegroundColor Yellow }
    Write-Host "Open the matching local-deploy\logs\build-<svc>.log to debug." -ForegroundColor Yellow
}
