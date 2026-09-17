# ============================================
# RINCO Comprehensive Demo Seed Orchestrator
# Loop 202 — Database Schemas + Seed Data
# ============================================
# Runs all seed scripts across:
#   - PostgreSQL (migrations/sql + migrations/seed)
#   - MongoDB (seed/external/mongo)
#   - ScyllaDB (seed/external/scylla)
#   - ClickHouse (seed/external/clickhouse)
#   - Valkey/Redis (seed/external/valkey)
#   - MinIO (seed/external/minio)
# ============================================

[CmdletBinding()]
param(
    [string]$PostgresHost = "localhost",
    [int]$PostgresPort = 5433,
    [string]$PostgresUser = "rinco",
    [string]$PostgresPassword = "rinco_dev_password",
    [string]$PostgresDb = "rinco",
    [string]$MongoHost = "localhost",
    [int]$MongoPort = 27017,
    [string]$MongoDb = "rinco",
    [string]$MongoUser = "rinco",
    [string]$MongoPassword = "rinco_dev_password",
    [string]$ScyllaHost = "localhost",
    [int]$ScyllaPort = 9042,
    [string]$ClickHouseHost = "localhost",
    [int]$ClickHousePort = 8123,
    [string]$ClickHouseUser = "rinco",
    [string]$ClickHousePassword = "rinco_dev_password",
    [string]$ValkeyHost = "localhost",
    [int]$ValkeyPort = 6379,
    [string]$ValkeyPassword = "rinco_dev_password",
    [string]$MinIOEndpoint = "http://localhost:9000",
    [string]$MinIOUser = "rinco",
    [string]$MinIOPassword = "rinco_dev_password",
    [switch]$ApplyMigrations = $true,
    [switch]$SeedPostgres = $true,
    [switch]$SeedMongo = $true,
    [switch]$SeedScylla = $true,
    [switch]$SeedClickHouse = $true,
    [switch]$SeedValkey = $true,
    [switch]$SeedMinIO = $true,
    [switch]$Force = $false
)

$ErrorActionPreference = 'Continue'
$ProjectRoot = (Resolve-Path "$PSScriptRoot\..").Path
$SeedDir = Join-Path $ProjectRoot "migrations\seed"
$MigrationDir = Join-Path $ProjectRoot "migrations\sql"
$ExternalSeedDir = Join-Path $ProjectRoot "seed\external"

# ============================================
# BANNER
# ============================================
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  RINCO Demo Seed — Loop 202 (Database)" -ForegroundColor Cyan
Write-Host "  Project: $ProjectRoot" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# ============================================
# PRE-FLIGHT CHECKS
# ============================================
function Test-Tool($name, $cmd) {
    if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
        Write-Host "[!] $name not found in PATH: $cmd" -ForegroundColor Yellow
        return $false
    }
    Write-Host "[OK] $name available" -ForegroundColor Green
    return $true
}

Write-Host "--- Pre-flight tool checks ---" -ForegroundColor Yellow
$toolCheck = @{ psql = $false; mongosh = $false; cqlsh = $false; "clickhouse-client" = $false; "redis-cli" = $false; mc = $false }
$toolCheck.psql = Test-Tool "psql" "psql" | Out-Null; if ($toolCheck.psql) { } else { $toolCheck.psql = $false }
$toolCheck.mongosh = Test-Tool "mongosh" "mongosh" | Out-Null
$toolCheck.cqlsh = Test-Tool "cqlsh" "cqlsh" | Out-Null
$toolCheck.clickhouse-client = Test-Tool "clickhouse-client" "clickhouse-client" | Out-Null
$toolCheck."redis-cli" = Test-Tool "redis-cli" "redis-cli" | Out-Null
$toolCheck.mc = Test-Tool "mc (MinIO)" "mc" | Out-Null
Write-Host ""

# ============================================
# HELPER FUNCTIONS
# ============================================
function Test-Service($name, $testCmd) {
    Write-Host "Checking $name..." -NoNewline
    try {
        $result = & $testCmd 2>&1
        if ($LASTEXITCODE -eq 0 -or $result -match "PONG|ready|Ok|UP|connected") {
            Write-Host " [OK]" -ForegroundColor Green
            return $true
        } else {
            Write-Host " [FAIL]" -ForegroundColor Red
            return $false
        }
    } catch {
        Write-Host " [FAIL]" -ForegroundColor Red
        return $false
    }
}

function Run-Step($name, $block) {
    Write-Host ""
    Write-Host "============================================" -ForegroundColor Magenta
    Write-Host "  $name" -ForegroundColor Magenta
    Write-Host "============================================" -ForegroundColor Magenta
    & $block
}

