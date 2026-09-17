# ============================================
# RINCO smoke test — PowerShell (Loop WS-A)
# ============================================
# Probes /health on every backend service + every frontend app.
# Exits 0 on all-2xx, 1 on unreachable, 2 on degraded.
#
# Usage:
#   pwsh scripts/smoke.ps1                 # localhost
#   $env:BASE='http://staging'; pwsh scripts/smoke.ps1
#   pwsh scripts/smoke.ps1 -Strict         # non-2xx is fatal
# ============================================
[CmdletBinding()]
param(
    [string]$Base   = 'http://localhost',
    [int]$TimeoutSec = 3,
    [switch]$Strict
)

$ErrorActionPreference = 'Continue'

$Checks = @(
    @{ Name='auth-service';          Port=8081; Kind='go';      Path='/health' },
    @{ Name='tenant-service';        Port=8082; Kind='go';      Path='/health' },
    @{ Name='crm-service';           Port=8083; Kind='go';      Path='/health' },
    @{ Name='dynamic-model-service'; Port=8084; Kind='go';      Path='/health' },
    @{ Name='lead-service';          Port=8085; Kind='go';      Path='/health' },
    @{ Name='landing-service';       Port=8086; Kind='go';      Path='/health' },
    @{ Name='email-service';         Port=8087; Kind='go';      Path='/health' },
    @{ Name='notification-service';  Port=8088; Kind='go';      Path='/health' },
    @{ Name='billing-service';       Port=8095; Kind='go';      Path='/health' },
    @{ Name='observability-service'; Port=8096; Kind='go';      Path='/health' },
    @{ Name='search-service';        Port=8097; Kind='go';      Path='/health' },
    @{ Name='meta-capi-service';     Port=8098; Kind='go';      Path='/health' },
    @{ Name='analytics-service';     Port=8099; Kind='go';      Path='/health' },
    @{ Name='ai-sre';                Port=8090; Kind='python';  Path='/health' },
    @{ Name='lead-scoring';          Port=8091; Kind='python';  Path='/health' },
    @{ Name='rag-chatbot';           Port=8092; Kind='python';  Path='/health' },
    @{ Name='stt-service';           Port=8094; Kind='python';  Path='/health' },
    @{ Name='recording-service';     Port=8093; Kind='rust';    Path='/health' },
    @{ Name='chat-engine';           Port=8101; Kind='rust';    Path='/health' },
    @{ Name='webrtc-sfu';            Port=8102; Kind='rust';    Path='/health' },
    @{ Name='landing-frontend';      Port=3000; Kind='frontend';Path='/health' },
    @{ Name='admin-portal-frontend'; Port=3001; Kind='frontend';Path='/health' },
    @{ Name='tenant-site-frontend';  Port=3002; Kind='frontend';Path='/health' },
    @{ Name='meeting-ui-frontend';   Port=3003; Kind='frontend';Path='/health' }
)

function Probe {
    param($Check)
    $url = "$Base`:$($Check.Port)$($Check.Path)"
    try {
        $r = Invoke-WebRequest -Uri $url -TimeoutSec $TimeoutSec -UseBasicParsing -ErrorAction Stop
        $code = $r.StatusCode
    } catch {
        $code = 0
        if ($_.Exception.Response) { $code = [int]$_.Exception.Response.StatusCode }
    }

    if ($code -ge 200 -and $code -lt 300) {
        Write-Host ("[OK] {0,-26} {1,-10} :{2}  {3}" -f $Check.Name, $Check.Kind, $Check.Port, $code) -ForegroundColor Green
        return 'ok'
    }
    if ($code -eq 0 -and $Check.Kind -eq 'frontend') {
        # Some frontend apps don't ship /health; fall back to root.
        $rootUrl = "$Base`:$($Check.Port)/"
        try {
            $r = Invoke-WebRequest -Uri $rootUrl -TimeoutSec $TimeoutSec -UseBasicParsing -ErrorAction Stop
            if ($r.StatusCode -ge 200 -and $r.StatusCode -lt 300) {
                Write-Host ("[OK] {0,-26} {1,-10} :{2}  {3} (root)" -f $Check.Name, $Check.Kind, $Check.Port, $r.StatusCode) -ForegroundColor Green
                return 'ok'
            }
        } catch {}
    }
    if ($code -eq 0) {
        Write-Host ("[--] {0,-26} {1,-10} :{2}  unreachable" -f $Check.Name, $Check.Kind, $Check.Port) -ForegroundColor Red
        return 'fail'
    }
    Write-Host ("[!]  {0,-26} {1,-10} :{2}  HTTP {3}" -f $Check.Name, $Check.Kind, $Check.Port, $code) -ForegroundColor Yellow
    return 'degraded'
}

Write-Host ("{0,-26} {1,-10} {2,-6}  {3}" -f "service", "kind", "port", "code")
Write-Host ("{0,-26} {1,-10} {2,-6}  {3}" -f "--------------------------", "----------", "------", "----")

$total = 0; $ok = 0; $degraded = 0; $failed = 0
$failedNames = @()

foreach ($c in $Checks) {
    $total++
    $r = Probe -Check $c
    switch ($r) {
        'ok'       { $ok++ }
        'degraded' { $degraded++; $failedNames += $c.Name }
        'fail'     { $failed++;   $failedNames += $c.Name }
    }
}

Write-Host ""
Write-Host ("Summary: {0} total, {1} healthy, {2} degraded, {3} failed" -f $total, $ok, $degraded, $failed)

if ($failed -gt 0) {
    Write-Host ""
    Write-Host "Failed services:" -ForegroundColor Red
    foreach ($n in $failedNames) { Write-Host "  - $n" }
    exit 1
}
if ($degraded -gt 0 -and $Strict) { exit 2 }
exit 0
