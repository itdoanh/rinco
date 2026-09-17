@echo off
setlocal enabledelayedexpansion

REM ==========================================================
REM RINCO Build All Go Services
REM ==========================================================

set SERVICES=auth-service tenant-service crm-service lead-service landing-service dynamic-model-service email-service notification-service observability-service billing-service search-service analytics-service meta-capi-service

for %%S in (%SERVICES%) do (
    echo ==========================================
    echo Building %%S...
    echo ==========================================
    cd /d c:\code\RINCO\services\%%S
    if not exist bin mkdir bin
    go build -o bin\%%S.exe .\cmd 2>nul
    if !ERRORLEVEL! EQU 0 (
        echo [OK] %%S built successfully
    ) else (
        echo [FAIL] %%S build failed
    )
)

echo ==========================================
echo All services build completed!
echo ==========================================
