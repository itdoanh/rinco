# WS-K Contract Validation Suite
#
# Walks every REST endpoint in the 10 critical Go services, calls each
# with a valid auth token, verifies the response shape, records latency.
#
# Usage:
#   .\scripts\wsk-contract-test.ps1
#
# Outputs:
#   logs/wsk-contract.txt            human-readable results
#   logs/wsk-contract-results.json   machine-readable
#   $LASTEXITCODE = 0 if all pass, 1 if any fail

$ErrorActionPreference = 'Continue'
$Script:Results = @()
$Script:Passed = 0
$Script:Failed = 0

# ---- Configuration ----
# Override any URL by setting the corresponding env var before running.
function Get-Url {
    param([string]$Key, [string]$Default)
    $v = [Environment]::GetEnvironmentVariable($Key)
    if ($v -and $v -ne '') { return $v } else { return $Default }
}

$AuthUrl         = Get-Url 'AUTH_URL'         'http://localhost:8081'
$TenantUrl       = Get-Url 'TENANT_URL'       'http://localhost:8082'
$CrmUrl          = Get-Url 'CRM_URL'          'http://localhost:8083'
$LeadUrl         = Get-Url 'LEAD_URL'         'http://localhost:8085'
$EmailUrl        = Get-Url 'EMAIL_URL'        'http://localhost:8087'
$NotificationUrl = Get-Url 'NOTIFICATION_URL' 'http://localhost:8088'
$SearchUrl       = Get-Url 'SEARCH_URL'       'http://localhost:8097'
$AnalyticsUrl    = Get-Url 'ANALYTICS_URL'    'http://localhost:8099'
$MetaCapiUrl     = Get-Url 'META_CAPI_URL'    'http://localhost:8098'
$BillingUrl      = Get-Url 'BILLING_URL'      'http://localhost:8095'
$TestEmail       = Get-Url 'TEST_EMAIL'       'admin@apexfintech.vn'
$TestPassword    = Get-Url 'TEST_PASSWORD'    'rinco_dev_password'
$TenantSlug      = Get-Url 'TENANT_SLUG'      'apexfintech'
$TimeoutSec      = 10