# ============================================
# SERVICE CHECKS
# ============================================
Write-Host "--- Pre-flight service checks ---" -ForegroundColor Yellow
$serviceStatus = @{}
$serviceStatus.postgres = if ($SeedPostgres) { Test-Service "PostgreSQL" { psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $PostgresDb -c "SELECT 1" -t 2>&1 } } else { $true }
$serviceStatus.mongo = if ($SeedMongo) { Test-Service "MongoDB" { mongosh --host $MongoHost:$MongoPort --username $MongoUser --password $MongoPassword --authenticationDatabase admin --quiet --eval 'db.runCommand({ping:1})' 2>&1 } } else { $true }
$serviceStatus.scylla = if ($SeedScylla) { Test-Service "ScyllaDB" { cqlsh $ScyllaHost $ScyllaPort -e "describe cluster" 2>&1 } } else { $true }
$serviceStatus.clickhouse = if ($SeedClickHouse) { Test-Service "ClickHouse" { curl.exe -s "http://${ClickHouseHost}:${ClickHousePort}/ping" 2>&1 } } else { $true }
$serviceStatus.valkey = if ($SeedValkey) { Test-Service "Valkey" { redis-cli -h $ValkeyHost -p $ValkeyPort -a $ValkeyPassword --no-auth-warning ping 2>&1 } } else { $true }
$serviceStatus.minio = if ($SeedMinIO) { Test-Service "MinIO" { curl.exe -s "${MinIOEndpoint}/minio/health/live" 2>&1 } } else { $true }
Write-Host ""

# ============================================
# STEP 1: POSTGRESQL MIGRATIONS
# ============================================
if ($ApplyMigrations -and $SeedPostgres -and $serviceStatus.postgres) {
    Run-Step "STEP 1: Apply PostgreSQL Migrations" {
        $env:PGPASSWORD = $PostgresPassword

        # Run all SQL migrations in order
        $migrationFiles = Get-ChildItem -Path $MigrationDir -Filter "*.sql" | Sort-Object Name

        foreach ($file in $migrationFiles) {
            Write-Host "  Applying migration: $($file.Name)" -ForegroundColor White
            psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $PostgresDb -v ON_ERROR_STOP=0 -f $file.FullName 2>&1 | ForEach-Object { Write-Host "    $_" -ForegroundColor Gray }
        }

        Write-Host "[OK] PostgreSQL migrations applied" -ForegroundColor Green
    }
}

# ============================================
# STEP 2: POSTGRESQL SEED DATA
# ============================================
if ($SeedPostgres -and $serviceStatus.postgres) {
    Run-Step "STEP 2: Seed PostgreSQL (migrations/seed)" {
        $env:PGPASSWORD = $PostgresPassword

        $MasterFile = Join-Path $SeedDir "00_master.sql"
        if (-not (Test-Path $MasterFile)) {
            Write-Host "[ERROR] Master seed not found: $MasterFile" -ForegroundColor Red
        } else {
            Write-Host "Running master seed (loop 202)..." -ForegroundColor White
            Push-Location $SeedDir
            try {
                psql -h $PostgresHost -p $PostgresPort -U $PostgresUser -d $PostgresDb -v ON_ERROR_STOP=0 -f $MasterFile 2>&1 | Out-Null
                Write-Host "[OK] PostgreSQL seeded" -ForegroundColor Green
            } finally {
                Pop-Location
            }
        }
    }
}

# ============================================
# STEP 3: MONGODB
# ============================================
if ($SeedMongo -and $serviceStatus.mongo) {
    Run-Step "STEP 3: Seed MongoDB" {
        $MongoSeed = Join-Path $ExternalSeedDir "mongo\seed.js"
        if (-not (Test-Path $MongoSeed)) {
            Write-Host "[ERROR] Mongo seed not found: $MongoSeed" -ForegroundColor Red
        } else {
            Write-Host "Running mongosh seed script..." -ForegroundColor White
            mongosh --host $MongoHost:$MongoPort -u $MongoUser -p $MongoPassword --authenticationDatabase admin --quiet $MongoDb $MongoSeed 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            Write-Host "[OK] MongoDB seeded" -ForegroundColor Green
        }
    }
}

# ============================================
# STEP 4: SCYLLADB
# ============================================
if ($SeedScylla -and $serviceStatus.scylla) {
    Run-Step "STEP 4: Seed ScyllaDB" {
        $ScyllaSeed = Join-Path $ExternalSeedDir "scylla\seed.sql"
        if (-not (Test-Path $ScyllaSeed)) {
            Write-Host "[ERROR] Scylla seed not found: $ScyllaSeed" -ForegroundColor Red
        } else {
            Write-Host "Running CQL seed script..." -ForegroundColor White
            cqlsh $ScyllaHost $ScyllaPort -u cassandra -p cassandra -f $ScyllaSeed 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            Write-Host "[OK] ScyllaDB seeded" -ForegroundColor Green
        }
    }
}

