# ============================================
# RINCO dev orchestration script — PowerShell
# Loop WS-A — mirrors scripts/dev.sh for Windows / PowerShell hosts
# ============================================
[CmdletBinding()]
param(
    [switch]$SkipSeed,
    [switch]$SkipFrontends,
    [int]$WaitInfraSeconds = 120,
    [int]$WaitServiceSeconds = 60
)

$ErrorActionPreference = 'Continue'

# ------------------------------------------------------------ paths
$ScriptDir   = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = (Resolve-Path "$ScriptDir\..").Path
$InfraDir    = Join-Path $ProjectRoot "infra"
$LogDir      = Join-Path $ProjectRoot "logs"
$PidsDir     = Join-Path $ProjectRoot ".dev\pids"

if (-not (Test-Path $LogDir))  { New-Item -ItemType Directory -Path $LogDir  -Force | Out-Null }
if (-not (Test-Path $PidsDir)) { New-Item -ItemType Directory -Path $PidsDir -Force | Out-Null }

# ------------------------------------------------------------ helpers
function Step($msg) { Write-Host "[$((Get-Date).ToString('HH:mm:ss'))] $msg" -ForegroundColor Cyan }
function Ok($msg)   { Write-Host "[OK] $msg" -ForegroundColor Green }
function Warn($msg) { Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Fail($msg) { Write-Host "[FAIL] $msg" -ForegroundColor Red }

function Wait-ForUrl {
    param([string]$Url, [string]$Label, [int]$MaxSeconds = 60)
    $waited = 0
    while ($waited -lt $MaxSeconds) {
        try {
            $r = Invoke-WebRequest -Uri $Url -TimeoutSec 2 -UseBasicParsing -ErrorAction Stop
            if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 400) { return $true }
        } catch {
            # not yet ready
        }
        Start-Sleep -Seconds 2
        $waited += 2
    }
    return $false
}

function Get-ComposeCmd {
    if ($null -ne (Get-Command docker -ErrorAction SilentlyContinue)) {
        try {
            $v = docker compose version 2>&1
            if ($LASTEXITCODE -eq 0) { return @('docker', 'compose') }
        } catch {}
    }
    if (Get-Command docker-compose -ErrorAction SilentlyContinue) { return @('docker-compose') }
    return @('docker', 'compose')
}

$ComposeCmd = Get-ComposeCmd

# ------------------------------------------------------------ .env
$EnvFile = Join-Path $ProjectRoot ".env"
if (-not (Test-Path $EnvFile)) {
    Step "creating default .env (PASETO keys etc.)"
    @'
POSTGRES_PASSWORD=rinco_dev_password
CLICKHOUSE_PASSWORD=rinco_dev_password
VALKEY_PASSWORD=rinco_dev_password
MONGO_PASSWORD=rinco_dev_password
MINIO_ROOT_PASSWORD=rinco_dev_password
PASETO_KEY_CURRENT=0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef
PASETO_KEY_PREVIOUS=fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210
ADMIN_API_KEY=dev_admin_key_change_me
MEILI_MASTER_KEY=masterKey
'@ | Out-File -FilePath $EnvFile -Encoding utf8
}

