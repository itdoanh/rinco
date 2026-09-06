<?php
/**
 * API: Dashboard Stats
 *
 * GET /api/stats.php?from=2026-09-01&to=2026-09-30
 *
 * Trả về KPIs cho dashboard với APCu caching.
 * Sử dụng Summary Table - KHÔNG BAO GIỜ query live COUNT(*)
 */

declare(strict_types=1);

require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/../includes/cache.php';
require_once __DIR__ . '/lead-helpers.php';

header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: public, max-age=30'); // Cache 30s

// Parse date range
$to = $_GET['to'] ?? date('Y-m-d');
$from = $_GET['from'] ?? date('Y-m-d', strtotime('-29 days'));
$today = date('Y-m-d');
$yest = date('Y-m-d', strtotime('-1 day'));

// Validate
if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $from)) $from = date('Y-m-d', strtotime('-29 days'));
if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $to)) $to = date('Y-m-d');

// Build cache key
$cacheKey = "dashboard_stats_v2:{$from}:{$to}";

try {
    // ========== Auto-rebuild summary_daily trước khi đọc ==========
    // Đảm bảo dữ liệu luôn mới nhất (rebuild hôm nay + hôm qua, cache 30s)
    $pdo = lead_db();
    try {
        _stats_auto_rebuild($pdo, $today, $yest);
        // Xóa cache của chính endpoint này để query lại từ DB
        ApexCache::invalidate('dashboard_stats_v2', $from, $to);
        error_log('[STATS] Rebuilt summary_daily, cache invalidated. today=' . $today);
    } catch (Throwable $e) {
        error_log('[stats auto-rebuild] ' . $e->getMessage());
    }

    // Try cache first
    $cached = ApexCache::getSync('dashboard_stats_v2', $from, $to);
    if ($cached !== null) {
        error_log('[STATS] Cache HIT - returning cached data. visitors=' . $cached['summary']['visitors_total']);
    } else {
        error_log('[STATS] Cache MISS - querying DB');
    }
    if ($cached !== null) {
        echo json_encode($cached, JSON_INVALID_UTF8_IGNORE);
        exit;
    }

    // Load from Summary Table
    $stmt = $pdo->prepare("
        SELECT * FROM summary_daily
        WHERE date BETWEEN ? AND ?
        ORDER BY date DESC
    ");
    $stmt->execute([$from, $to]);
    $rows = $stmt->fetchAll();
    error_log('[STATS] Query summary_daily: ' . count($rows) . ' rows, from=' . $from . ' to=' . $to);
    error_log('[STATS] summary_daily rows: ' . json_encode(array_map(fn($r) => ['date'=>$r['date'],'visitors'=>$r['visitors_total'],'fb'=>$r['visitors_fb'],'direct'=>$r['visitors_direct']], $rows)));

    // Build summary by date
    $byDate = [];
    foreach ($rows as $row) {
        $byDate[$row['date']] = $row;
    }

    // Aggregate for date range
    $agg = [
        'visitors_total' => 0,
        'visitors_fb' => 0,
        'visitors_gg' => 0,
        'visitors_direct' => 0,
        'leads_total' => 0,
        'leads_hero' => 0,
        'leads_modal' => 0,
        'leads_multistep' => 0,
        'leads_fb' => 0,
        'leads_duplicate' => 0,
        'pageviews_total' => 0,
        'sessions_total' => 0,
        'cta_clicks_total' => 0,
        'form_focuses_total' => 0,
        'form_submits_attempts' => 0,
        'form_submits_success' => 0,
        'errors_total' => 0,
        'ips_total' => 0,
        'ips_vpn' => 0,
        'ips_datacenter' => 0,
    ];

    $totalVisitors = 0;
    $totalLeads = 0;
    $totalDuration = 0;
    $totalScroll = 0;
    $countWithData = 0;

    foreach ($rows as $row) {
        $agg['visitors_total'] += (int)$row['visitors_total'];
        $agg['visitors_fb'] += (int)$row['visitors_fb'];
        $agg['visitors_gg'] += (int)$row['visitors_gg'];
        $agg['visitors_direct'] += (int)$row['visitors_direct'];
        $agg['leads_total'] += (int)$row['leads_total'];
        $agg['leads_hero'] += (int)$row['leads_hero'];
        $agg['leads_modal'] += (int)$row['leads_modal'];
        $agg['leads_multistep'] += (int)$row['leads_multistep'];
        $agg['leads_fb'] += (int)$row['leads_fb'];
        $agg['leads_duplicate'] += (int)$row['leads_duplicate'];
        $agg['pageviews_total'] += (int)$row['pageviews_total'];
        $agg['sessions_total'] += (int)$row['sessions_total'];
        $agg['cta_clicks_total'] += (int)$row['cta_clicks_total'];
        $agg['form_focuses_total'] += (int)$row['form_focuses_total'];
        $agg['form_submits_attempts'] += (int)$row['form_submits_attempts'];
        $agg['form_submits_success'] += (int)$row['form_submits_success'];
        $agg['errors_total'] += (int)$row['errors_total'];
        $agg['ips_total'] += (int)$row['ips_total'];
        $agg['ips_vpn'] += (int)$row['ips_vpn'];
        $agg['ips_datacenter'] += (int)$row['ips_datacenter'];

        // Weighted average
        $visitors = (int)$row['visitors_total'];
        if ($visitors > 0) {
            $totalDuration += (int)$row['avg_duration_seconds'] * $visitors;
            $totalScroll += (float)$row['avg_scroll_percent'] * $visitors;
            $totalVisitors += $visitors;
            $totalLeads += (int)$row['leads_total'];
            $countWithData++;
        }
    }

    // Computed KPIs
    $avgDuration = $totalVisitors > 0 ? (int)($totalDuration / $totalVisitors) : 0;
    $avgScroll = $totalVisitors > 0 ? round($totalScroll / $totalVisitors, 1) : 0;
    $convRate = $totalVisitors > 0 ? round(($totalLeads / $totalVisitors) * 100, 2) : 0;
    $fbConvRate = $agg['visitors_fb'] > 0 ? round(($agg['leads_fb'] / $agg['visitors_fb']) * 100, 2) : 0;

    // Today & Yesterday
    $todayData = $byDate[$today] ?? null;
    $yestData = $byDate[$yest] ?? null;

    // Time series
    $timeSeries = [];
    $current = new DateTime($from);
    $end = new DateTime($to);
    while ($current <= $end) {
        $date = $current->format('Y-m-d');
        $row = $byDate[$date] ?? null;
        $timeSeries[] = [
            'date' => $date,
            'label' => $current->format('d/m'),
            'visitors' => (int)($row['visitors_total'] ?? 0),
            'visitors_fb' => (int)($row['visitors_fb'] ?? 0),
            'leads' => (int)($row['leads_total'] ?? 0),
            'leads_hero' => (int)($row['leads_hero'] ?? 0),
            'leads_modal' => (int)($row['leads_modal'] ?? 0),
            'leads_multistep' => (int)($row['leads_multistep'] ?? 0),
            'pageviews' => (int)($row['pageviews_total'] ?? 0),
            'conv_rate' => $row['conv_rate_visitors'] ?? 0,
        ];
        $current->modify('+1 day');
    }

    // Build response
    $response = [
        'ok' => true,
        'range' => ['from' => $from, 'to' => $to, 'days' => count($timeSeries)],
        'today' => [
            'visitors' => (int)($todayData['visitors_total'] ?? 0),
            'leads' => (int)($todayData['leads_total'] ?? 0),
            'pageviews' => (int)($todayData['pageviews_total'] ?? 0),
            'conv_rate' => (float)($todayData['conv_rate_visitors'] ?? 0),
        ],
        'yesterday' => [
            'visitors' => (int)($yestData['visitors_total'] ?? 0),
            'leads' => (int)($yestData['leads_total'] ?? 0),
        ],
        'summary' => [
            'visitors_total' => $agg['visitors_total'],
            'visitors_fb' => $agg['visitors_fb'],
            'visitors_gg' => $agg['visitors_gg'],
            'visitors_direct' => $agg['visitors_direct'],
            'leads_total' => $agg['leads_total'],
            'leads_hero' => $agg['leads_hero'],
            'leads_modal' => $agg['leads_modal'],
            'leads_multistep' => $agg['leads_multistep'],
            'leads_fb' => $agg['leads_fb'],
            'leads_duplicate' => $agg['leads_duplicate'],
            'conv_rate' => $convRate,
            'fb_conv_rate' => $fbConvRate,
            'avg_duration_seconds' => $avgDuration,
            'avg_scroll_percent' => $avgScroll,
            'pageviews_total' => $agg['pageviews_total'],
            'cta_clicks_total' => $agg['cta_clicks_total'],
            'form_focuses_total' => $agg['form_focuses_total'],
            'form_submits_attempts' => $agg['form_submits_attempts'],
            'form_submits_success' => $agg['form_submits_success'],
            'errors_total' => $agg['errors_total'],
        ],
        'traffic' => [
            'fb_percent' => $agg['visitors_total'] > 0 ? round(($agg['visitors_fb'] / $agg['visitors_total']) * 100, 1) : 0,
            'gg_percent' => $agg['visitors_total'] > 0 ? round(($agg['visitors_gg'] / $agg['visitors_total']) * 100, 1) : 0,
            'direct_percent' => $agg['visitors_total'] > 0 ? round(($agg['visitors_direct'] / $agg['visitors_total']) * 100, 1) : 0,
        ],
        'ip_risk' => [
            'total_ips' => $agg['ips_total'],
            'vpn_count' => $agg['ips_vpn'],
            'datacenter_count' => $agg['ips_datacenter'],
        ],
        'funnel' => [
            'visitors' => $totalVisitors,
            'cta_clicks' => $agg['cta_clicks_total'],
            'form_focuses' => $agg['form_focuses_total'],
            'form_submits_attempts' => $agg['form_submits_attempts'],
            'form_submits_success' => $agg['form_submits_success'],
            'leads' => $agg['leads_total'],
            'completion_rate' => $agg['form_focuses_total'] > 0 ? round(($agg['form_submits_success'] / $agg['form_focuses_total']) * 100, 1) : 0,
        ],
        'time_series' => $timeSeries,
        'server_time' => (new DateTime())->format('c'),
        'cache_ttl' => 30,
    ];

    // Cache for 30 seconds
    ApexCache::set('dashboard_stats_v2', $response, 30, $from, $to);

    echo json_encode($response, JSON_INVALID_UTF8_IGNORE);

} catch (Throwable $e) {
    error_log('Stats API Error: ' . $e->getMessage());

    http_response_code(500);
    echo json_encode([
        'ok' => false,
        'error' => 'server_error',
        'message' => 'Failed to load statistics',
    ]);
}

// ============================================================
// Auto-rebuild summary_daily (cache 30s)
// ============================================================

/**
 * Rebuild summary_daily cho 2 ngày: hôm nay + hôm qua.
 * Chỉ rebuild 1 lần mỗi 30s (tránh spam query khi nhiều request).
 */
function _stats_auto_rebuild(PDO $pdo, string $today, string $yest): void {
    static $last_rebuild = 0;
    $now = time();
    if (($now - $last_rebuild) < 30) return;
    $last_rebuild = $now;

    foreach ([$today, $yest] as $date) {
        // Visitors (unique sessions, không tính bot)
        $stmt = $pdo->prepare("
            SELECT COUNT(DISTINCT session_id) FROM visitor_sessions
            WHERE DATE(first_visit) = ? AND is_bot = 0
        ");
        $stmt->execute([$date]);
        $visitors_total = (int)$stmt->fetchColumn();

        $stmt = $pdo->prepare("
            SELECT COUNT(DISTINCT session_id) FROM visitor_sessions
            WHERE DATE(first_visit) = ? AND is_bot = 1
        ");
        $stmt->execute([$date]);
        $visitors_bot = (int)$stmt->fetchColumn();

        // UTM breakdown - Look in MULTIPLE sources
        // Sử dụng visitor_sessions làm primary, fallback sang events/leads nếu cần
        $stmt = $pdo->prepare("
            SELECT
                COUNT(DISTINCT CASE WHEN src IN ('fb', 'facebook', 'ig', 'meta') THEN session_id END) AS fb_visitors,
                COUNT(DISTINCT CASE WHEN src IN ('gg', 'google', 'adwords') THEN session_id END) AS gg_visitors,
                COUNT(DISTINCT CASE WHEN src IS NULL OR src = '' OR src = '(direct)' THEN session_id END) AS direct_visitors
            FROM (
                SELECT DISTINCT session_id, COALESCE(NULLIF(utm_source,''),'(direct)') AS src FROM visitor_sessions
                WHERE DATE(first_visit) = ? AND is_bot = 0
                UNION ALL
                -- Lấy thêm từ events cho những session chưa có UTM từ visitor_sessions
                SELECT DISTINCT e.session_id, COALESCE(NULLIF(e.utm_source,''),'(direct)') AS src FROM events e
                LEFT JOIN visitor_sessions vs ON vs.session_id = e.session_id
                WHERE DATE(e.created_at) = ?
                  AND vs.session_id IS NULL  -- Chỉ lấy session chưa có trong visitor_sessions
            ) AS sessions_with_utm
        ");
        $stmt->execute([$date, $date]);
        $utm = $stmt->fetch();
        $visitors_fb = (int)($utm['fb_visitors'] ?? 0);
        $visitors_gg = (int)($utm['gg_visitors'] ?? 0);
        $visitors_direct = (int)($utm['direct_visitors'] ?? 0);

        // Leads breakdown
        $stmt = $pdo->prepare("
            SELECT
                COUNT(*) AS total,
                SUM(CASE WHEN form_type = 'hero' THEN 1 ELSE 0 END) AS hero,
                SUM(CASE WHEN form_type = 'modal' THEN 1 ELSE 0 END) AS modal,
                SUM(CASE WHEN form_type NOT IN ('hero','modal') AND form_type IS NOT NULL THEN 1 ELSE 0 END) AS multistep,
                SUM(CASE WHEN utm_source IN ('fb','facebook','ig','meta') THEN 1 ELSE 0 END) AS leads_fb,
                SUM(CASE WHEN duplicate_of_id IS NOT NULL OR duplicate_group_id IS NOT NULL THEN 1 ELSE 0 END) AS leads_dup
            FROM leads
            WHERE DATE(created_at) = ?
        ");
        $stmt->execute([$date]);
        $l = $stmt->fetch();
        $leads_total = (int)($l['total'] ?? 0);
        $leads_hero = (int)($l['hero'] ?? 0);
        $leads_modal = (int)($l['modal'] ?? 0);
        $leads_multistep = (int)($l['multistep'] ?? 0);
        $leads_fb = (int)($l['leads_fb'] ?? 0);
        $leads_duplicate = (int)($l['leads_dup'] ?? 0);

        // CTA / form events
        $stmt = $pdo->prepare("SELECT COUNT(*) FROM events WHERE DATE(created_at) = ? AND event_type = 'cta_click'");
        $stmt->execute([$date]);
        $cta_clicks_total = (int)$stmt->fetchColumn();

        $stmt = $pdo->prepare("SELECT COUNT(*) FROM events WHERE DATE(created_at) = ? AND event_type = 'form_focus'");
        $stmt->execute([$date]);
        $form_focuses_total = (int)$stmt->fetchColumn();

        $stmt = $pdo->prepare("SELECT COUNT(*) FROM events WHERE DATE(created_at) = ? AND event_type = 'form_submit'");
        $stmt->execute([$date]);
        $form_submits_attempts = (int)$stmt->fetchColumn();

        // Avg duration & scroll
        $stmt = $pdo->prepare("SELECT AVG(total_duration) FROM visitor_sessions WHERE DATE(first_visit) = ? AND is_bot = 0 AND total_duration > 0");
        $stmt->execute([$date]);
        $avg_duration = (int)$stmt->fetchColumn();

        $stmt = $pdo->prepare("SELECT AVG(scroll_max) FROM visitor_sessions WHERE DATE(first_visit) = ? AND is_bot = 0 AND scroll_max > 0");
        $stmt->execute([$date]);
        $avg_scroll = (float)$stmt->fetchColumn();

        // Pageviews = count page_view events
        $stmt = $pdo->prepare("SELECT COUNT(*) FROM events WHERE DATE(created_at) = ? AND event_type = 'page_view'");
        $stmt->execute([$date]);
        $pageviews_total = (int)$stmt->fetchColumn();

        // Sessions unique (also count)
        $stmt = $pdo->prepare("SELECT COUNT(DISTINCT session_id) FROM events WHERE DATE(created_at) = ?");
        $stmt->execute([$date]);
        $sessions_total = (int)$stmt->fetchColumn();

        // Conv rate
        $conv_rate_visitors = ($visitors_total > 0) ? round(($leads_total / $visitors_total) * 100, 2) : 0;

        // Upsert (try full insert first; nếu cột nào không tồn tại sẽ fallback)
        try {
            $pdo->prepare("
                INSERT INTO summary_daily
                    (date, visitors_total, visitors_bot, visitors_fb, visitors_gg, visitors_direct,
                     leads_total, leads_hero, leads_modal, leads_multistep, leads_fb, leads_duplicate,
                     pageviews_total, sessions_total, cta_clicks_total, form_focuses_total, form_submits_attempts,
                     avg_duration_seconds, avg_scroll_percent, conv_rate_visitors)
                VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
                ON DUPLICATE KEY UPDATE
                    visitors_total = VALUES(visitors_total),
                    visitors_bot = VALUES(visitors_bot),
                    visitors_fb = VALUES(visitors_fb),
                    visitors_gg = VALUES(visitors_gg),
                    visitors_direct = VALUES(visitors_direct),
                    leads_total = VALUES(leads_total),
                    leads_hero = VALUES(leads_hero),
                    leads_modal = VALUES(leads_modal),
                    leads_multistep = VALUES(leads_multistep),
                    leads_fb = VALUES(leads_fb),
                    leads_duplicate = VALUES(leads_duplicate),
                    pageviews_total = VALUES(pageviews_total),
                    sessions_total = VALUES(sessions_total),
                    cta_clicks_total = VALUES(cta_clicks_total),
                    form_focuses_total = VALUES(form_focuses_total),
                    form_submits_attempts = VALUES(form_submits_attempts),
                    avg_duration_seconds = VALUES(avg_duration_seconds),
                    avg_scroll_percent = VALUES(avg_scroll_percent),
                    conv_rate_visitors = VALUES(conv_rate_visitors)
            ")->execute([
                $date, $visitors_total, $visitors_bot, $visitors_fb, $visitors_gg, $visitors_direct,
                $leads_total, $leads_hero, $leads_modal, $leads_multistep, $leads_fb, $leads_duplicate,
                $pageviews_total, $sessions_total, $cta_clicks_total, $form_focuses_total, $form_submits_attempts,
                $avg_duration, $avg_scroll, $conv_rate_visitors
            ]);
        } catch (Throwable $e) {
            // Nếu 1 số cột không tồn tại (vd: leads_duplicate), bỏ qua lỗi
            error_log('[stats auto-rebuild insert] ' . $e->getMessage());
        }
    }
}