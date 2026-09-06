<?php
/**
 * CRON: Build Summary Daily Table
 *
 * Chạy mỗi 5 phút bằng crontab:
 * */5 * * * * php /home/hanghoap/public_html/chiase/cron/build_summary_daily.php >> /home/hanghoap/logs/summary.log 2>&1
 *
 * Tính toán pre-computed KPIs cho dashboard.
 * KẾT QUẢ: Dashboard load < 50ms thay vì 800ms-2s
 */

declare(strict_types=1);

require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/../includes/cache.php';
require_once __DIR__ . '/../includes/db-helpers.php';

echo "[" . date('Y-m-d H:i:s') . "] Starting summary build...\n";

$pdo = ApexDB::get();

// Ngày cần tính: hôm nay và hôm qua
$dates = [
    date('Y-m-d'),                          // Today
    date('Y-m-d', strtotime('-1 day')),    // Yesterday
];

foreach ($dates as $date) {
    $from = $date . ' 00:00:00';
    $to   = $date . ' 23:59:59';

    echo "Building summary for {$date}...\n";

    try {
        buildSummaryForDate($pdo, $date, $from, $to);
        echo "  [OK] {$date}\n";
    } catch (Throwable $e) {
        echo "  [ERROR] {$date}: " . $e->getMessage() . "\n";
        error_log("Summary build error for {$date}: " . $e->getMessage());
    }
}

// Invalidate cache
ApexCache::invalidateNamespace('summary_daily');
ApexCache::invalidateNamespace('dashboard_stats_v2');
ApexCache::invalidateNamespace('timeseries');
ApexCache::invalidateNamespace('traffic_sources');
ApexCache::invalidateNamespace('campaign_breakdown');
ApexCache::invalidateNamespace('funnel_data');

echo "[" . date('Y-m-d H:i:s') . "] Summary build complete.\n";

/**
 * Build summary cho 1 ngày
 */
