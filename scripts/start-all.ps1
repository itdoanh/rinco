#!/usr/bin/env pwsh
# ============================================================
# RINCO — Start all services (PowerShell / Windows native)
# ============================================================
# Usage:
#   pwsh scripts/start-all.ps1
# ============================================================

$ErrorActionPreference = 'Continue'
$RootDir = Resolve-Path (Join-Path $PSScriptRoot "..")

# Map service-name -> port
$services = [ordered]@{
    "auth-service"           = 8080
    "crm-service"            = 8081
    "tenant-service"         = 8082
    "landing-service"        = 8083
    "meta-capi-service"      = 8084
    "lead-service"           = 8085
    "chat-engine"            = 8086
    "webrtc-sfu"             = 8087
    "notification-service"   = 8088
    "email-service"          = 8089
    "search-service"         = 8090
    "dynamic-model-service"  = 8091
    "billing-service"        = 8092
    "observability-service"  = 8093
}

Write-Host "Starting $($services.Count) Go/Rust services..." -ForegroundColor Cyan

foreach ($svc in $services.GetEnumerator()) {
    $name = $svc.Key
    $port = $svc.Value
    $path = Join-Path $RootDir "services/$name"

    if (-not (Test-Path $path)) {
        Write-Warning "[$name] path not found: $path"
        continue
    }

    $bin = Get-ChildItem -Path "$path/bin" -Filter "*.exe" -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($bin) {
        Write-Host "  [+] Starting $name on port $port ..." -ForegroundColor Green
        Start-Process -FilePath $bin.FullName -WorkingDirectory $path -WindowStyle Hidden
        Start-Sleep -Milliseconds 500
    } else {
        Write-Warning "[-] Binary not found for $name - run 'make go-build' or 'cargo build' first"
    }
}

# AI / Python services
$aiServices = [ordered]@{
    "lead-scoring" = 8094
    "rag-chatbot"  = 8095
    "ai-sre"       = 8096
    "stt-service"  = 8097
}

Write-Host ""
Write-Host "Starting $($aiServices.Count) AI services (Python)..." -ForegroundColor Cyan
foreach ($svc in $aiServices.GetEnumerator()) {
    $name = $svc.Key
    $port = $svc.Value
    $path = Join-Path $RootDir "services/$name"
    if (Test-Path "$path/main.py") {
        Write-Host "  [+] Starting $name on port $port ..." -ForegroundColor Green
        Start-Process -FilePath "python" -ArgumentList "main.py" -WorkingDirectory $path -WindowStyle Hidden
        Start-Sleep -Milliseconds 500
    } else {
        Write-Warning "[-] main.py not found for $name"
    }
}

# Frontend services
$frontends = [ordered]@{
    "admin-portal" = 3001
    "landing"      = 3002
    "meeting-ui"   = 3003
    "tenant-site"  = 3004
}

Write-Host ""
Write-Host "Starting $($frontends.Count) frontend dev servers..." -ForegroundColor Cyan
foreach ($fe in $frontends.GetEnumerator()) {
    $name = $fe.Key
    $port = $fe.Value
    $path = Join-Path $RootDir "frontend/$name"
    if (Test-Path "$path/package.json") {
        Write-Host "  [+] Starting $name on port $port ..." -ForegroundColor Green
        Start-Process -FilePath "npm" -ArgumentList "run","dev","--","--port",$port -WorkingDirectory $path -WindowStyle Hidden
        Start-Sleep -Seconds 1
    } else {
        Write-Warning "[-] package.json not found for $name"
    }
}

Write-Host ""
Write-Host "All services started. Open:" -ForegroundColor Green
Write-Host "   Admin portal:  http://localhost:3001" -ForegroundColor Cyan
Write-Host "   Landing:       http://localhost:3002" -ForegroundColor Cyan
Write-Host "   Meeting UI:    http://localhost:3003" -ForegroundColor Cyan
Write-Host "   Tenant site:   http://localhost:3004" -ForegroundColor Cyan