# ---- Helper ----
function Invoke-ContractRequest {
    param(
        [string]$Method,
        [string]$Url,
        [hashtable]$Headers = @{},
        [string]$Body = $null,
        [string[]]$RequiredFields = @()
    )
    $reqStart = Get-Date
    $statusCode = 0
    $body = $null
    try {
        $headers = @{
            'Content-Type' = 'application/json'
            'Accept'       = 'application/json'
        }
        foreach ($k in $Headers.Keys) { $headers[$k] = $Headers[$k] }
        $params = @{
            Uri             = $Url
            Method          = $Method
            Headers         = $headers
            UseBasicParsing = $true
            TimeoutSec      = $TimeoutSec
        }
        if ($Body) { $params.Body = $Body }
        $res = Invoke-WebRequest @params
        $statusCode = [int]$res.StatusCode
        $body = $res.Content
    } catch {
        $statusCode = 0
        $body = $_.Exception.Message
    }
    $elapsedMs = [int]((Get-Date) - $reqStart).TotalMilliseconds

    # Verify required fields
    $missingFields = @()
    if ($statusCode -ge 200 -and $statusCode -lt 300 -and $RequiredFields.Count -gt 0) {
        try {
            $parsed = $body | ConvertFrom-Json
            foreach ($field in $RequiredFields) {
                $parts = $field.Split('.')
                $node = $parsed
                $found = $true
                foreach ($p in $parts) {
                    if ($node.PSObject.Properties[$p]) {
                        $node = $node.$p
                    } else {
                        $missingFields += $field
                        $found = $false
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
        Url           = $Url
        Method        = $Method
        StatusCode    = $statusCode
        LatencyMs     = $elapsedMs
        RequiredFields = ($RequiredFields -join ', ')
        MissingFields = ($missingFields -join ', ')
        Pass          = $pass
    }
    if ($pass) { $Script:Passed++ } else { $Script:Failed++ }

    return @{
        Pass = $pass
        StatusCode = $statusCode
        LatencyMs = $elapsedMs
        MissingFields = $missingFields
    }
}

# ---- Login ----
Write-Host "==> Login to auth-service"
$loginBody = @{
    email       = $TestEmail
    password   = $TestPassword
    tenant_slug = $TenantSlug
} | ConvertTo-Json -Compress

try {
    $res = Invoke-WebRequest -Uri "$AuthUrl/v1/auth/login" -Method POST `
        -Body $loginBody -ContentType 'application/json' -UseBasicParsing -TimeoutSec $TimeoutSec
    $parsed = $res.Content | ConvertFrom-Json
    $token = $parsed.access_token
    if (-not $token) { throw 'No access_token in login response' }
    Write-Host "==> Auth token obtained (len=$($token.Length))"
} catch {
    Write-Host "FATAL: auth login failed: $_"
    exit 2
}
$authHeaders = @{ 'Authorization' = "Bearer $token" }

# ---- Endpoint matrix ----
# Each entry: service name, URL, HTTP method, required top-level JSON fields
$endpoints = @(
    @{ Svc='auth';      Url="$AuthUrl/v1/auth/me";                       Method='GET';  Fields=@('id','tenant_id','email') },
    @{ Svc='tenant';   Url="$TenantUrl/v1/tenants/by-domain/apexfintech.vn"; Method='GET'; Fields=@('tenant_id') },
    @{ Svc='crm';      Url="$CrmUrl/api/crm/contacts?page=1&limit=10";  Method='GET'; Fields=@() },
    @{ Svc='lead';     Url="$LeadUrl/api/leads?page=1&limit=10";         Method='GET'; Fields=@() },
    @{ Svc='email';    Url="$EmailUrl/v1/email/templates";               Method='GET'; Fields=@() },
    @{ Svc='notif';    Url="$NotificationUrl/v1/notifications/me";        Method='GET'; Fields=@() },
    @{ Svc='search';   Url="$SearchUrl/v1/search?q=test";                 Method='GET'; Fields=@() },
    @{ Svc='analytics';Url="$AnalyticsUrl/v1/analytics/events/recent";    Method='GET'; Fields=@() },
    @{ Svc='meta';     Url="$MetaCapiUrl/v1/meta/health";                Method='GET'; Fields=@() },
    @{ Svc='billing';  Url="$BillingUrl/v1/billing/plans";               Method='GET'; Fields=@() }
)

Write-Host ""
Write-Host "==> Running $($endpoints.Count) contract checks"
foreach ($ep in $endpoints) {
    $result = Invoke-ContractRequest -Method $ep.Method -Url $ep.Url -Headers $authHeaders -RequiredFields $ep.Fields
    $marker = if ($result.Pass) { 'PASS' } else { 'FAIL' }
    $fieldInfo = if ($result.MissingFields.Count -gt 0) { " missing=$($result.MissingFields -join ',')" } else { '' }
    $urlShort = $ep.Url -replace 'http[s]?://[^/]+', ''  # strip host
    Write-Host ("  [{0,-4}] {1,-10} {2,45} {3,6}ms status={4}{5}" -f $marker, $ep.Svc, $urlShort, $result.LatencyMs, $result.StatusCode, $fieldInfo)
}

# ---- Write results ----
# Determine logs dir relative to script location
$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = if ($scriptDir -match 'scripts[/\\]?$') { $scriptDir -replace '[/\\]scripts$', '' } else { (Get-Location).Path }
$logDir = Join-Path $repoRoot 'logs'
if (-not (Test-Path $logDir)) { New-Item -ItemType Directory -Path $logDir -Force | Out-Null }

$txt = Join-Path $logDir 'wsk-contract.txt'
$json = Join-Path $logDir 'wsk-contract-results.json'

"WS-K Contract Validation Results
================================
Run:    $(Get-Date -Format 'o')
Auth:   $AuthUrl  User: $TestEmail  Tenant: $TenantSlug

SUMMARY: $($Script:Passed) passed, $($Script:Failed) failed (of $($endpoints.Count))

DETAILS:" | Set-Content $txt

foreach ($r in $Script:Results) {
    $marker = if ($r.Pass) { 'PASS' } else { 'FAIL' }
    $line = "[{0,-4}] {1} {2}  status={3}  latency={4}ms" -f $marker, $r.Method, $r.Url, $r.StatusCode, $r.LatencyMs
    if ($r.MissingFields) { $line += "  missing=[$($r.MissingFields)]" }
    Add-Content -Path $txt -Value $line
}

$Script:Results | ConvertTo-Json -Depth 5 | Set-Content $json

Write-Host ""
Write-Host "==> $Script:Passed passed, $Script:Failed failed  |  Results: $txt"
Write-Host "==> JSON: $json"
if ($Script:Failed -gt 0) { exit 1 } else { exit 0 }
