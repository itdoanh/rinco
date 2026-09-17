#!/usr/bin/env pwsh
# ============================================================
# RINCO — Health check all services (PowerShell)
# ============================================================
# Returns aggregate healthy/unhealthy counts and exit code 1 if any DOWN.
# ============================================================

$ErrorActionPreference = 'Continue'

$services = @(
    @{ name = "auth-service";          port = 8080; path = "/health" }
    @{ name = "crm-service";           port = 8081; path = "/health" }
    @{ name = "tenant-service";        port = 8082; path = "/health" }
    @{ name = "landing-service";       port = 8083; path = "/health" }
    @{ name = "meta-capi-service";     port = 8084; path = "/health" }
    @{ name = "lead-service";          port = 8085; path = "/health" }
    @{ name = "chat-engine";           port = 8086; path = "/health" }
    @{ name = "webrtc-sfu";            port = 8087; path = "/health" }
    @{ name = "notification-service";  port = 8088; path = "/health" }
    @{ name = "email-service";         port = 8089; path = "/health" }
    @{ name = "search-service";        port = 8090; path = "/health" }
    @{ name = "dynamic-model-service"; port = 8091; path = "/health" }
    @{ name = "billing-service";       port = 8092; path = "/health" }
    @{ name = "observability-service"; port = 8093; path = "/health" }
    @{ name = "lead-scoring";          port = 8094; path = "/health" }
    @{ name = "rag-chatbot";           port = 8095; path = "/health" }
    @{ name = "ai-sre";                port = 8096; path = "/health" }
    @{ name = "stt-service";           port = 8097; path = "/health" }
    @{ name = "admin-portal";          port = 3001; path = "/"        }
    @{ name = "landing";               port = 3002; path = "/"        }
    @{ name = "meeting-ui";            port = 3003; path = "/"        }
    @{ name = "tenant-site";           port = 3004; path = "/"        }
)

$healthy = 0
$unhealthy = 0
$results = @()

foreach ($svc in $services) {
    $url = "http://localhost:$($svc.port)$($svc.path)"
    try {
        $response = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 5 -ErrorAction Stop
        if ($response.StatusCode -ge 200 -and $response.StatusCode -lt 500) {
            Write-Host "  [OK]   $($svc.name) :$($svc.port)" -ForegroundColor Green
            $results += [pscustomobject]@{ Name = $svc.name; Port = $svc.port; Status = "OK"; Code = $response.StatusCode }
            $healthy++
        } else {
            Write-Host "  [WARN] $($svc.name) :$($svc.port) -> HTTP $($response.StatusCode)" -ForegroundColor Yellow
            $results += [pscustomobject]@{ Name = $svc.name; Port = $svc.port; Status = "WARN"; Code = $response.StatusCode }
            $unhealthy++
        }
    } catch {
        Write-Host "  [DOWN] $($svc.name) :$($svc.port)" -ForegroundColor Red
        $results += [pscustomobject]@{ Name = $svc.name; Port = $svc.port; Status = "DOWN"; Code = "n/a" }
        $unhealthy++
    }
}

Write-Host ""
Write-Host "===== Health Summary =====" -ForegroundColor Cyan
Write-Host "Healthy:   $healthy" -ForegroundColor Green
Write-Host "Unhealthy: $unhealthy" -ForegroundColor Red
Write-Host "Total:     $($services.Count)"

if ($unhealthy -gt 0) { exit 1 } else { exit 0 }
