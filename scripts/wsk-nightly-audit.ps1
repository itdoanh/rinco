# WS-K Nightly Audit
#
# Runs once per day at 03:00 (schedule via Windows Task Scheduler):
#   - go mod verify on all Go services
#   - pip-audit on all Python requirements
#   - npm audit (--omit=dev) on all 4 frontends
#   - cargo audit on Rust services (if cargo-audit installed)
#   - k6 baseline suite (scaled down: 10 VUs)
#   - Writes logs/wsk-nightly-<date>.txt
#
# Schedule (run once, requires admin PowerShell):
#   $Action = New-ScheduledTaskAction -Execute 'powershell.exe' `
#     -Argument '-NoProfile -ExecutionPolicy Bypass -File "C:\code\RINCO\scripts\wsk-nightly-audit.ps1"'
#   $Trigger = New-ScheduledTaskTrigger -Daily -At 03:00
#   Register-ScheduledTask -TaskName 'WSK-NightlyAudit' `
#     -Action $Action -Trigger $Trigger -Description 'WS-K nightly security audit'

[CmdletBinding()]
param(
    [string]$K6Vus = 10,
    [int]$K6Duration = 30,      # seconds
    [switch]$SkipK6,            # skip load tests (faster nightly)
    [switch]$SkipFrontend       # skip npm audit (faster nightly)
)

$ErrorActionPreference = 'Continue'
# Resolve $root to the RINCO repo root.  When running as a file, we can
# use the known parent of the scripts/ directory.  If running interactively
# without a script path, fall back to $PWD.
$root = $PWD.Path
# Detect if we're running from within the scripts/ directory.
$resolved = if ($root -match 'scripts[/\\]?$') {
    $root -replace '[/\\]scripts[/\\]?$', ''
} else {
    $root
}
Set-Location $resolved

$date = Get-Date -Format 'yyyy-MM-dd'
$logDir = Join-Path $root 'logs'
if (-not (Test-Path $logDir)) {
    New-Item -ItemType Directory -Path $logDir -Force | Out-Null
}
$logFile = Join-Path $logDir "wsk-nightly-$date.txt"

function Write-Section {
    param([string]$Title)
    Write-Host "`n=== $Title ==="
    Add-Content -Path $logFile -Value "`n=== $Title ==="
}

function Run-And-Log {
    param([string]$Cmd)
    Write-Host ">>> $Cmd"
    Add-Content -Path $logFile -Value ">>> $Cmd"
    try {
        $out = cmd /c $Cmd 2>&1
        if ($out) {
            Add-Content -Path $logFile -Value ($out | Out-String)
        }
        Write-Host "<<< OK"
        Add-Content -Path $logFile -Value "<<< OK"
    } catch {
        Write-Host "<<< FAILED: $_"
        Add-Content -Path $logFile -Value "<<< FAILED: $_"
    }
}

$banner = @"
WS-K Nightly Audit
==================
Date:  $(Get-Date -Format 'o')
Root:  $root
"@
$banner | Set-Content $logFile
Write-Host $banner

# ---------------------------------------------------------------------------
# 1. Go module verify
# ---------------------------------------------------------------------------
Write-Section "1. Go module verify"
Get-ChildItem -Path 'services' -Directory -ErrorAction SilentlyContinue |
    Where-Object { Test-Path (Join-Path $_.FullName 'go.mod') } |
    ForEach-Object {
        Run-And-Log "cd /d $($_.FullName) && go mod verify"
    }

# ---------------------------------------------------------------------------
# 2. pip-audit (Python services)
# ---------------------------------------------------------------------------
Write-Section "2. pip-audit on Python services"
$pythonServices = @('lead-scoring', 'rag-chatbot', 'ai-sre', 'stt-service')
foreach ($svc in $pythonServices) {
    $reqPath = Join-Path $root "services/$svc/requirements.txt"
    if (Test-Path $reqPath) {
        Run-And-Log "pip-audit -r `"$reqPath`" --no-deps"
    } else {
        Add-Content -Path $logFile -Value "(no requirements.txt at $reqPath)"
    }
}

# ---------------------------------------------------------------------------
# 3. npm audit on frontends
# ---------------------------------------------------------------------------
if (-not $SkipFrontend) {
    Write-Section "3. npm audit on frontends"
    $frontends = @('landing', 'admin-portal', 'tenant-site', 'meeting-ui')
    foreach ($fe in $frontends) {
        $fePath = Join-Path $root "frontend/$fe"
        if (Test-Path $fePath) {
            Run-And-Log "cd /d `"$fePath`" && npm audit --omit=dev"
        }
    }
}