# ============================================
# STEP 5: CLICKHOUSE
# ============================================
if ($SeedClickHouse -and $serviceStatus.clickhouse) {
    Run-Step "STEP 5: Seed ClickHouse Analytics" {
        $ChSeed = Join-Path $ExternalSeedDir "clickhouse\seed.sql"
        if (-not (Test-Path $ChSeed)) {
            Write-Host "[ERROR] ClickHouse seed not found: $ChSeed" -ForegroundColor Red
        } else {
            Write-Host "Running clickhouse-client seed script..." -ForegroundColor White
            clickhouse-client --host $ClickHouseHost --port $ClickHousePort --user $ClickHouseUser --password $ClickHousePassword --multiquery --queries-file $ChSeed 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            Write-Host "[OK] ClickHouse seeded" -ForegroundColor Green
        }
    }
}

# ============================================
# STEP 6: VALKEY
# ============================================
if ($SeedValkey -and $serviceStatus.valkey) {
    Run-Step "STEP 6: Seed Valkey (Redis-compatible)" {
        $ValkeySeed = Join-Path $ExternalSeedDir "valkey\seed.sh"
        if (-not (Test-Path $ValkeySeed)) {
            Write-Host "[ERROR] Valkey seed not found: $ValkeySeed" -ForegroundColor Red
        } else {
            Write-Host "Running Valkey seed bash script..." -ForegroundColor White
            bash.exe $ValkeySeed $ValkeyHost $ValkeyPort $ValkeyPassword 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            Write-Host "[OK] Valkey seeded" -ForegroundColor Green
        }
    }
}

# ============================================
# STEP 7: MINIO
# ============================================
if ($SeedMinIO -and $serviceStatus.minio) {
    Run-Step "STEP 7: Seed MinIO (Object Storage)" {
        $MinIOSeed = Join-Path $ExternalSeedDir "minio\seed.sh"
        if (-not (Test-Path $MinIOSeed)) {
            Write-Host "[ERROR] MinIO seed not found: $MinIOSeed" -ForegroundColor Red
        } else {
            Write-Host "Running MinIO seed bash script..." -ForegroundColor White
            $env:MINIO_HOST = ($MinIOEndpoint -replace "http://", "").Split(":")[0]
            $env:MINIO_API_PORT = ($MinIOEndpoint -replace "http://", "").Split(":")[1]
            $env:MINIO_USER = $MinIOUser
            $env:MINIO_PASSWORD = $MinIOPassword
            bash.exe $MinIOSeed 2>&1 | ForEach-Object { Write-Host "  $_" -ForegroundColor Gray }
            Write-Host "[OK] MinIO seeded" -ForegroundColor Green
        }
    }
}

# ============================================
# SUMMARY
# ============================================
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  RINCO Demo Seed Complete!" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""
Write-Host "Database Status:" -ForegroundColor White
Write-Host "  PostgreSQL  : $(([bool]$serviceStatus.postgres) ? 'OK' : 'SKIPPED')" -ForegroundColor $(if ($serviceStatus.postgres) { 'Green' } else { 'Yellow' })
Write-Host "  MongoDB     : $(([bool]$serviceStatus.mongo) ? 'OK' : 'SKIPPED')" -ForegroundColor $(if ($serviceStatus.mongo) { 'Green' } else { 'Yellow' })
Write-Host "  ScyllaDB    : $(([bool]$serviceStatus.scylla) ? 'OK' : 'SKIPPED')" -ForegroundColor $(if ($serviceStatus.scylla) { 'Green' } else { 'Yellow' })
Write-Host "  ClickHouse  : $(([bool]$serviceStatus.clickhouse) ? 'OK' : 'SKIPPED')" -ForegroundColor $(if ($serviceStatus.clickhouse) { 'Green' } else { 'Yellow' })
Write-Host "  Valkey      : $(([bool]$serviceStatus.valkey) ? 'OK' : 'SKIPPED')" -ForegroundColor $(if ($serviceStatus.valkey) { 'Green' } else { 'Yellow' })
Write-Host "  MinIO       : $(([bool]$serviceStatus.minio) ? 'OK' : 'SKIPPED')" -ForegroundColor $(if ($serviceStatus.minio) { 'Green' } else { 'Yellow' })
Write-Host ""

Write-Host "Test users available (password = 'rinco_dev_password'):" -ForegroundColor White
Write-Host "  - admin@rinco.app            (super admin, root tenant)" -ForegroundColor Gray
Write-Host "  - admin@apexfintech.vn       (Apex Fintech tenant admin)" -ForegroundColor Gray
Write-Host "  - admin@hct.vn               (HCT Consulting tenant admin)" -ForegroundColor Gray
Write-Host "  - demo@demo.com              (Demo Company tenant admin)" -ForegroundColor Gray
Write-Host ""
Write-Host "Total demo records: ~3000 across all databases" -ForegroundColor White
Write-Host ""
