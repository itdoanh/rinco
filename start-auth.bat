@echo off
setlocal

REM ==========================================================
REM RINCO Auth Service Runner
REM ==========================================================

set AUTH_HTTP_ADDR=:8081
set AUTH_DATABASE_URL=postgres://rinco:rinco_dev_password@localhost:5433/rinco?sslmode=disable
set AUTH_VALKEY_URL=localhost:6379
set AUTH_VALKEY_PASSWORD=rinco_dev_password
set AUTH_PASETO_KEY=6f1c2e3d4b5a69788798a0b1c2d3e4f506172839405162738495a6b7c8d9e0f1
set AUTH_PASETO_PUBLIC=6f1c2e3d4b5a69788798a0b1c2d3e4f506172839405162738495a6b7c8d9e0f1
set AUTH_NATS_URL=nats://localhost:4222
set AUTH_RP_ID=localhost
set AUTH_RP_ORIGIN=http://localhost:3001
set AUTH_WEB_BASE_URL=http://localhost:8081
set ENV=development

cd /d %~dp0\services\auth-service
echo Starting auth-service on :8081...
bin\auth-service.exe
