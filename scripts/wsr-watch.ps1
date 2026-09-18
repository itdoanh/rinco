# ============================================================
# WS-R Health Watcher — RINCO Platform
# Periodically checks all backend + frontend health endpoints,
# logs results, restarts unhealthy containers, and produces
# a daily summary.
#
# Usage:   .\wsr-watch.ps1
# Output:  logs/wsr-watch-<date>.log
# PID:     logs/wsr-watch.pid
# ============================================================

param(
    [int]$IntervalSeconds = 60,
    [int]$RestartThreshold = 2,      # restart after 2 consecutive failures
    [int]$CrashLoopMinutes = 5       # restart k8s pod if CrashLoop > 5 min
)

$ErrorActionPreference = "Continue"
$BaseDir = "C:\code\RINCO"
$LogDir = "$BaseDir\logs"
$LogPrefix = "wsr-watch"

# Ensure log dir
if (-not (Test-Path $LogDir)) {
    New-Item -ItemType Directory -Path $LogDir -Force | Out-Null
}

$DateStr = Get-Date -Format "yyyyMMdd"
$LogFile = "$LogDir\${LogPrefix}-$DateStr.log"
$SummaryFile = "$LogDir\${LogPrefix}-summary-$DateStr.txt"
$PidFile = "$LogDir\${LogPrefix}.pid"

# Record PID
$Pid = $PID
$Pid | Out-File -FilePath $PidFile -Encoding utf8

