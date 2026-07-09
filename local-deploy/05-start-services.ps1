# =============================================================================
# 05-start-services.ps1
# Launches all 28 application services (25 Java + 1 Node + 2 Go) as background
# processes, in dependency-priority order. Each gets a port + DB/Kafka env.
# PIDs saved to logs\pids.json so 06-stop-all.ps1 can clean up.
# =============================================================================
[CmdletBinding()]
param(
    [string]$PgHost   = 'localhost',
    [int]   $PgPort   = 5433,
    [string]$PgUser   = 'postgres',
    [string]$PgPass   = 'postgres',
    [string]$DbName   = 'rainmaker',
    [string]$Kafka    = 'localhost:9092',
    [int]   $Xmx      = 192,    # MB per JVM
    [int]   $Xms      = 64,
    [switch]$Minimal
)

$ErrorActionPreference = 'Stop'
$Root   = Split-Path -Parent $PSScriptRoot
$LogDir = "$PSScriptRoot\logs"
New-Item -ItemType Directory -Force -Path $LogDir | Out-Null

# ---- Service registry -------------------------------------------------------
# Priority groups: lower starts first.
$javaSvcs = @(
    @{ name='egov-mdms-service';    port=8094; pri=5 },
    @{ name='egov-user';            port=8081; pri=10 },
    @{ name='egov-idgen';           port=8088; pri=10 },
    @{ name='tenant';               port=8200; pri=10 },
    @{ name='egov-localization';    port=8087; pri=10 },
    @{ name='egov-filestore';       port=8083; pri=10 },
    @{ name='egov-location';        port=8084; pri=10 },
    @{ name='egov-enc-service';     port=8089; pri=10 },
    @{ name='egov-common-masters';  port=8086; pri=10 },
    @{ name='egov-otp';             port=8096; pri=10 },
    @{ name='egov-url-shortening';  port=8099; pri=10 },
    @{ name='egov-searcher';        port=8098; pri=10 },
    @{ name='egov-pg-service';      port=8097; pri=10 },
    @{ name='egov-persister';       port=8082; pri=20 },
    @{ name='egov-indexer';         port=8092; pri=20 },
    @{ name='egov-notification-mail'; port=8093; pri=20 },
    @{ name='egov-notification-sms'; port=8095; pri=20 },
    @{ name='egov-accesscontrol';   port=8085; pri=20 },
    @{ name='user-otp';             port=8201; pri=20 },
    @{ name='egov-workflow-v2';     port=8290; pri=30 },
    @{ name='property-services';    port=8280; pri=40 },
    @{ name='pt-calculator-v2';     port=8281; pri=50 },
    @{ name='billing-service';      port=8202; pri=50 },
    @{ name='egov-apportion-service'; port=8204; pri=50 },
    @{ name='collection-services';  port=8203; pri=60 }
)

$essentials = @('egov-mdms-service', 'egov-user', 'egov-idgen', 'egov-location', 'egov-enc-service', 'egov-persister', 'egov-workflow-v2', 'property-services', 'billing-service', 'collection-services')
if ($Minimal) {
    Write-Host "RAM-Save mode active! Filtering to only start essential integration services." -ForegroundColor Yellow
    $javaSvcs = $javaSvcs | Where-Object { $essentials -contains $_.name }
}

