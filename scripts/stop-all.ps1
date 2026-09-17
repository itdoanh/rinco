#!/usr/bin/env pwsh
# ============================================================
# RINCO — Stop all services (PowerShell)
# ============================================================
# Gracefully kills the local RINCO processes started by start-all.ps1.
# ============================================================

$ErrorActionPreference = 'SilentlyContinue'

$processPatterns = @(
    "auth-service",
    "crm-service",
    "tenant-service",
    "landing-service",
    "meta-capi-service",
    "lead-service",
    "chat-engine",
    "webrtc-sfu",
    "notification-service",
    "email-service",
    "search-service",
    "dynamic-model-service",
    "billing-service",
    "observability-service",
    "lead-scoring",
    "rag-chatbot",
    "ai-sre",
    "stt-service",
    "admin-portal",
    "landing-ui",
    "meeting-ui",
    "tenant-site",
    "next-server",
    "node",
    "python"
)

Write-Host "Stopping RINCO services..." -ForegroundColor Cyan

$stopped = 0
foreach ($pattern in $processPatterns) {
    $procs = Get-Process -Name "*${pattern}*" -ErrorAction SilentlyContinue
    foreach ($proc in $procs) {
        try {
            Write-Host "  [-] Stopping $($proc.ProcessName) (PID $($proc.Id))"
            Stop-Process -Id $proc.Id -Force
            $stopped++
        } catch {
            Write-Warning "  [!] Could not stop $($proc.ProcessName) (PID $($proc.Id)): $_"
        }
    }
}

Write-Host ""
Write-Host "Stopped $stopped process(es)." -ForegroundColor Green
