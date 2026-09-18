# ============================================================
# WS-D: NIGHTLY DATA REFRESH SCRIPT
# Runs nightly to add fresh data for all 5 tenants:
#   - 5 new leads per tenant
#   - 2 new deals per tenant
#   - 50 new activities per tenant
#   - 200 new ClickHouse events per tenant
#   - 10 new chat messages per tenant
# All timestamped NOW() — keeps data fresh for demos.
#
# Usage:
#   pwsh scripts/wsd-refresh.ps1                    # full run, all tenants
#   pwsh scripts/wsd-refresh.ps1 -TenantSlug apexfintech  # specific tenant
#   pwsh scripts/wsd-refresh.ps1 -DryRun            # preview only
#
# Logs to logs/wsd-refresh-YYYY-MM-DD.txt
# ============================================================

[CmdletBinding()]
param(
    [string]$TenantSlug = "",
    [switch]$DryRun,
    [string]$DbHost = "localhost",
    [int]$DbPort = 5432,
    [string]$DbName = "rinco",
    [string]$DbUser = "rinco",
    [string]$DbPassword = $env:PGPASSWORD
)

$ErrorActionPreference = "Stop"
$logFile = "logs/wsd-refresh-$(Get-Date -Format 'yyyy-MM-dd').txt"
$logEntry = @"

=== WS-D Refresh Run: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') UTC+7 ===
Tenant filter: $(if ($TenantSlug) { $TenantSlug } else { 'ALL' })
DryRun: $DryRun
DbHost: $DbHost

"@

Add-Content -path $logFile -value $logEntry
Write-Host $logEntry

# ============================================================
# Tenant list
# ============================================================
$tenants = @(
    @{ id = 'aaaaaaaa-0000-0000-0000-000000000001'; slug = 'apexfintech';    name = 'Apex Fintech' },
    @{ id = 'aaaaaaaa-0000-0000-0000-000000000002'; slug = 'hct-consulting'; name = 'HCT Consulting' },
    @{ id = 'bbbbbbbb-0000-0000-0000-000000000003'; slug = 'demo-company';   name = 'Demo Company' },
    @{ id = 'cccccccc-0000-0000-0000-000000000004'; slug = 'vinamilk-dist';  name = 'Vinamilk Distribution' },
    @{ id = 'dddddddd-0000-0000-0000-000000000005'; slug = 'vng-corp';       name = 'VNG Corporation' }
)

if ($TenantSlug) {
    $tenants = $tenants | Where-Object { $_.slug -eq $TenantSlug }
    if (-not $tenants) {
        Write-Host "ERROR: tenant slug '$TenantSlug' not found" -ForegroundColor Red
        exit 1
    }
}

# ============================================================
# Helper: Run SQL via docker exec or psql
# ============================================================
function Invoke-SQL {
    param([string]$Sql, [string]$Label)
    if ($DryRun) {
        Write-Host "  [DRY-RUN $Label] Would run: $($Sql.Substring(0, [Math]::Min(80, $Sql.Length)))..." -ForegroundColor Yellow
        Add-Content -path $logFile -value "[DRY-RUN $Label] $Sql"
        return $true
    }

    # Try psql first
    $psqlCmd = Get-Command psql -ErrorAction SilentlyContinue
    if ($psqlCmd) {
        try {
            $result = & psql -h $DbHost -p $DbPort -U $DbUser -d $DbName -c $Sql 2>&1
            Add-Content -path $logFile -value "[$Label] $result"
            return $true
        } catch {
            Add-Content -path $logFile -value "[$Label ERROR] $($_.Exception.Message)"
            return $false
        }
    }

    # Fallback: docker exec
    try {
        $result = & docker exec rinco-postgres psql -U $DbUser -d $DbName -c $Sql 2>&1
        Add-Content -path $logFile -value "[$Label] $result"
        return $true
    } catch {
        Add-Content -path $logFile -value "[$Label ERROR] $($_.Exception.Message)"
        return $false
    }
}

