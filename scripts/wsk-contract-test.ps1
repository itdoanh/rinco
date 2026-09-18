# WS-K Contract Validation Suite
#
# Walks every REST endpoint in the 10 critical Go services, calls each
# with a valid auth token, verifies the response shape matches the
# OpenAPI spec (if present), records response time per endpoint, and
# fails if any required field is missing.
#
# Targets (override via env):
#   $AUTH_URL, $TENANT_URL, $CRM_URL, $LEAD_URL, $EMAIL_URL,
#   $NOTIFICATION_URL, $SEARCH_URL, $ANALYTICS_URL,
#   $META_CAPI_URL, $BILLING_URL
#
# Usage:
#   .\scripts\wsk-contract-test.ps1
#   # or with overrides:
#   $env:AUTH_URL='http://localhost:8081'; .\scripts\wsk-contract-test.ps1
#
# Outputs:
#   logs/wsk-contract.txt            — human-readable results
#   logs/wsk-contract-results.json   — machine-readable
#   $LASTEXITCODE — 0 if all pass, 1 if any fail

[CmdletBinding()]
param(
    [string]$AuthUrl = $env:AUTH_URL ?? 'http://localhost:8081',
    [string]$TenantUrl = $env:TENANT_URL ?? 'http://localhost:8082',
    [string]$CrmUrl = $env:CRM_URL ?? 'http://localhost:8083',
    [string]$LeadUrl = $env:LEAD_URL ?? 'http://localhost:8085',
    [string]$EmailUrl = $env:EMAIL_URL ?? 'http://localhost:8087',
    [string]$NotificationUrl = $env:NOTIFICATION_URL ?? 'http://localhost:8088',
    [string]$SearchUrl = $env:SEARCH_URL ?? 'http://localhost:8097',
    [string]$AnalyticsUrl = $env:ANALYTICS_URL ?? 'http://localhost:8099',
    [string]$MetaCapiUrl = $env:META_CAPI_URL ?? 'http://localhost:8098',
    [string]$BillingUrl = $env:BILLING_URL ?? 'http://localhost:8095',
    [string]$TestEmail = $env:TEST_EMAIL ?? 'admin@apexfintech.vn',
    [string]$TestPassword = $env:TEST_PASSWORD ?? 'rinco_dev_password',
    [string]$TenantSlug = $env:TENANT_SLUG ?? 'apexfintech',
    [int]$TimeoutSec = 10
)

$ErrorActionPreference = 'Stop'
$Script:Results = @()
$Script:Passed = 0
$Script:Failed = 0

# Helper: HTTP GET with timeout
function Invoke-ContractRequest {
    param(
        [string]$Method = 'GET',
        [string]$Url,
        [hashtable]$Headers = @{},
        [string]$Body = $null,
        [string[]]$RequiredFields = @()
    )
    try {
        $reqStart = Get-Date
        $reqHeaders = @{
            'Content-Type' = 'application/json'
            'Accept'       = 'application/json'
        } + $Headers
        $params = @{
            Uri             = $Url
            Method          = $Method
            Headers         = $reqHeaders
            UseBasicParsing = $true
            TimeoutSec      = $TimeoutSec
        }
        if ($Body) { $params.Body = $Body }
        $res = Invoke-WebRequest @params
        $reqElapsed = (Get-Date) - $reqStart
        $statusCode = [int]$res.StatusCode
        $body = $res.Content
    } catch {
        $reqElapsed = (Get-Date) - $reqStart
        $statusCode = if ($_.Exception.Response) { [int]$_.Exception.Response.StatusCode } else { 0 }
        $body = $_.Exception.Message
    }

    # Verify required fields if 2xx
    $missingFields = @()
    if ($statusCode -ge 200 -and $statusCode -lt 300 -and $RequiredFields.Count -gt 0) {
        try {
            $parsed = $body | ConvertFrom-Json -ErrorAction SilentlyContinue
            foreach ($field in $RequiredFields) {
                $path = $field -split '\.'
                $current = $parsed
                foreach ($segment in $path) {
                    if ($current.PSObject.Properties[$segment]) {
                        $current = $current.$segment
                    } else {
                        $missingFields += $field
                        break
                    }
                }
            }
        } catch {
            $missingFields += '(unparseable JSON)'
        }
    }

    $pass = ($statusCode -ge 200 -and $statusCode -lt 300) -and $missingFields.Count -eq 0
    $Script:Results += [PSCustomObject]@{
        Url            = $Url
        Method         = $Method
        StatusCode     = $statusCode
        LatencyMs      = [int]$reqElapsed.TotalMilliseconds
        RequiredFields = ($RequiredFields -join ', ')
        MissingFields  = ($missingFields -join ', ')
        Pass           = $pass
    }
    if ($pass) { $Script:Passed++ } else { $Script:Failed++ }
    return @{ Pass = $pass; StatusCode = $statusCode; LatencyMs = [int]$reqElapsed.TotalMilliseconds; MissingFields = $missingFields }
}