# ---------------------------------------------------------------------------
# 4. cargo audit (Rust services, if installed)
# ---------------------------------------------------------------------------
Write-Section "4. cargo audit on Rust services"
$rustServices = @('chat-engine', 'webrtc-sfu', 'recording-service')
$cargoAuditExe = Get-Command cargo-audit -ErrorAction SilentlyContinue
if (-not $cargoAuditExe) {
    $cargoAuditExe = Get-ChildItem -Path 'C:\Users\doanh\.cargo\bin' -Filter 'cargo-audit.exe' -ErrorAction SilentlyContinue |
        Select-Object -First 1 -ExpandProperty FullName
    if ($cargoAuditExe) {
        $cargoAuditExe = @{ Source = $cargoAuditExe }
    }
}
foreach ($svc in $rustServices) {
    $svcPath = Join-Path $root "services/$svc"
    if (Test-Path (Join-Path $svcPath 'Cargo.toml')) {
        if ($cargoAuditExe -and (Get-Command cargo -ErrorAction SilentlyContinue)) {
            Run-And-Log "cd /d `"$svcPath`" && cargo audit"
        } else {
            Add-Content -Path $logFile -Value "(cargo-audit not installed; skipping $svc)"
        }
    }
}

# ---------------------------------------------------------------------------
# 5. k6 baseline suite (scaled down)
# ---------------------------------------------------------------------------
if (-not $SkipK6) {
    Write-Section "5. k6 baseline load tests (VUs=$K6Vus, dur=$($K6Duration)s)"
    $k6Path = Get-Command k6 -ErrorAction SilentlyContinue
    if (-not $k6Path) {
        $k6Candidates = @('C:\Program Files\k6\k6.exe', 'C:\Program Files (x86)\k6\k6.exe')
        foreach ($c in $k6Candidates) {
            if (Test-Path $c) { $k6Path = @{ Source = $c }; break }
        }
    }
    if (-not $k6Path) {
        $k6Exe = Get-ChildItem -Path 'C:\Users\doanh\AppData\Local\Microsoft\WinGet\Packages' `
            -Recurse -Filter 'k6.exe' -ErrorAction SilentlyContinue |
            Select-Object -First 1 -ExpandProperty FullName
        if ($k6Exe) { $k6Path = @{ Source = $k6Exe } }
    }
    $k6Scripts = @(
        @{ Script = 'wsk-k6-auth.yml'; VUs = $K6Vus },
        @{ Script = 'wsk-k6-crm-read.yml'; VUs = [Math]::Max(5, [int]([int]$K6Vus / 2)) },
        @{ Script = 'wsk-k6-lead-ingest.yml'; VUs = [Math]::Max(5, [int]([int]$K6Vus / 2)) }
    )
    foreach ($k6 in $k6Scripts) {
        $scriptPath = Join-Path $root "scripts/$($k6.Script)"
        if ($k6Path -and (Test-Path $scriptPath)) {
            $k6Bin = $k6Path.Source
            Run-And-Log "cd /d `"$root`" && `"$k6Bin`" run --vus $($k6.VUs) --duration $($K6Duration)s `"$scriptPath`""
        } else {
            Add-Content -Path $logFile -Value "(k6 or $scriptPath not available; skipping)"
        }
    }
}

# ---------------------------------------------------------------------------
# 6. Docker health check
# ---------------------------------------------------------------------------
Write-Section "6. Docker health check"
if (Get-Command docker -ErrorAction SilentlyContinue) {
    Run-And-Log "docker ps --format ""{{.Names}} {{.Status}}"""
} else {
    Add-Content -Path $logFile -Value "(docker not available)"
}

$footer = @"

=== END OF NIGHTLY AUDIT ===
Run completed at $(Get-Date -Format 'o')
"@
$footer | Add-Content $logFile
Write-Host $footer

# Print summary to console for Task Scheduler to capture.
Write-Host ""
Write-Host "WS-K nightly audit complete.  Results: $logFile"