function buildSummaryForDate(PDO $pdo, string $date, string $from, string $to): void
{
    // ===== VISITORS =====
    $visitorsSql = "
        SELECT
            COUNT(DISTINCT session_id) AS visitors_total,
            SUM(is_bot) AS visitors_bot,
            SUM(CASE WHEN utm_source LIKE '%facebook%' OR utm_source = 'fb' OR fbclid IS NOT NULL THEN 1 ELSE 0 END) AS visitors_fb,
            SUM(CASE WHEN utm_source LIKE '%google%' OR utm_source IN ('gg','google') OR gclid IS NOT NULL THEN 1 ELSE 0 END) AS visitors_gg,
            SUM(CASE WHEN (utm_source IS NULL OR utm_source = '') AND (referrer IS NULL OR referrer = '') THEN 1 ELSE 0 END) AS visitors_direct,
            SUM(CASE WHEN utm_source IS NULL AND referrer != '' AND referrer NOT LIKE '%facebook%' AND referrer NOT LIKE '%google%' THEN 1 ELSE 0 END) AS visitors_organic
        FROM visitor_sessions
        WHERE first_visit BETWEEN :from AND :to
    ";
    $stmt = $pdo->prepare($visitorsSql);
    $stmt->execute(['from' => $from, 'to' => $to]);
    $visitors = $stmt->fetch(PDO::FETCH_ASSOC);

    // ===== PAGEVIEWS & ENGAGEMENT =====
    $pageviewsSql = "
        SELECT
            SUM(page_views) AS pageviews_total,
            COUNT(DISTINCT session_id) AS sessions_total,
            AVG(total_duration) AS avg_duration_seconds,
            AVG(scroll_max) AS avg_scroll_percent
        FROM visitor_sessions
        WHERE first_visit BETWEEN :from AND :to
          AND is_bot = 0
    ";
    $stmt = $pdo->prepare($pageviewsSql);
    $stmt->execute(['from' => $from, 'to' => $to]);
    $engagement = $stmt->fetch(PDO::FETCH_ASSOC);

    // ===== LEADS =====
    $leadsSql = "
        SELECT
            COUNT(*) AS leads_total,
            SUM(CASE WHEN form_type = 'hero' THEN 1 ELSE 0 END) AS leads_hero,
            SUM(CASE WHEN form_type = 'modal' THEN 1 ELSE 0 END) AS leads_modal,
            SUM(CASE WHEN form_type = 'multistep' THEN 1 ELSE 0 END) AS leads_multistep,
            SUM(CASE WHEN utm_source LIKE '%facebook%' OR utm_source = 'fb' OR fbclid IS NOT NULL THEN 1 ELSE 0 END) AS leads_fb,
            SUM(CASE WHEN utm_source LIKE '%google%' OR gclid IS NOT NULL THEN 1 ELSE 0 END) AS leads_gg,
            SUM(CASE WHEN utm_source IS NULL AND fbclid IS NULL AND gclid IS NULL THEN 1 ELSE 0 END) AS leads_organic,
            SUM(CASE WHEN status = 'duplicate' THEN 1 ELSE 0 END) AS leads_duplicate,
            SUM(CASE WHEN status = 'invalid' THEN 1 ELSE 0 END) AS leads_invalid
        FROM leads
        WHERE created_at BETWEEN :from AND :to
    ";
    $stmt = $pdo->prepare($leadsSql);
    $stmt->execute(['from' => $from, 'to' => $to]);
    $leads = $stmt->fetch(PDO::FETCH_ASSOC);

    // ===== FORM FUNNEL (từ events) =====
    $funnelSql = "
        SELECT
            SUM(CASE WHEN event_type = 'cta_click' THEN 1 ELSE 0 END) AS cta_clicks_total,
            SUM(CASE WHEN event_type = 'form_focus' THEN 1 ELSE 0 END) AS form_focuses_total,
            SUM(CASE WHEN event_type = 'form_submit_attempt' THEN 1 ELSE 0 END) AS form_submits_attempts,
            SUM(CASE WHEN event_type = 'form_submit_success' THEN 1 ELSE 0 END) AS form_submits_success,
            SUM(CASE WHEN event_type = 'form_idle_abandon' OR event_type = 'form_abandon' THEN 1 ELSE 0 END) AS form_abandons
        FROM events
        WHERE created_at BETWEEN :from AND :to
    ";
    $stmt = $pdo->prepare($funnelSql);
    $stmt->execute(['from' => $from, 'to' => $to]);
    $funnel = $stmt->fetch(PDO::FETCH_ASSOC);

    // ===== ERRORS =====
    $errorsSql = "
        SELECT
            COUNT(*) AS errors_total,
            SUM(CASE WHEN error_stage = 'network' THEN 1 ELSE 0 END) AS errors_network,
            SUM(CASE WHEN error_stage = 'validation' THEN 1 ELSE 0 END) AS errors_validation,
            SUM(CASE WHEN error_stage = 'db' THEN 1 ELSE 0 END) AS errors_db
        FROM lead_errors
        WHERE created_at BETWEEN :from AND :to
    ";
    $stmt = $pdo->prepare($errorsSql);
    $stmt->execute(['from' => $from, 'to' => $to]);
    $errors = $stmt->fetch(PDO::FETCH_ASSOC);

    // ===== IP RISK =====
    $ipSql = "
        SELECT
            COUNT(DISTINCT ip_hash) AS ips_total,
            SUM(is_vpn) AS ips_vpn,
            SUM(is_datacenter) AS ips_datacenter
        FROM ip_tracking
        WHERE last_seen BETWEEN :from AND :to
    ";
    $stmt = $pdo->prepare($ipSql);
    $stmt->execute(['from' => $from, 'to' => $to]);
    $ipRisk = $stmt->fetch(PDO::FETCH_ASSOC);

    // ===== COMPUTE KPIs =====
    $visitorsTotal = (int)($visitors['visitors_total'] ?? 0);
    $visitorsReal = $visitorsTotal - (int)($visitors['visitors_bot'] ?? 0);
    $leadsTotal = (int)($leads['leads_total'] ?? 0);
    $formFocuses = (int)($funnel['form_focuses_total'] ?? 0);

    $convRate = $visitorsTotal > 0 ? round($leadsTotal / $visitorsTotal, 4) : 0;
    $fbVisitors = (int)($visitors['visitors_fb'] ?? 0);
    $leadsFb = (int)($leads['leads_fb'] ?? 0);
    $fbConvRate = $fbVisitors > 0 ? round($leadsFb / $fbVisitors, 4) : 0;
    $formCompletionRate = $formFocuses > 0 ? round((int)($funnel['form_submits_success'] ?? 0) / $formFocuses, 4) : 0;

    // ===== UPSERT SUMMARY =====
    $upsertSql = "
        INSERT INTO summary_daily (
            date,
            visitors_total, visitors_bot, visitors_fb, visitors_gg, visitors_direct, visitors_organic,
            pageviews_total, sessions_total, avg_duration_seconds, avg_scroll_percent,
            leads_total, leads_hero, leads_modal, leads_multistep,
            leads_fb, leads_gg, leads_organic, leads_duplicate, leads_invalid,
            cta_clicks_total, form_focuses_total, form_submits_attempts, form_submits_success, form_abandons,
            errors_total, errors_network, errors_validation, errors_db,
            ips_total, ips_vpn, ips_datacenter,
            conv_rate_visitors, conv_rate_fb, form_completion_rate,
            updated_at
        ) VALUES (
            :date,
            :visitors_total, :visitors_bot, :visitors_fb, :visitors_gg, :visitors_direct, :visitors_organic,
            :pageviews_total, :sessions_total, :avg_duration_seconds, :avg_scroll_percent,
            :leads_total, :leads_hero, :leads_modal, :leads_multistep,
            :leads_fb, :leads_gg, :leads_organic, :leads_duplicate, :leads_invalid,
            :cta_clicks_total, :form_focuses_total, :form_submits_attempts, :form_submits_success, :form_abandons,
            :errors_total, :errors_network, :errors_validation, :errors_db,
            :ips_total, :ips_vpn, :ips_datacenter,
            :conv_rate_visitors, :conv_rate_fb, :form_completion_rate,
            NOW()
        )
        ON DUPLICATE KEY UPDATE
            visitors_total = VALUES(visitors_total),
            visitors_bot = VALUES(visitors_bot),
            visitors_fb = VALUES(visitors_fb),
            visitors_gg = VALUES(visitors_gg),
            visitors_direct = VALUES(visitors_direct),
            visitors_organic = VALUES(visitors_organic),
            pageviews_total = VALUES(pageviews_total),
            sessions_total = VALUES(sessions_total),
            avg_duration_seconds = VALUES(avg_duration_seconds),
            avg_scroll_percent = VALUES(avg_scroll_percent),
            leads_total = VALUES(leads_total),
            leads_hero = VALUES(leads_hero),
            leads_modal = VALUES(leads_modal),
            leads_multistep = VALUES(leads_multistep),
            leads_fb = VALUES(leads_fb),
            leads_gg = VALUES(leads_gg),
            leads_organic = VALUES(leads_organic),
            leads_duplicate = VALUES(leads_duplicate),
            leads_invalid = VALUES(leads_invalid),
            cta_clicks_total = VALUES(cta_clicks_total),
            form_focuses_total = VALUES(form_focuses_total),
            form_submits_attempts = VALUES(form_submits_attempts),
            form_submits_success = VALUES(form_submits_success),
            form_abandons = VALUES(form_abandons),
            errors_total = VALUES(errors_total),
            errors_network = VALUES(errors_network),
            errors_validation = VALUES(errors_validation),
            errors_db = VALUES(errors_db),
            ips_total = VALUES(ips_total),
            ips_vpn = VALUES(ips_vpn),
            ips_datacenter = VALUES(ips_datacenter),
            conv_rate_visitors = VALUES(conv_rate_visitors),
            conv_rate_fb = VALUES(conv_rate_fb),
            form_completion_rate = VALUES(form_completion_rate),
            updated_at = NOW()
    ";

    $stmt = $pdo->prepare($upsertSql);
    $stmt->execute([
        'date' => $date,
        'visitors_total' => $visitorsTotal,
        'visitors_bot' => (int)($visitors['visitors_bot'] ?? 0),
        'visitors_fb' => $fbVisitors,
        'visitors_gg' => (int)($visitors['visitors_gg'] ?? 0),
        'visitors_direct' => (int)($visitors['visitors_direct'] ?? 0),
        'visitors_organic' => (int)($visitors['visitors_organic'] ?? 0),
        'pageviews_total' => (int)($engagement['pageviews_total'] ?? 0),
        'sessions_total' => (int)($engagement['sessions_total'] ?? 0),
        'avg_duration_seconds' => (int)round($engagement['avg_duration_seconds'] ?? 0),
        'avg_scroll_percent' => round((float)($engagement['avg_scroll_percent'] ?? 0), 2),
        'leads_total' => $leadsTotal,
        'leads_hero' => (int)($leads['leads_hero'] ?? 0),
        'leads_modal' => (int)($leads['leads_modal'] ?? 0),
        'leads_multistep' => (int)($leads['leads_multistep'] ?? 0),
        'leads_fb' => $leadsFb,
        'leads_gg' => (int)($leads['leads_gg'] ?? 0),
        'leads_organic' => (int)($leads['leads_organic'] ?? 0),
        'leads_duplicate' => (int)($leads['leads_duplicate'] ?? 0),
        'leads_invalid' => (int)($leads['leads_invalid'] ?? 0),
        'cta_clicks_total' => (int)($funnel['cta_clicks_total'] ?? 0),
        'form_focuses_total' => $formFocuses,
        'form_submits_attempts' => (int)($funnel['form_submits_attempts'] ?? 0),
        'form_submits_success' => (int)($funnel['form_submits_success'] ?? 0),
        'form_abandons' => (int)($funnel['form_abandons'] ?? 0),
        'errors_total' => (int)($errors['errors_total'] ?? 0),
        'errors_network' => (int)($errors['errors_network'] ?? 0),
        'errors_validation' => (int)($errors['errors_validation'] ?? 0),
        'errors_db' => (int)($errors['errors_db'] ?? 0),
        'ips_total' => (int)($ipRisk['ips_total'] ?? 0),
        'ips_vpn' => (int)($ipRisk['ips_vpn'] ?? 0),
        'ips_datacenter' => (int)($ipRisk['ips_datacenter'] ?? 0),
        'conv_rate_visitors' => $convRate,
        'conv_rate_fb' => $fbConvRate,
        'form_completion_rate' => $formCompletionRate,
    ]);
}