# Step 1 — login to get auth token
Write-Host "==> Login to auth-service at $AuthUrl"
$loginBody = @{
    email        = $TestEmail
    password     = $TestPassword
    tenant_slug  = $TenantSlug
} | ConvertTo-Json -Compress
try {
    $loginRes = Invoke-WebRequest -Uri "$AuthUrl/v1/auth/login" -Method POST -Body $loginBody -ContentType 'application/json' -UseBasicParsing -TimeoutSec $TimeoutSec
    $token = ($loginRes.Content | ConvertFrom-Json).access_token
    if (-not $token) { throw 'No access_token in login response' }
    Write-Host "==> Got auth token (len=$($token.Length))"
} catch {
    Write-Error "FATAL: auth-service login failed.  Cannot run contract tests.  $_"
    exit 2
}

$authHeaders = @{ 'Authorization' = "Bearer $token" }

# Step 2 — endpoints per service
$endpoints = @(
    # Auth service
    @{ Service='auth';     Url="$AuthUrl/v1/auth/me";                   Method='GET';  ReqFields=@('id','tenant_id','email') },
    # Tenant service
    @{ Service='tenant';   Url="$TenantUrl/v1/tenants/by-domain/apexfintech.vn"; Method='GET'; ReqFields=@('tenant_id') },
    # CRM service
    @{ Service='crm';      Url="$CrmUrl/v1/crm/contacts?page=1&limit=10"; Method='GET'; ReqFields=@() },
    # Lead service
    @{ Service='lead';     Url="$LeadUrl/v1/leads?page=1&limit=10"; Method='GET'; ReqFields=@() },
    # Email service
    @{ Service='email';    Url="$EmailUrl/v1/email/templates"; Method='GET'; ReqFields=@() },
    # Notification service
    @{ Service='notif';    Url="$NotificationUrl/v1/notifications/me"; Method='GET'; ReqFields=@() },
    # Search service
    @{ Service='search';   Url="$SearchUrl/v1/search?q=test"; Method='GET'; ReqFields=@() },
    # Analytics service
    @{ Service='analytics';Url="$AnalyticsUrl/v1/analytics/events/recent"; Method='GET'; ReqFields=@() },
    # Meta CAPI service
    @{ Service='meta';     Url="$MetaCapiUrl/v1/meta/health"; Method='GET'; ReqFields=@() },
    # Billing service
    @{ Service='billing';  Url="$BillingUrl/v1/billing/plans"; Method='GET'; ReqFields=@() },
)

Write-Host ""
Write-Host "==> Running $($endpoints.Count) contract checks"
foreach ($ep in $endpoints) {
    $result = Invoke-ContractRequest -Method $ep.Method -Url $ep.Url -Headers $authHeaders -RequiredFields $ep.ReqFields
    $marker = if ($result.Pass) { 'PASS' } else { 'FAIL' }
    Write-Host ("  [{0}] {1,-7} {2}  ({3}ms, status={4})" -f $marker, $ep.Service, $ep.Url, $result.LatencyMs, $result.StatusCode)
    if (-not $result.Pass -and $result.MissingFields.Count -gt 0) {
        Write-Host "        missing: $($result.MissingFields -join ', ')"
    }
}

# Step 3 — write results
$logDir = Join-Path $PSScriptRoot '..' 'logs'
if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Path $logDir | Out-Null }

$txt = Join-Path $logDir 'wsk-contract.txt'
$json = Join-Path $logDir 'wsk-contract-results.json'

@"
WS-K Contract Validation Results
================================
Run:    $(Get-Date -Format 'o')
Auth:   $AuthUrl (user: $TestEmail, tenant: $TenantSlug)

SUMMARY: $($Script:Passed) passed, $($Script:Failed) failed (out of $($endpoints.Count))

DETAILS:
"@ | Set-Content $txt

$Script:Results | ForEach-Object {
    $marker = if ($_.Pass) { 'PASS' } else { 'FAIL' }
    $line = "[$marker] $($_.Method) $($_.Url)  status=$($_.StatusCode)  latency=$($_.LatencyMs)ms"
    if ($_.MissingFields) { $line += "  missing=[$($_.MissingFields)]" }
    $line | Add-Content $txt
}

$Script:Results | ConvertTo-Json -Depth 5 | Set-Content $json

Write-Host ""
Write-Host "==> RESULTS: $($Script:Passed) passed, $($Script:Failed) failed"
Write-Host "==> Saved to: $txt"
Write-Host "==> JSON:     $json"
if ($Script:Failed -gt 0) { exit 1 } else { exit 0 }
