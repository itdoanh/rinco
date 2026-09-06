#!/usr/bin/env pwsh
# Setup RINCO development environment
param(
    [switch]$SkipInfra = $false,
    [switch]$SkipMigrate = $false,
    [switch]$SkipBuild = $false
)

$ErrorActionPreference = 'Stop'
$RootDir = Split-Path -Parent $PSScriptRoot

Write-Host "========================================" -ForegroundColor Cyan
Write-Host " RINCO Development Environment Setup" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan

# 1. Check prerequisites
Write-Host "`n[1/6] Checking prerequisites..." -ForegroundColor Yellow
$prerequisites = @{
    "docker" = "docker --version"
    "docker-compose" = "docker compose version"
    "go" = "go version"
    "rust" = "rustc --version"
    "bun" = "bun --version"
    "python" = "python --version"
}

$missing = @()
foreach ($tool in $prerequisites.Keys) {
    try {
        $cmd = $prerequisites[$tool]
        $version = Invoke-Expression $cmd 2>&1
        Write-Host "  ✓ $tool : $version" -ForegroundColor Green
    } catch {
        $missing += $tool
        Write-Host "  ✗ $tool : not found" -ForegroundColor Red
    }
}

if ($missing.Count -gt 0) {
    Write-Host "`nMissing tools: $($missing -join ', ')" -ForegroundColor Red
    Write-Host "Please install them before continuing." -ForegroundColor Red
    exit 1
}

# 2. Generate environment file
Write-Host "`n[2/6] Generating .env file..." -ForegroundColor Yellow
$envContent = @"
# RINCO Development Environment
PASETO_KEY_CURRENT=$(openssl rand -hex 32)
PASETO_KEY_PREVIOUS=
ADMIN_API_KEY=dev_admin_key_$(openssl rand -hex 8)
INGESTION_HMAC_SECRET=dev_secret_$(openssl rand -hex 16)
TURN_SECRET=dev_turn_secret_$(openssl rand -hex 16)

# AI
TELEGRAM_BOT_TOKEN=
TELEGRAM_CHAT_ID=
SLACK_WEBHOOK_URL=
GITHUB_TOKEN=

# External
NEXT_PUBLIC_FB_PIXEL_ID=
"@
$envPath = "$RootDir/.env"
if (-not (Test-Path $envPath)) {
    Set-Content -Path $envPath -Value $envContent
    Write-Host "  ✓ .env created" -ForegroundColor Green
} else {
    Write-Host "  ! .env already exists" -ForegroundColor Yellow
}

# 3. Start infrastructure
if (-not $SkipInfra) {
    Write-Host "`n[3/6] Starting infrastructure (Docker Compose)..." -ForegroundColor Yellow
    Push-Location "$RootDir/infra"
    try {
        docker compose -f docker-compose.yml up -d
        Write-Host "  ✓ Infrastructure started" -ForegroundColor Green
    } catch {
        Write-Host "  ✗ Failed to start infra: $_" -ForegroundColor Red
        Pop-Location
        exit 1
    }
    Pop-Location
} else {
    Write-Host "`n[3/6] Skipping infrastructure start" -ForegroundColor Yellow
}

# 4. Wait for databases
Write-Host "`n[4/6] Waiting for databases to be ready..." -ForegroundColor Yellow
Start-Sleep -Seconds 10

# 5. Build services
if (-not $SkipBuild) {
    Write-Host "`n[5/6] Building Go packages..." -ForegroundColor Yellow
    Push-Location $RootDir
    try {
        go mod tidy
        Write-Host "  ✓ Go modules tidied" -ForegroundColor Green
    } catch {
        Write-Host "  ! Go mod tidy failed (this is OK if not running services yet)" -ForegroundColor Yellow
    }
    Pop-Location
}

# 6. Done
Write-Host "`n[6/6] Setup complete!" -ForegroundColor Green
Write-Host @"

Next steps:
  1. Start services:    cd infra && docker compose -f docker-compose.services.yml up
  2. Run migrations:    bash scripts/migrate.sh (or pwsh scripts/migrate.ps1)
  3. Test:              curl http://localhost:8081/health
  4. View docs:         open docs/DEV-PLAN.md

Frontend URLs (when services are up):
  - Landing:     http://localhost:3001
  - Admin:       http://localhost:3002
  - Grafana:     http://localhost:3000 (admin/admin)
  - Prometheus:  http://localhost:9090
  - Jaeger:      http://localhost:16686
  - MinIO:       http://localhost:9001 (rinco/rinco_dev_password)
  - MailHog:     http://localhost:8025

"@ -ForegroundColor Cyan