$jdbc = "jdbc:postgresql://$PgHost`:$PgPort/$DbName"
$MdmsPath = Join-Path $Root "local-deploy\mdms-data\data"
$MasterConfigPath = Join-Path $Root "local-deploy\mdms-data\master-config.json"
$MasterConfigUrl = "file:///" + $MasterConfigPath.Replace('\', '/')

# ---- JVM env shared across all Spring Boot services -------------------------
$commonEnv = @{
    'SPRING_DATASOURCE_URL'      = $jdbc
    'SPRING_DATASOURCE_USERNAME' = $PgUser
    'SPRING_DATASOURCE_PASSWORD' = $PgPass
    'SPRING_KAFKA_BOOTSTRAP_SERVERS' = $Kafka
    'KAFKA_CONFIG_BOOTSTRAP_SERVER_CONFIG' = $Kafka
    'KAFKA_BOOTSTRAP_SERVER_CONFIG'        = $Kafka
    'EGOV_MDMS_HOST'             = 'http://localhost:8094/'
    'MDMS_SERVICE_HOST'          = 'http://localhost:8094/'
    'EGOV_SERVICES_EGOV_MDMS_HOSTNAME' = 'http://localhost:8094/'
    'EGOV_USER_HOST'             = 'http://localhost:8081/'
    'EGOV_SERVICES_EGOV_USER_HOSTNAME' = 'http://localhost:8081/'
    'EGOV_IDGEN_HOST'            = 'http://localhost:8088/'
    'EGOV_LOCALIZATION_HOST'     = 'http://localhost:8087/'
    'EGOV_LOCALISATION_HOST'     = 'http://localhost:8087/'
    'EGOV_LOCALISATION_HOSTNAME' = 'http://localhost:8087/'
    'EGOV_WORKFLOW_HOST'         = 'http://localhost:8290/'
    'EGOV_FILESTORE_HOST'        = 'http://localhost:8083/'
    'EGOV_LOCATION_HOST'         = 'http://localhost:8084/'
    'EGOV_ACCESSCONTROL_HOST'    = 'http://localhost:8085/'
    'EGOV_BILLING_HOST'          = 'http://localhost:8202/'
    'EGOV_BILLINGSERVICE_HOST'   = 'http://localhost:8202/'
    'EGBS_HOST'                  = 'http://localhost:8202/'
    'EGOV_COLLECTION_HOST'       = 'http://localhost:8203/'
    'EGOV_COLLECTIONSERVICE_HOST'= 'http://localhost:8203/'
    'EGOV_PROPERTY_HOST'         = 'http://localhost:8280/'
    'EGOV_PT_HOST'               = 'http://localhost:8280/'
    'EGOV_PT_REGISTRY_HOST'      = 'http://localhost:8280/'
    'EGOV_PT_CALCULATOR_HOST'    = 'http://localhost:8281/'
    'EGOV_ASSESSMENTSERVICE_HOST'= 'http://localhost:8281/'
    'EGOV_CALCULATION_HOST'      = 'http://localhost:8281/'
    'EGOV_MDMS_CONF_PATH'        = $MdmsPath
    'MASTERS_CONFIG_URL'         = $MasterConfigUrl
    'EGOV_ENC_HOST'              = 'http://localhost:8089/'
    'EGOV_ENC_SIGN_HOST'         = 'http://localhost:8089/'
}

function Start-Java {
    param($svc)
    $svcDir = Join-Path $Root $svc.name
    $jar = Get-ChildItem "$svcDir\target\*.jar" -ErrorAction SilentlyContinue |
           Where-Object { $_.Name -notmatch '\.original$' } | Select-Object -First 1
    if (-not $jar) {
        Write-Host "[skip] $($svc.name) - no jar in $svcDir\target. Build it first." -ForegroundColor Yellow
        return $null
    }

    foreach ($k in $commonEnv.Keys) { Set-Item "Env:$k" $commonEnv[$k] }

    $jvmArgs = "-Xmx${Xmx}m -Xms${Xms}m -Dserver.port=$($svc.port) -Dspring.datasource.url=$jdbc -Dspring.datasource.username=$PgUser -Dspring.datasource.password=$PgPass -Dspring.kafka.bootstrap.servers=$Kafka -Dflyway.enabled=false -Dspring.flyway.enabled=false -Degov.enc.host=http://localhost:8089/ -Degov.enc.sign.host=http://localhost:8089/ -Dmdms.host=http://localhost:8094/ -Degov.mdms.host=http://localhost:8094/ -Degov.services.accesscontrol.host=http://localhost:8085/ -Degov.localization.host=http://localhost:8087/ -Degov.localisation.host=http://localhost:8087/ -Degov.user.host=http://localhost:8081/ -Degov.idgen.host=http://localhost:8088/ -Degov.location.host=http://localhost:8084/ -Degov.billingservice.host=http://localhost:8202/ -Degov.collectionservice.host=http://localhost:8203/ -Degov.workflow.host=http://localhost:8290/ -Degov.workflow.v2.host=http://localhost:8290/ -Duser.service.host=http://localhost:8081/ -Degov.persist.yml.repo.path=C:/Users/Aditya/Downloads/DIGIT-OSS-master/municipal-services-go/local-deploy/configs/egov-persister"
    $log = "$LogDir\$($svc.name).log"
    $cmd = "java $jvmArgs -jar `"$($jar.FullName)`" > `"$log`" 2>&1"

    Write-Host "[java] $($svc.name) -> port $($svc.port) -> $log" -ForegroundColor Cyan
    $p = Start-Process -FilePath 'cmd.exe' -ArgumentList '/c', $cmd -PassThru -WindowStyle Hidden
    return @{ name=$svc.name; pid=$p.Id; port=$svc.port }
}

function Start-Go {
    param([string]$Name, [int]$Port, [hashtable]$Env_)
    $bin = "$Root\$Name\bin\$Name.exe"
    if (-not (Test-Path $bin)) {
        Write-Host "[skip] $Name - binary missing: $bin" -ForegroundColor Yellow
        return $null
    }
    
    $setCmds = @()
    foreach ($k in $Env_.Keys) {
        $val = "$($Env_[$k])".Trim()
        $setCmds += "set $k=$val"
    }
    $setPrefix = $setCmds -join "&&"
    
    $log = "$LogDir\$Name.log"
    Write-Host "[go]   $Name -> port $Port -> $log" -ForegroundColor Cyan
    $cmd = "$setPrefix&&`"$bin`" > `"$log`" 2>&1"
    $p = Start-Process -FilePath 'cmd.exe' -ArgumentList '/c', $cmd -PassThru -WindowStyle Hidden
    return @{ name=$Name; pid=$p.Id; port=$Port }
}

function Start-Node {
    $pdfDir = "$Root\pdf-service"
    $log = "$LogDir\pdf-service.log"
    Write-Host "[node] pdf-service -> port 8080 -> $log" -ForegroundColor Cyan
    $env:PORT = '8080'
    $env:DATA_CONFIG_URLS   = 'https://raw.githubusercontent.com/egovernments/egov-pdf/master/data-config/receipt.json'
    $env:FORMAT_CONFIG_URLS = 'https://raw.githubusercontent.com/egovernments/egov-pdf/master/format-config/receipt.json'
    $entry = if (Test-Path "$pdfDir\dist\index.js") { 'dist' } else { 'src/index.js' }
    $p = Start-Process -FilePath 'cmd.exe' `
        -ArgumentList '/c', "node `"$pdfDir\$entry`" > `"$log`" 2>&1" `
        -PassThru -WindowStyle Hidden -WorkingDirectory $pdfDir
    return @{ name='pdf-service'; pid=$p.Id; port=8080 }
}

# ---- Boot in priority order ------------------------------------------------
$pids = @()

foreach ($pri in ($javaSvcs.pri | Sort-Object -Unique)) {
    $batch = $javaSvcs | Where-Object { $_.pri -eq $pri }
    Write-Host "--- priority $pri ($($batch.Count) services) ---" -ForegroundColor Magenta
    foreach ($s in $batch) {
        $rec = Start-Java $s
        if ($rec) { $pids += $rec }
    }
    if ($pri -eq 5) {
        Write-Host "Waiting for egov-mdms-service to be fully up and listening on port 8094..." -ForegroundColor Yellow
        $elapsed = 0
        $timeout = 180
        while ($elapsed -lt $timeout) {
            $conn = Test-NetConnection -ComputerName 'localhost' -Port 8094 -WarningAction SilentlyContinue
            if ($conn.TcpTestSucceeded) {
                Write-Host "egov-mdms-service is UP and listening!" -ForegroundColor Green
                break
            }
            Start-Sleep -Seconds 3
            $elapsed += 3
        }
    } else {
        Write-Host "Sleeping 8s before next priority group..."
        Start-Sleep -Seconds 8
    }
}

# pdf-service after Kafka-dependent batch
if (-not $Minimal) {
    $rec = Start-Node
    if ($rec) { $pids += $rec }
}

# Go services last
$goEnv = @{
    'SERVER_PORT'   = '8090'
    'DB_HOST'       = $PgHost
    'DB_PORT'       = "$PgPort"
    'DB_USER'       = $PgUser
    'DB_PASSWORD'   = $PgPass
    'DB_NAME'       = $DbName
    'KAFKA_BROKERS' = $Kafka
    'KAFKA_GROUP_ID'= 'egov-ws-services'
    'IS_EXTERNAL_WORKFLOW_ENABLED' = 'true'
    'IS_IDGEN_ENABLED'             = 'true'
    'IS_MDMS_ENABLED'              = 'true'
    'IS_PROPERTY_ENABLED'          = 'true'
    'IS_USER_ENABLED'              = 'true'
    'IS_BILLING_ENABLED'           = 'true'
    'EGOV_IDGEN_HOST'              = 'http://localhost:8088/'
    'EGOV_MDMS_HOST'               = 'http://localhost:8094/'
    'EGOV_PROPERTY_HOST'           = 'http://localhost:8280/'
    'EGOV_USER_HOST'               = 'http://localhost:8081/'
    'WORKFLOW_CONTEXT_PATH'        = 'http://localhost:8290/'
    'EGOV_WS_CALCULATION_HOST'     = 'http://localhost:8091/'
    'EGOV_BILLING_SERVICE_HOST'    = 'http://localhost:8202/'
    'EGOV_COLLECTION_HOST'         = 'http://localhost:8203/'
    'EGOV_IDGEN_WCAPID_NAME'       = 'waterservice.application.id'
    'EGOV_IDGEN_WCID_NAME'         = 'waterservice.connection.id'
}
$rec = Start-Go -Name 'ws-services' -Port 8090 -Env_ $goEnv
if ($rec) { $pids += $rec }

$goEnv['SERVER_PORT']    = '8091'
$goEnv['KAFKA_GROUP_ID'] = 'egov-ws-calculator'
$rec = Start-Go -Name 'ws-calculator' -Port 8091 -Env_ $goEnv
if ($rec) { $pids += $rec }

# ---- Persist PIDs ----------------------------------------------------------
$pids | ConvertTo-Json | Set-Content -Path "$LogDir\pids.json" -Encoding ASCII

Write-Host ""
Write-Host "=== Launched $($pids.Count) services. PIDs in $LogDir\pids.json ===" -ForegroundColor Green
Write-Host "Tail any service:  Get-Content $LogDir\<svc>.log -Wait -Tail 50"
Write-Host "Wait ~2-3 minutes for all JVMs to register on their ports." -ForegroundColor Yellow