# ------------------------------------------------------------ service table
$Services = @(
    @{ Name = 'auth-service';           Dir = 'services/auth-service';           URL = 'http://localhost:8081/health'; Lang = 'go'     },
    @{ Name = 'tenant-service';         Dir = 'services/tenant-service';         URL = 'http://localhost:8082/health'; Lang = 'go'     },
    @{ Name = 'crm-service';            Dir = 'services/crm-service';            URL = 'http://localhost:8083/health'; Lang = 'go'     },
    @{ Name = 'dynamic-model-service';  Dir = 'services/dynamic-model-service';  URL = 'http://localhost:8084/health'; Lang = 'go'     },
    @{ Name = 'lead-service';           Dir = 'services/lead-service';           URL = 'http://localhost:8085/health'; Lang = 'go'     },
    @{ Name = 'landing-service';        Dir = 'services/landing-service';        URL = 'http://localhost:8086/health'; Lang = 'go'     },
    @{ Name = 'email-service';          Dir = 'services/email-service';          URL = 'http://localhost:8087/health'; Lang = 'go'     },
    @{ Name = 'notification-service';   Dir = 'services/notification-service';   URL = 'http://localhost:8088/health'; Lang = 'go'     },
    @{ Name = 'billing-service';        Dir = 'services/billing-service';        URL = 'http://localhost:8095/health'; Lang = 'go'     },
    @{ Name = 'observability-service';  Dir = 'services/observability-service';  URL = 'http://localhost:8096/health'; Lang = 'go'     },
    @{ Name = 'search-service';         Dir = 'services/search-service';         URL = 'http://localhost:8097/health'; Lang = 'go'     },
    @{ Name = 'meta-capi-service';      Dir = 'services/meta-capi-service';      URL = 'http://localhost:8098/health'; Lang = 'go'     },
    @{ Name = 'analytics-service';      Dir = 'services/analytics-service';      URL = 'http://localhost:8099/health'; Lang = 'go'     },
    @{ Name = 'ai-sre';                 Dir = 'services/ai-sre';                 URL = 'http://localhost:8090/health'; Lang = 'python' },
    @{ Name = 'lead-scoring';           Dir = 'services/lead-scoring';           URL = 'http://localhost:8091/health'; Lang = 'python' },
    @{ Name = 'rag-chatbot';            Dir = 'services/rag-chatbot';            URL = 'http://localhost:8092/health'; Lang = 'python' },
    @{ Name = 'stt-service';            Dir = 'services/stt-service';            URL = 'http://localhost:8094/health'; Lang = 'python' },
    @{ Name = 'recording-service';      Dir = 'services/recording-service';      URL = 'http://localhost:8093/health'; Lang = 'rust'   },
    @{ Name = 'chat-engine';            Dir = 'services/chat-engine';            URL = 'http://localhost:8101/health'; Lang = 'rust'   },
    @{ Name = 'webrtc-sfu';             Dir = 'services/webrtc-sfu';             URL = 'http://localhost:8102/health'; Lang = 'rust'   }
)

$Frontends = @(
    @{ Name = 'landing';      Dir = 'frontend/landing';      URL = 'http://localhost:3000/health' },
    @{ Name = 'admin-portal'; Dir = 'frontend/admin-portal'; URL = 'http://localhost:3001/health' },
    @{ Name = 'tenant-site';  Dir = 'frontend/tenant-site';  URL = 'http://localhost:3002/health' },
    @{ Name = 'meeting-ui';   Dir = 'frontend/meeting-ui';   URL = 'http://localhost:3003/health' }
)

# ------------------------------------------------------------ 1. docker network
Step "ensuring rinco-network exists"
try {
    docker network inspect rinco-network 2>&1 | Out-Null
} catch {
    docker network create --driver bridge --subnet=172.25.0.0/16 rinco-network | Out-Null
}

# ------------------------------------------------------------ 2. infra compose
Step "starting infrastructure stack"
Push-Location $InfraDir
try {
    & $ComposeCmd[0] $(if ($ComposeCmd.Count -gt 1) { $ComposeCmd[1] }) -f docker-compose.yml up -d
} finally { Pop-Location }

# ------------------------------------------------------------ 3. wait for infra
Step "waiting for infra health (Postgres + Valkey + NATS)"
$waited = 0
while ($waited -lt $WaitInfraSeconds) {
    $pg = $false; $vk = $false; $nats = $false
    try {
        docker exec rinco-postgres pg_isready -U rinco 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { $pg = $true }
    } catch {}
    try {
        docker exec rinco-valkey valkey-cli ping 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { $vk = $true }
    } catch {}
    try {
        docker exec rinco-nats nats-server --version 2>&1 | Out-Null
        if ($LASTEXITCODE -eq 0) { $nats = $true }
    } catch {}
    if ($pg -and $vk -and $nats) {
        Ok "infra healthy"
        break
    }
    Start-Sleep -Seconds 5
    $waited += 5
}