# ---- Service definitions ----
# Each entry: @{ Name; HealthUrl; ContainerName; Type (docker|k8s); Port }
$Services = @(
    # --- Go Services ---
    @{ Name = "auth-service";           HealthUrl = "http://localhost:8081/health";    Container = "rinco-auth-service";           Type = "docker"; Port = 8081 },
    @{ Name = "tenant-service";         HealthUrl = "http://localhost:8082/health";    Container = "rinco-tenant-service";         Type = "docker"; Port = 8082 },
    @{ Name = "crm-service";           HealthUrl = "http://localhost:8083/health";    Container = "rinco-crm-service";           Type = "docker"; Port = 8083 },
    @{ Name = "dynamic-model-service";  HealthUrl = "http://localhost:8084/health";    Container = "rinco-dynamic-model-service";  Type = "docker"; Port = 8084 },
    @{ Name = "lead-service";          HealthUrl = "http://localhost:8085/health";    Container = "rinco-lead-service";          Type = "docker"; Port = 8085 },
    @{ Name = "landing-service";        HealthUrl = "http://localhost:8086/health";    Container = "rinco-landing-service";        Type = "docker"; Port = 8086 },
    @{ Name = "email-service";         HealthUrl = "http://localhost:8087/health";    Container = "rinco-email-service";         Type = "docker"; Port = 8087 },
    @{ Name = "notification-service";  HealthUrl = "http://localhost:8088/health";    Container = "rinco-notification-service";  Type = "docker"; Port = 8088 },
    @{ Name = "billing-service";       HealthUrl = "http://localhost:8095/health";    Container = "rinco-billing-service";       Type = "docker"; Port = 8095 },
    @{ Name = "observability-service";HealthUrl = "http://localhost:8096/health";    Container = "rinco-observability-service";Type = "docker"; Port = 8096 },
    @{ Name = "search-service";        HealthUrl = "http://localhost:8097/health";    Container = "rinco-search-service";        Type = "docker"; Port = 8097 },
    @{ Name = "meta-capi-service";    HealthUrl = "http://localhost:8098/health";    Container = "rinco-meta-capi-service";    Type = "docker"; Port = 8098 },
    @{ Name = "analytics-service";    HealthUrl = "http://localhost:8099/health";    Container = "rinco-analytics-service";    Type = "docker"; Port = 8099 },
    # --- Rust Services ---
    @{ Name = "chat-engine";           HealthUrl = "http://localhost:8101/health";    Container = "rinco-chat-engine";           Type = "docker"; Port = 8101 },
    @{ Name = "webrtc-sfu";           HealthUrl = "http://localhost:8102/health";    Container = "rinco-webrtc-sfu";           Type = "docker"; Port = 8102 },
    @{ Name = "recording-service";    HealthUrl = "http://localhost:8093/health";    Container = "rinco-recording-service";    Type = "docker"; Port = 8093 },
    # --- Python Services ---
    @{ Name = "ai-sre";               HealthUrl = "http://localhost:8090/health";    Container = "rinco-ai-sre";               Type = "docker"; Port = 8090 },
    @{ Name = "lead-scoring";         HealthUrl = "http://localhost:8092/health";    Container = "rinco-lead-scoring";         Type = "docker"; Port = 8092 },
    @{ Name = "rag-chatbot";          HealthUrl = "http://localhost:8091/health";    Container = "rinco-rag-chatbot";          Type = "docker"; Port = 8091 },
    @{ Name = "stt-service";         HealthUrl = "http://localhost:8094/health";    Container = "rinco-stt-service";         Type = "docker"; Port = 8094 },
    # --- Frontends ---
    @{ Name = "landing-frontend";     HealthUrl = "http://localhost:3000/health";    Container = "rinco-landing-frontend";     Type = "docker"; Port = 3000 },
    @{ Name = "admin-portal";         HealthUrl = "http://localhost:3001/health";    Container = "rinco-admin-portal";         Type = "docker"; Port = 3001 },
    @{ Name = "tenant-site";          HealthUrl = "http://localhost:3002/health";    Container = "rinco-tenant-site-frontend";Type = "docker"; Port = 3002 },
    @{ Name = "meeting-ui";           HealthUrl = "http://localhost:3003/health";    Container = "rinco-meeting-ui-frontend";  Type = "docker"; Port = 3003 },
    # --- Databases ---
    @{ Name = "postgres";             HealthUrl = "http://localhost:5433";            Container = "rinco-postgres";             Type = "docker"; Port = 5433 },
    @{ Name = "nats";                 HealthUrl = "http://localhost:8222/healthz";  Container = "rinco-nats";                 Type = "docker"; Port = 4222 },
    @{ Name = "valkey";               HealthUrl = "http://localhost:6379";           Container = "rinco-valkey";               Type = "docker"; Port = 6379 },
    @{ Name = "minio";                HealthUrl = "http://localhost:9000/minio/health/live"; Container = "rinco-minio";       Type = "docker"; Port = 9000 },
    @{ Name = "meilisearch";          HealthUrl = "http://localhost:7700/health";   Container = "rinco-meilisearch";          Type = "docker"; Port = 7700 },
    @{ Name = "qdrant";               HealthUrl = "http://localhost:6333/";          Container = "rinco-qdrant";               Type = "docker"; Port = 6333 },
    @{ Name = "mongodb";              HealthUrl = "http://localhost:27017";          Container = "rinco-mongodb";              Type = "docker"; Port = 27017 },
    @{ Name = "scylla";               HealthUrl = "http://localhost:9042";           Container = "rinco-scylla";               Type = "docker"; Port = 9042 },
    @{ Name = "clickhouse";           HealthUrl = "http://localhost:8123/ping";     Container = "rinco-clickhouse";           Type = "docker"; Port = 8123 },
    # --- Observability ---
    @{ Name = "prometheus";           HealthUrl = "http://localhost:9090/-/healthy"; Container = "rinco-prometheus";           Type = "docker"; Port = 9090 },
    @{ Name = "grafana";              HealthUrl = "http://localhost:13000/api/health"; Container = "rinco-grafana";              Type = "docker"; Port = 13000 },
    @{ Name = "loki";                 HealthUrl = "http://localhost:3100/ready";     Container = "rinco-loki";                 Type = "docker"; Port = 3100 }
)

# ---- State tracking ----
$FailureCount = @{}   # consecutive failures per service
$TotalRestarts = @{}  # total restarts per service
$LastRestart = @{}    # timestamp of last restart
$PassCount = 0
$FailCount = 0
$StartTime = Get-Date

function Write-Log {
    param([string]$Msg, [string]$Level = "INFO")
    $ts = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    $line = "[$ts] [$Level] $Msg"
    $line | Out-File -FilePath $LogFile -Append -Encoding utf8
    Write-Host $line
}

function Test-Health {
    param([string]$Url, [int]$TimeoutSec = 5)
    try {
        $resp = Invoke-WebRequest -Uri $Url -UseBasicParsing -TimeoutSec $TimeoutSec -ErrorAction SilentlyContinue
        if ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400) {
            return @{ OK = $true; Status = $resp.StatusCode; Time = 0 }
        }
        return @{ OK = $false; Status = $resp.StatusCode; Time = 0 }
    }
    catch {
        return @{ OK = $false; Status = 0; Time = 0; Error = $_.Exception.Message }
    }
}

