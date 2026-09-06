#!/usr/bin/env pwsh
# Health check tất cả services
$ErrorActionPreference = 'Continue'

$services = @(
    @{ Name = 'auth-service'; URL = 'http://localhost:8081/health' },
    @{ Name = 'tenant-service'; URL = 'http://localhost:8082/health' },
    @{ Name = 'crm-service'; URL = 'http://localhost:8083/health' },
    @{ Name = 'dynamic-model-service'; URL = 'http://localhost:8084/health' },
    @{ Name = 'lead-service'; URL = 'http://localhost:8085/health' },
    @{ Name = 'landing-service'; URL = 'http://localhost:8086/health' },
    @{ Name = 'email-service'; URL = 'http://localhost:8087/health' },
    @{ Name = 'notification-service'; URL = 'http://localhost:8088/health' },
    @{ Name = 'observability-service'; URL = 'http://localhost:8089/health' },
    @{ Name = 'ai-sre'; URL = 'http://localhost:8090/health' },
    @{ Name = 'rag-chatbot'; URL = 'http://localhost:8091/health' },
    @{ Name = 'lead-scoring'; URL = 'http://localhost:8092/health' },
    @{ Name = 'stt-service'; URL = 'http://localhost:8093/health' },
    @{ Name = 'chat-engine'; URL = 'http://localhost:8094/health' },
    @{ Name = 'webrtc-sfu'; URL = 'http://localhost:8095/health' },
    @{ Name = 'recording-service'; URL = 'http://localhost:8096/health' }
)

Write-Host "`n=== RINCO Service Health Check ===" -ForegroundColor Cyan
$healthy = 0
$unhealthy = 0

foreach ($svc in $services) {
    try {
        $response = Invoke-WebRequest -Uri $svc.URL -Method GET -TimeoutSec 5 -UseBasicParsing
        if ($response.StatusCode -eq 200) {
            Write-Host "  ✓ $($svc.Name): HEALTHY" -ForegroundColor Green
            $healthy++
        } else {
            Write-Host "  ✗ $($svc.Name): HTTP $($response.StatusCode)" -ForegroundColor Red
            $unhealthy++
        }
    } catch {
        Write-Host "  ✗ $($svc.Name): DOWN" -ForegroundColor Red
        $unhealthy++
    }
}

Write-Host "`n=== Summary ===" -ForegroundColor Cyan
Write-Host "  Healthy: $healthy" -ForegroundColor Green
Write-Host "  Unhealthy: $unhealthy" -ForegroundColor Red
Write-Host ""
