# ============================================================
# WS-D: WINDOWS TASK SCHEDULER REGISTRATION
# Registers a nightly Windows Task Scheduler job to run wsd-refresh.ps1
# at 01:00 every day.
#
# Usage:
#   pwsh scripts/wsd-schedule.ps1                    # register
#   pwsh scripts/wsd-schedule.ps1 -Action unregister  # unregister
# ============================================================

[CmdletBinding()]
param(
    [ValidateSet('register', 'unregister')]
    [string]$Action = 'register'
)

$taskName = "WS-D Nightly Data Refresh"
$scriptPath = Join-Path $PSScriptRoot "wsd-refresh.ps1"

if ($Action -eq 'unregister') {
    Write-Host "Unregistering task '$taskName'..."
    try {
        Unregister-ScheduledTask -TaskName $taskName -Confirm:$false
        Write-Host "Done." -ForegroundColor Green
    } catch {
        Write-Host "Failed to unregister: $($_.Exception.Message)" -ForegroundColor Red
    }
    exit 0
}

Write-Host "Registering task '$taskName'..."
Write-Host "  Script: $scriptPath"
Write-Host "  Schedule: Daily 01:00 UTC+7 (18:00 UTC previous day)"

try {
    # Build action: run pwsh with the refresh script
    $psExec = (Get-Command pwsh -ErrorAction SilentlyContinue).Source
    if (-not $psExec) {
        $psExec = (Get-Command powershell -ErrorAction SilentlyContinue).Source
    }
    if (-not $psExec) {
        Write-Host "ERROR: pwsh or powershell not found in PATH" -ForegroundColor Red
        exit 1
    }
    Write-Host "  Executor: $psExec"

    $action = New-ScheduledTaskAction -Execute $psExec -Argument "-NoProfile -ExecutionPolicy Bypass -File `"$scriptPath`""
    $trigger = New-ScheduledTaskTrigger -Daily -At "01:00"
    $settings = New-ScheduledTaskSettingsSet -AllowStartIfOnBatteries -DontStopIfGoingOnBatteries -StartWhenAvailable
    $principal = New-ScheduledTaskPrincipal -UserId "$env:USERDOMAIN\$env:USERNAME" -LogonType S4U -RunLevel Highest

    Register-ScheduledTask -TaskName $taskName -Action $action -Trigger $trigger -Settings $settings -Principal $principal -Description "WS-D nightly data refresh: 5 leads, 2 deals, 50 activities per tenant for 5 tenants. Logs to logs/wsd-refresh-YYYY-MM-DD.txt"

    Write-Host "Done." -ForegroundColor Green
    Write-Host "Verify with: Get-ScheduledTask -TaskName '$taskName'"
} catch {
    Write-Host "Failed to register: $($_.Exception.Message)" -ForegroundColor Red
    Write-Host "You may need to run PowerShell as Administrator."
    exit 1
}