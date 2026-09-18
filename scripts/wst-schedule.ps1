# WS-T Nightly Schedule Registration Script
# Registers a Windows Task Scheduler task to run the WS-T suite at 02:00 daily.
# Requires PowerShell as Administrator.

param(
    [string]$TaskName = "RINCO-WS-T-Nightly",
    [string]$Description = "RINCO WS-T E2E test suite — runs every night at 02:00",
    [string]$Time = "02:00",
    [string]$WorkingDir = "C:\code\RINCO\frontend\e2e",
    [string]$LogFile = "C:\code\RINCO\logs\wst-nightly.txt"
)

$ErrorActionPreference = "Stop"

Write-Output "=== WS-T Nightly Schedule Registration ==="
Write-Output "Task: $TaskName"
Write-Output "Time: $Time daily"
Write-Output "Working dir: $WorkingDir"
Write-Output "Log: $LogFile"
Write-Output ""

# Check if task already exists
$existing = Get-ScheduledTask -TaskName $TaskName -ErrorAction SilentlyContinue
if ($existing) {
    Write-Output "Task already exists. Re-registering..."
    Unregister-ScheduledTask -TaskName $TaskName -Confirm:$false
}

# Create action: run npm test
$action = New-ScheduledTaskAction `
    -WorkingDirectory $WorkingDir `
    -Execute "npm.cmd" `
    -Argument "test -- --reporter=list,html"

# Create trigger: daily at specified time
$trigger = New-ScheduledTaskTrigger -Daily -At $Time

# Create principal: run as current user
$principal = New-ScheduledTaskPrincipal `
    -UserId $env:USERNAME `
    -LogonType Interactive `
    -RunLevel Highest

# Task settings
$settings = New-ScheduledTaskSettingsSet `
    -AllowStartIfOnBatteries `
    -DontStopIfGoingOnBatteries `
    -StartWhenAvailable `
    -RunOnlyIfNetworkAvailable:$false `
    -DontStopOnIdleEnd

# Register the task
Register-ScheduledTask `
    -TaskName $TaskName `
    -Description $Description `
    -Action $action `
    -Trigger $trigger `
    -Principal $principal `
    -Settings $settings `
    -Force

# Add a second trigger: on failure, also capture the output
Write-Output ""
Write-Output "Task registered. Verifying..."

$verify = Get-ScheduledTask -TaskName $TaskName
if ($verify) {
    Write-Output "SUCCESS: Task '$TaskName' registered."
    Write-Output "  State: $($verify.State)"
    Write-Output "  Next Run: $($verify.NextRunTime)"
    Write-Output "  Description: $($verify.Description)"
    Write-Output ""
    Write-Output "To run immediately (dry-run):"
    Write-Output "  Start-ScheduledTask -TaskName '$TaskName'"
    Write-Output ""
    Write-Output "To unregister:"
    Write-Output "  Unregister-ScheduledTask -TaskName '$TaskName' -Confirm:`$false"
} else {
    Write-Output "ERROR: Task registration failed."
    exit 1
}

Write-Output ""
Write-Output "NOTE: The nightly task expects dev servers (npm run dev on ports 3000-3003)"
Write-Output "to already be running. If they are not, tests will fail gracefully with"
Write-Output "a 'ERR_CONNECTION_REFUSED' in the output log (see wst-nightly-*.txt files)."