# ------------------------------------------------------------ 4. migrate + seed
if (-not $SkipSeed) {
    Step "running migrations"
    $migrateScript = Join-Path $ScriptDir "migrate.sh"
    if (Test-Path $migrateScript) {
        bash.exe $migrateScript
    } else {
        Warn "scripts/migrate.sh missing — skipping"
    }

    Step "seeding databases"
    $seedScript = Join-Path $ScriptDir "seed-all.ps1"
    if (Test-Path $seedScript) {
        & $seedScript
    } else {
        Warn "scripts/seed-all.ps1 missing — skipping"
    }
}

# ------------------------------------------------------------ 5. services compose
Step "starting backend services (docker compose)"
Push-Location $InfraDir
try {
    & $ComposeCmd[0] $(if ($ComposeCmd.Count -gt 1) { $ComposeCmd[1] }) -f docker-compose.services.yml up -d
} finally { Pop-Location }

# ------------------------------------------------------------ 6. wait for health
Step "waiting for backend /health endpoints"
foreach ($svc in $Services) {
    if (Wait-ForUrl -Url $svc.URL -Label $svc.Name -MaxSeconds $WaitServiceSeconds) {
        Ok "$($svc.Name) ($($svc.Lang)) -> $($svc.URL)"
    } else {
        Warn "$($svc.Name) not healthy within $WaitServiceSeconds s"
    }
}

# ------------------------------------------------------------ 7. frontends
if (-not $SkipFrontends) {
    Step "starting 4 frontend dev servers (background)"
    foreach ($fe in $Frontends) {
        $log = Join-Path $LogDir "$($fe.Name).log"
        $pidf = Join-Path $PidsDir "$($fe.Name).pid"
        $dir = Join-Path $ProjectRoot $fe.Dir
        if (-not (Test-Path (Join-Path $dir "package.json"))) {
            Warn "$($fe.Name): package.json missing"
            continue
        }
        Push-Location $dir
        try {
            $cmd = if (Get-Command bun -ErrorAction SilentlyContinue) { 'bun' } else { 'npm' }
            $args = @('run', 'dev')
            $proc = Start-Process -FilePath $cmd -ArgumentList $args `
                -RedirectStandardOutput $log `
                -RedirectStandardError  $log `
                -NoNewWindow -PassThru
            $proc.Id | Out-File -FilePath $pidf -Encoding ascii
            Ok "$($fe.Name) started (pid $($proc.Id)) -> $log"
        } finally {
            Pop-Location
        }
    }
}

# ------------------------------------------------------------ 8. status table
Step "status summary"
Write-Host ""
Write-Host ("{0,-22} {1,-8} {2,-44}" -f "service", "lang", "url")
Write-Host ("{0,-22} {1,-8} {2,-44}" -f "----------------------", "------", "--------------------------------------------")
foreach ($svc in $Services) {
    $healthy = Wait-ForUrl -Url $svc.URL -Label $svc.Name -MaxSeconds 2
    $glyph = if ($healthy) { "[OK]" } else { "[--]" }
    $color = if ($healthy) { "Green" } else { "Red" }
    Write-Host ("{0} {1,-20} {2,-8} {3}" -f $glyph, $svc.Name, $svc.Lang, $svc.URL) -ForegroundColor $color
}

Write-Host ""
Write-Host ("{0,-22} {1,-44}" -f "frontend", "url")
Write-Host ("{0,-22} {1,-44}" -f "----------------------", "--------------------------------------------")
foreach ($fe in $Frontends) {
    $pidf = Join-Path $PidsDir "$($fe.Name).pid"
    $alive = $false
    if (Test-Path $pidf) {
        $pid = Get-Content $pidf -ErrorAction SilentlyContinue
        if ($pid -and (Get-Process -Id $pid -ErrorAction SilentlyContinue)) { $alive = $true }
    }
    $glyph = if ($alive) { "[OK]" } else { "[--]" }
    $color = if ($alive) { "Green" } else { "Red" }
    Write-Host ("{0} {1,-20} {2}" -f $glyph, $fe.Name, $fe.URL) -ForegroundColor $color
}

Write-Host ""
Ok "dev stack up. tail logs with: Get-Content logs/<service>.log -Wait"
Ok "stop everything with: scripts/stop-all.ps1"
