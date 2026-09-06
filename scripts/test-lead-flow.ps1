#!/usr/bin/env pwsh
# Test lead submission flow end-to-end
$ErrorActionPreference = 'Continue'

$BaseUrl = $env:LANDING_URL ?? 'http://localhost:8086'
$TenantId = $env:TENANT_ID ?? '11111111-1111-1111-1111-111111111111'

Write-Host "`n=== Testing Lead Submission Flow ===" -ForegroundColor Cyan

# 1. Submit lead
Write-Host "`n[1/4] Submitting lead..." -ForegroundColor Yellow
$payload = @{
    form_id = 'webinar-2026'
    tenant_id = $TenantId
    full_name = 'Nguyen Van Test'
    phone = '0912345678'
    email = 'test@example.com'
    utm_source = 'facebook_ads'
    utm_campaign = 'webinar_q4_2026'
    fbclid = 'fb.test.123'
    source = 'landing'
    idempotency_key = "test-$(Get-Date -Format 'yyyyMMddHHmmss')"
    client_sent_at = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
}

try {
    $response = Invoke-WebRequest -Uri "$BaseUrl/v1/leads/submit" `
        -Method POST `
        -ContentType 'application/json' `
        -Body ($payload | ConvertTo-Json -Depth 10) `
        -UseBasicParsing

    $body = $response.Content | ConvertFrom-Json
    Write-Host "  ✓ Lead accepted: $($body.lead_id)" -ForegroundColor Green
    Write-Host "  ✓ Event ID: $($body.event_id)" -ForegroundColor Green
} catch {
    Write-Host "  ✗ Submission failed: $_" -ForegroundColor Red
    Write-Host "  (Make sure landing-service is running: docker compose -f infra/docker-compose.services.yml up -d landing-service)" -ForegroundColor Yellow
    exit 1
}

# 2. Check lead in PostgreSQL via API
Write-Host "`n[2/4] Fetching lead from CRM..." -ForegroundColor Yellow
try {
    $tenantSlug = 'demo'
    $loginRes = Invoke-WebRequest -Uri 'http://localhost:8081/v1/auth/login' `
        -Method POST `
        -ContentType 'application/json' `
        -Body (@{ email = 'demo@demo.com'; password = 'rinco_dev_password'; tenant_slug = $tenantSlug } | ConvertTo-Json) `
        -UseBasicParsing
    $token = ($loginRes.Content | ConvertFrom-Json).access_token

    $headers = @{
        Authorization = "Bearer $token"
        'X-Tenant-ID' = '11111111-1111-1111-1111-111111111111'
        'X-User-ID' = '11111111-1111-1111-1111-111111111111'
    }
    $leads = Invoke-WebRequest -Uri 'http://localhost:8083/v1/crm/leads' -Headers $headers -UseBasicParsing
    Write-Host "  ✓ Found $($leads.Content | ConvertFrom-Json).count leads" -ForegroundColor Green
} catch {
    Write-Host "  ! CRM check failed (this is OK if auth-service is not running): $_" -ForegroundColor Yellow
}

# 3. Check observability
Write-Host "`n[3/4] Checking service health..." -ForegroundColor Yellow
try {
    $health = Invoke-WebRequest -Uri 'http://localhost:8089/v1/observability/health/all' -UseBasicParsing
    $body = $health.Content | ConvertFrom-Json
    Write-Host "  ✓ Overall: $($body.overall)" -ForegroundColor Green
    Write-Host "  ✓ Services: $(($body.services | Measure-Object).Count)" -ForegroundColor Green
} catch {
    Write-Host "  ! Observability check failed" -ForegroundColor Yellow
}

# 4. Test scoring
Write-Host "`n[4/4] Testing lead scoring..." -ForegroundColor Yellow
try {
    $scorePayload = @{
        email = 'test@example.com'
        phone = '0912345678'
        full_name = 'Nguyen Van Test'
        has_phone = $true
        has_email = $true
        page_views = 5
        time_on_site_seconds = 300
        source = 'facebook_ads'
    }
    $score = Invoke-WebRequest -Uri 'http://localhost:8092/score' `
        -Method POST `
        -ContentType 'application/json' `
        -Headers @{ 'X-Tenant-ID' = $TenantId } `
        -Body ($scorePayload | ConvertTo-Json) `
        -UseBasicParsing
    Write-Host "  ✓ Score: $($score.Content | ConvertFrom-Json).score" -ForegroundColor Green
} catch {
    Write-Host "  ! Scoring failed (lead-scoring may not be running)" -ForegroundColor Yellow
}

Write-Host "`n✓ Test complete`n" -ForegroundColor Green