function Test-DockerHealth {
    param([string]$ContainerName)
    try {
        $inspect = docker inspect $ContainerName --format "{{.State.Status}} {{.State.Health.Status}}" 2>$null
        if ($LASTEXITCODE -ne 0) { return "missing" }
        $parts = $inspect -split " "
        if ($parts[0] -eq "running") {
            return if ($parts.Count -gt 1) { $parts[1] } else { "running" }
        }
        return $parts[0]
    }
    catch { return "error" }
}

function Restart-DockerContainer {
    param([string]$ContainerName, [string]$ServiceName)
    Write-Log "Restarting container $ContainerName..." "WARN"
    docker restart $ContainerName 2>&1 | Out-Null
    $LastRestart[$ServiceName] = Get-Date
    if (-not $TotalRestarts.ContainsKey($ServiceName)) { $TotalRestarts[$ServiceName] = 0 }
    $TotalRestarts[$ServiceName]++
}

# ---- Main loop ----
Write-Log "WS-R Health Watcher started (PID=$Pid)"
Write-Log "Monitoring $($Services.Count) services every $IntervalSeconds seconds"
Write-Log "Log: $LogFile"

$Run = $true
$LoopCount = 0
$LastSummaryDate = (Get-Date -Format "yyyyMMdd")

while ($Run) {
    $LoopCount++
    $Ts = Get-Date -Format "HH:mm:ss"
    
    # Check for daily summary time (02:00 local)
    $CurrentDate = Get-Date -Format "yyyyMMdd"
    if ($CurrentDate -ne $LastSummaryDate -and (Get-Date -Format "HHmm") -ge "0200") {
        Write-Log "--- Daily Summary ---" "INFO"
        $summary = @"
WS-R Daily Summary — $LastSummaryDate
==========================================
Loop iterations today: $LoopCount
Total restarts: $($TotalRestarts.Values | Measure-Object -Sum).Sum
Services checked: $($Services.Count)
"@
        $summary | Out-File -FilePath $SummaryFile -Append -Encoding utf8
        $LastSummaryDate = $CurrentDate
        $LoopCount = 0
        $TotalRestarts = @{}
    }

    $pass = 0; $fail = 0
    $results = @()

    foreach ($svc in $Services) {
        $name = $svc.Name
        $url = $svc.HealthUrl
        $container = $svc.Container
        $port = $svc.Port

        # Check Docker container status
        $dockerStatus = Test-DockerHealth -ContainerName $container

        if ($dockerStatus -eq "missing") {
            # Container doesn't exist — skip HTTP check
            $results += @{ Name = $name; Status = "MISSING"; Time = 0; Docker = "missing" }
            $FailCount++
            $fail++
            if (-not $FailureCount.ContainsKey($name)) { $FailureCount[$name] = 0 }
            $FailureCount[$name]++
            continue
        }

        # Time the HTTP request
        $sw = [Diagnostics.Stopwatch]::StartNew()
        $result = Test-Health -Url $url
        $sw.Stop()
        $rt = $sw.ElapsedMilliseconds

        if ($result.OK) {
            $results += @{ Name = $name; Status = "PASS"; Time = $rt; Docker = $dockerStatus }
            $PassCount++; $pass++
            $FailureCount[$name] = 0
        }
        else {
            $results += @{ Name = $name; Status = "FAIL"; Time = $rt; Docker = $dockerStatus; Error = $result.Error }
            $FailCount++; $fail++
            if (-not $FailureCount.ContainsKey($name)) { $FailureCount[$name] = 0 }
            $FailureCount[$name]++
        }
    }

    # Log results
    $okResults = $results | Where-Object { $_.Status -eq "PASS" }
    $ngResults = $results | Where-Object { $_.Status -ne "PASS" }
    
    $header = "=== Loop $LoopCount | $Ts | PASS=$pass FAIL=$fail ==="
    Write-Log $header "INFO"
    
    foreach ($r in $okResults) {
        Write-Log "  $($r.Name): PASS ($($r.Time)ms, docker=$($r.Docker))" "INFO"
    }
    foreach ($r in $ngResults) {
        Write-Log "  $($r.Name): FAIL (docker=$($r.Docker), err=$($r.Error))" "WARN"
        
        # Auto-restart after $RestartThreshold consecutive failures
        if ($FailureCount[$r.Name] -ge $RestartThreshold) {
            Restart-DockerContainer -ContainerName $r.Name -ServiceName $r.Name
        }
    }

    # Short sleep
    Start-Sleep -Seconds $IntervalSeconds
}

Write-Log "WS-R Health Watcher stopped"
