@echo off
setlocal enabledelayedexpansion

REM ==========================================================
REM RINCO Start Core Services
REM ==========================================================

set POSTGRES_URL=postgres://rinco:rinco_dev_password@localhost:5433/rinco?sslmode=disable
set VALKEY_URL=localhost:6379
set VALKEY_PASSWORD=rinco_dev_password
set NATS_URL=nats://localhost:4222

REM Tenant Service - Port 8082
echo Starting tenant-service on :8082...
cd /d c:\code\RINCO
start /b cmd /c "set TENANT_DATABASE_URL=%POSTGRES_URL% && set TENANT_VALKEY_URL=redis://:%VALKEY_PASSWORD%@%VALKY_URL% && services\tenant-service\bin\tenant-service.exe 2>>logs\tenant.log"

REM CRM Service - Port 8083  
echo Starting crm-service on :8083...
start /b cmd /c "set CRM_DATABASE_URL=%POSTGRES_URL% && set CRM_VALKEY_URL=redis://:%VALKEY_PASSWORD%@%VALKY_URL% && services\crm-service\bin\crm-service.exe 2>>logs\crm.log"

REM Lead Service - Port 8085
echo Starting lead-service on :8085...
start /b cmd /c "set LEAD_DATABASE_URL=%POSTGRES_URL% && set LEAD_VALKEY_URL=redis://:%VALKEY_PASSWORD%@%VALKY_URL% && services\lead-service\bin\lead-service.exe 2>>logs\lead.log"

REM Landing Service - Port 8086
echo Starting landing-service on :8086...
start /b cmd /c "set LANDING_DATABASE_URL=%POSTGRES_URL% && set LANDING_VALKEY_URL=redis://:%VALKEY_PASSWORD%@%VALKY_URL% && services\landing-service\bin\landing-service.exe 2>>logs\landing.log"

echo.
echo All core services started!
echo Check logs in logs\ directory
echo.
