# =============================================================================
# 01-install-user-prereqs.ps1
# User-level installation of portable JDK 8 and Maven.
# Requires NO Administrator privileges and NO UAC elevation.
# =============================================================================

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$ToolsDir = "C:\Users\Aditya\tools"
$JdkZip = "$ToolsDir\jdk8.zip"
$MvnZip = "$ToolsDir\maven.zip"

Write-Host "==> Creating tools folder at $ToolsDir..." -ForegroundColor Cyan
if (-not (Test-Path $ToolsDir)) {
    New-Item -ItemType Directory -Path $ToolsDir | Out-Null
}

# ---- Download and Extract JDK 8 ---------------------------------------------
if (-not (Test-Path "$ToolsDir\jdk8")) {
    Write-Host "==> Downloading portable JDK 8 (GA release)..." -ForegroundColor Cyan
    $jdkUrl = "https://api.adoptium.net/v3/binary/latest/8/ga/windows/x64/jdk/hotspot/normal/adoptium?project=jdk"
    Invoke-WebRequest -Uri $jdkUrl -OutFile $JdkZip -UseBasicParsing

    Write-Host "==> Extracting JDK 8 to $ToolsDir\jdk8..." -ForegroundColor Cyan
    Expand-Archive -Path $JdkZip -DestinationPath "$ToolsDir\jdk8" -Force
    Remove-Item $JdkZip -Force
    Write-Host "    JDK 8 extracted successfully." -ForegroundColor Green
} else {
    Write-Host "    JDK 8 already exists in $ToolsDir\jdk8, skipping." -ForegroundColor DarkGray
}

$jdkFolder = Get-ChildItem "$ToolsDir\jdk8" -Directory | Select-Object -First 1
$jdkPath = $jdkFolder.FullName

# ---- Download and Extract Maven ---------------------------------------------
if (-not (Test-Path "$ToolsDir\maven")) {
    Write-Host "==> Downloading portable Apache Maven 3.9.6..." -ForegroundColor Cyan
    $mvnUrl = "https://archive.apache.org/dist/maven/maven-3/3.9.6/binaries/apache-maven-3.9.6-bin.zip"
    Invoke-WebRequest -Uri $mvnUrl -OutFile $MvnZip -UseBasicParsing

    Write-Host "==> Extracting Maven to $ToolsDir\maven..." -ForegroundColor Cyan
    Expand-Archive -Path $MvnZip -DestinationPath "$ToolsDir\maven" -Force
    Remove-Item $MvnZip -Force
    Write-Host "    Maven extracted successfully." -ForegroundColor Green
} else {
    Write-Host "    Maven already exists in $ToolsDir\maven, skipping." -ForegroundColor DarkGray
}

$mvnFolder = Get-ChildItem "$ToolsDir\maven" -Directory | Select-Object -First 1
$mvnPath = $mvnFolder.FullName

# ---- Configure Environment Variables for User -------------------------------
Write-Host "==> Configuring User Environment Variables..." -ForegroundColor Cyan

[Environment]::SetEnvironmentVariable('JAVA_HOME', $jdkPath, 'User')
Write-Host "    JAVA_HOME set to $jdkPath (User)" -ForegroundColor Green

[Environment]::SetEnvironmentVariable('MAVEN_HOME', $mvnPath, 'User')
Write-Host "    MAVEN_HOME set to $mvnPath (User)" -ForegroundColor Green

# Update User PATH
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$jdkBin = "$jdkPath\bin"
$mvnBin = "$mvnPath\bin"

$pathUpdated = $false
if ($userPath -notlike "*$jdkBin*") {
    $userPath = "$userPath;$jdkBin"
    $pathUpdated = $true
}
if ($userPath -notlike "*$mvnBin*") {
    $userPath = "$userPath;$mvnBin"
    $pathUpdated = $true
}

if ($pathUpdated) {
    [Environment]::SetEnvironmentVariable('Path', $userPath, 'User')
    Write-Host "    Added JDK and Maven to User PATH." -ForegroundColor Green
} else {
    Write-Host "    JDK and Maven already present in User PATH." -ForegroundColor DarkGray
}

Write-Host ""
Write-Host "=============================================================" -ForegroundColor Green
Write-Host " Portable JDK 8 & Maven setup complete." -ForegroundColor Green
Write-Host " Restart your terminal so the new Environment variables load." -ForegroundColor Yellow
Write-Host "=============================================================" -ForegroundColor Green
