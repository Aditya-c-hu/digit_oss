# =============================================================================
# 07-smoke-test.ps1
# End-to-end WS workflow against locally-running services.
# Run AFTER 05-start-services.ps1 and after JVMs settle (~2-3 min).
# =============================================================================
[CmdletBinding()]
param(
    [string]$Tenant = 'pb.amritsar'
)

$ErrorActionPreference = 'Continue'
$ProgressPreference   = 'SilentlyContinue'
$Samples = "$PSScriptRoot\samples"

function Hit {
    param([string]$Url, [string]$Method = 'GET', $Body = $null)
    try {
        $params = @{ Uri = $Url; Method = $Method; TimeoutSec = 15 }
        if ($Body) {
            $params.Body = $Body
            $params.ContentType = 'application/json'
        }
        $r = Invoke-WebRequest @params -UseBasicParsing
        return @{ ok = $true; code = [int]$r.StatusCode; body = $r.Content }
    } catch {
        return @{ ok = $false; code = $_.Exception.Response.StatusCode.value__; body = $_.ToString() }
    }
}

function Step {
    param([string]$Label, [hashtable]$Result, [int[]]$ExpectCodes = @(200,201,202))
    $emoji = if ($Result.ok -and $ExpectCodes -contains $Result.code) { '[OK]' } else { '[FAIL]' }
    $color = if ($emoji -eq '[OK]') { 'Green' } else { 'Red' }
    Write-Host "$emoji $Label  HTTP $($Result.code)" -ForegroundColor $color
    if ($emoji -eq '[FAIL]') {
        Write-Host "      $($Result.body.Substring(0, [Math]::Min(300, $Result.body.Length)))" -ForegroundColor DarkGray
    }
}

# ---- 1. Health checks ------------------------------------------------------
Write-Host "==> 1. Health checks" -ForegroundColor Cyan
$endpoints = @{
    'egov-mdms-service' = 'http://localhost:8094/egov-mdms-service/health'
    'egov-user'         = 'http://localhost:8081/user/health'
    'egov-idgen'        = 'http://localhost:8088/egov-idgen/health'
    'egov-workflow-v2'  = 'http://localhost:8290/egov-workflow-v2/health'
    'property-services' = 'http://localhost:8280/property-services/health'
    'billing-service'   = 'http://localhost:8202/billing-service/health'
    'collection-services' = 'http://localhost:8203/collection-services/health'
    'ws-services'       = 'http://localhost:8090/health'
    'ws-calculator'     = 'http://localhost:8091/health'
}
foreach ($n in $endpoints.Keys) {
    Step $n (Hit $endpoints[$n]) @(200,404)  # 404 still means HTTP layer alive
}

# ---- 2. MDMS sanity --------------------------------------------------------
Write-Host ""
Write-Host "==> 2. MDMS _search" -ForegroundColor Cyan
$mdmsBody = @{
    RequestInfo = @{ apiId='ws'; ver='1.0'; action='_search'; authToken='test' }
    MdmsCriteria = @{
        tenantId = $Tenant
        moduleDetails = @(@{
            moduleName = 'ws-services-masters'
            masterDetails = @(@{ name = 'Connection' })
        })
    }
} | ConvertTo-Json -Depth 6
Step 'MDMS search' (Hit 'http://localhost:8094/egov-mdms-service/v1/_search' POST $mdmsBody)

# ---- 3. Create property ----------------------------------------------------
Write-Host ""
Write-Host "==> 3. Create property" -ForegroundColor Cyan
$propBody = Get-Content "$Samples\property-create.json" -Raw
$propRes = Hit 'http://localhost:8280/property-services/property/_create' POST $propBody
Step 'Property _create' $propRes
$propertyId = $null
if ($propRes.ok) {
    try { $propertyId = (ConvertFrom-Json $propRes.body).Properties[0].propertyId } catch {}
    Write-Host "    propertyId = $propertyId" -ForegroundColor Yellow
}

# ---- 4. Create WS connection ----------------------------------------------
if ($propertyId) {
    Write-Host ""
    Write-Host "==> 4. Create WS connection" -ForegroundColor Cyan
    Write-Host "    Waiting 3s for persister to commit property to DB..." -ForegroundColor DarkGray
    Start-Sleep -Seconds 3
    $wsBody = (Get-Content "$Samples\ws-create.json" -Raw) -replace 'REPLACE_WITH_PROPERTY_ID', $propertyId
    $wsRes = Hit 'http://localhost:8090/ws-services/wc/_create' POST $wsBody
    Step 'WS _create' $wsRes
    $appNo = $null
    if ($wsRes.ok) {
        try { $appNo = (ConvertFrom-Json $wsRes.body).WaterConnection[0].applicationNo } catch {}
        Write-Host "    applicationNo = $appNo" -ForegroundColor Yellow
    }

    # ---- 5. Search ---------------------------------------------------------
    Write-Host ""
    Write-Host "==> 5. WS search" -ForegroundColor Cyan
    $searchBody = @{
        RequestInfo = @{ apiId='ws'; ver='1.0'; action='_search'; authToken='test-token' }
    } | ConvertTo-Json
    Step 'WS _search' (Hit "http://localhost:8090/ws-services/wc/_search?tenantId=$Tenant" POST $searchBody)

    # ---- 6. Estimate -------------------------------------------------------
    if ($appNo) {
        Write-Host ""
        Write-Host "==> 6. WS calculator estimate" -ForegroundColor Cyan
        $estBody = (Get-Content "$Samples\ws-estimate.json" -Raw) -replace 'REPLACE_WITH_APPLICATION_NO', $appNo
        Step 'Calculator _estimate' (Hit 'http://localhost:8091/ws-calculator/waterCalculator/_estimate' POST $estBody)
    }
}

Write-Host ""
Write-Host "=== Smoke test done. Inspect FAILs above. ===" -ForegroundColor Cyan
Write-Host "For a full Postman walk:  Import municipal-services-go\postman\*.json into Postman" -ForegroundColor Yellow