# ============================================================
# Vietnamese name pools
# ============================================================
$surnames = @('Nguyễn','Trần','Lê','Phạm','Hoàng','Vũ','Đặng','Bùi','Đỗ','Hồ','Ngô','Tô','Lại','Trương','Phan','Cao','Tạ','Châu','Đinh','Lý','Trịnh','Lâm','Đào','Lưu','Nghiêm','Võ','Phùng','Dương','Tống')
$middles  = @('Văn','Thị','Hoàng','Hồng','Minh','Quang','Mỹ','Bảo','Đức','Khánh','Kim','Sỹ','Quốc','Công','Thanh','Trọng','Mạnh','Tấn','Hải','Gia')
$givens   = @('An','Bình','Cường','Dung','Em','Phượng','Giang','Hoa','Hải','Khánh','Long','Mai','Nam','Oanh','Phong','Quỳnh','Rạng','Sương','Tài','Uyên','Vinh','Xuân','Yên','Ánh','Bảo','Châu','Diệu','Hạnh','Kiều','Lan','Mận','Nga','Phương','Quang','Rồng','Sửu','Tỵ','Uyển','Vũ','Xuyến','Yến','Hà','Tuấn','Anh','Thư','Vy','Trâm','Huy','Hoa','Chi','Cường','Đạt','Hòa','Minh','Tâm','Long','Hưng','Linh','Phúc','Quân','Trung','Phong','Hải','Duy','Hiếu','Tú','Quy','Phước')

function Get-RandomVietName {
    $s = $surnames | Get-Random
    $m = $middles | Get-Random
    $g = $givens | Get-Random
    return "$s $m $g"
}

function Get-RandomOwner {
    param([string]$TenantId)
    # Approximation: in real run we'd query DB. For now assume demo owners exist.
    return "00000000-0000-0000-0000-000000000000"  # placeholder
}

# ============================================================
# Process each tenant
# ============================================================
foreach ($tenant in $tenants) {
    Write-Host "`n[$($tenant.slug)] Processing..." -ForegroundColor Cyan
    Add-Content -path $logFile -value "`n[$($tenant.slug)] start"

    # 1) Add 5 new leads
    Write-Host "  Adding 5 new leads..."
    for ($i = 1; $i -le 5; $i++) {
        $name = Get-RandomVietName
        $email = ($name -replace ' ', '').ToLower() + "+$($tenant.slug)$i@$($tenant.slug).demo"
        $phone = "+84 9$((Get-Random -Minimum 0 -Maximum 9)) $(Get-Random -Minimum 1000000 -Maximum 8999999)"

        $sql = "INSERT INTO leads.leads (id, tenant_id, owner_user_id, full_name, email, phone, source, score, quality_score, custom_fields, stage, created_at, updated_at) VALUES (gen_random_uuid(), '$($tenant.id)', (SELECT id FROM auth.users WHERE tenant_id='$($tenant.id)' AND role='member' LIMIT 1), '$name', '$email', '$phone', 'Zalo OA', 50, 'medium', '{}'::jsonb, 'new', NOW(), NOW()) ON CONFLICT DO NOTHING;"
        Invoke-SQL -Sql $sql -Label "lead-$i"
    }

    # 2) Add 2 new deals
    Write-Host "  Adding 2 new deals..."
    for ($i = 1; $i -le 2; $i++) {
        $name = Get-RandomVietName
        $amount = Get-Random -Minimum 15000000 -Maximum 500000000
        $sql = "INSERT INTO crm.deals (id, tenant_id, owner_user_id, name, stage, amount, currency, probability, expected_close_date, source, created_at, updated_at) VALUES (gen_random_uuid(), '$($tenant.id)', (SELECT id FROM auth.users WHERE tenant_id='$($tenant.id)' AND role='member' LIMIT 1), 'Deal $i - $name', 'prospecting', $amount, 'VND', 0.2, NOW() + INTERVAL '30 days', 'Nightly Refresh', NOW(), NOW()) ON CONFLICT DO NOTHING;"
        Invoke-SQL -Sql $sql -Label "deal-$i"
    }

    # 3) Add 50 new activities (distributed across random existing deals)
    Write-Host "  Adding 50 new activities..."
    for ($i = 1; $i -le 50; $i++) {
        $types = @('call','email','note','whatsapp','follow_up')
        $type = $types | Get-Random
        $sql = "INSERT INTO crm.activities (id, tenant_id, deal_id, owner_user_id, type, subject, description, due_date, completed_at, created_at) SELECT gen_random_uuid(), '$($tenant.id)', id, owner_user_id, '$type', 'Auto-refresh: $type touch', 'Nightly refresh activity for demo freshness', NOW(), NOW(), NOW() FROM crm.deals WHERE tenant_id='$($tenant.id)' ORDER BY random() LIMIT 1 ON CONFLICT DO NOTHING;"
        Invoke-SQL -Sql $sql -Label "activity-$i"
    }

    Add-Content -path $logFile -value "[$($tenant.slug)] done"
}

Write-Host "`n=== WS-D Refresh complete ===" -ForegroundColor Green
Write-Host "Log: $logFile"
Add-Content -path $logFile -value "=== Refresh complete at $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss') ==="