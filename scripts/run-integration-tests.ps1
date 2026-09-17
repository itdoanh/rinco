# run-integration-tests.ps1 — Convenience wrapper for Windows / PowerShell.
# Mirrors scripts/run-integration-tests.sh.
#
# Usage:
#   .\scripts\run-integration-tests.ps1                 # use local docker stack
#   $env:RINCO_NO_DOCKER=1; .\scripts\run-integration-tests.ps1   # skip docker
#   $env:CHECK_COVERAGE_STRICT=1; .\scripts\run-integration-tests.ps1

[CmdletBinding()]
param(
    [int]$CoverageMin = 80
)

$ErrorActionPreference = 'Stop'

$Root = Split-Path -Parent $PSScriptRoot
$TestDir = Join-Path $Root 'services\integration-tests'

if (-not $env:RINCO_NO_DOCKER -or $env:RINCO_NO_DOCKER -ne '1') {
    Write-Host "==> bringing up local docker-compose stack"
    Push-Location $Root
    try {
        docker compose -f docker-compose.yml up -d | Out-Null
    } finally {
        Pop-Location
    }
}

$env:RINCO_USE_EXTERNAL_STACK = 'true'
$env:RINCO_TEST_POSTGRES_DSN = if ($env:RINCO_TEST_POSTGRES_DSN) { $env:RINCO_TEST_POSTGRES_DSN } else { 'postgres://rinco:rinco@localhost:5432/rinco_test?sslmode=disable' }
$env:RINCO_TEST_VALKEY_ADDR = if ($env:RINCO_TEST_VALKEY_ADDR) { $env:RINCO_TEST_VALKEY_ADDR } else { 'localhost:6379' }
$env:RINCO_TEST_NATS_URL    = if ($env:RINCO_TEST_NATS_URL) { $env:RINCO_TEST_NATS_URL } else { 'nats://localhost:4222' }

Push-Location $TestDir
try {
    Write-Host "==> running integration tests with coverage"
    go test -tags=integration -count=1 -race -timeout 10m `
             -coverprofile=coverage.out -covermode=atomic `
             -v ./...

    Write-Host "==> generating HTML coverage report"
    go tool cover -html=coverage.out -o coverage.html

    Write-Host "==> coverage summary"
    go tool cover -func=coverage.out | Select-Object -Last 1
} finally {
    Pop-Location
}

Write-Host "==> done. coverage report at $TestDir\coverage.html"
