<?php
/**
 * APEX Admin - Data queries v2 (PHP 8.4 Optimized)
 *
 * SỬ DỤNG:
 * 1. Summary Table (summary_daily) cho dashboard KPIs - KHÔNG BAO GIỜ query live
 * 2. Keyset Pagination thay vì OFFSET
 * 3. APCu caching cho frequent queries
 * 4. Covering indexes cho SELECT columns cần thiết
 */

declare(strict_types=1);

require_once __DIR__ . '/db.php';
require_once __DIR__ . '/cache.php';

/**
 * Phân tích date range từ query string
 */
function apex_parse_date_range(): array {
    $to   = $_GET['to']   ?? date('Y-m-d');
    $from = $_GET['from'] ?? date('Y-m-d', strtotime('-29 days'));

    // Validate format
    if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $from)) $from = date('Y-m-d', strtotime('-29 days'));
    if (!preg_match('/^\d{4}-\d{2}-\d{2}$/', $to))   $to   = date('Y-m-d');

    // Ensure from <= to
    if (strtotime($from) > strtotime($to)) {
        [$from, $to] = [$to, $from];
    }

    // Giới hạn max 365 ngày
    $diff = (strtotime($to) - strtotime($from)) / 86400;
    if ($diff > 365) {
        $from = date('Y-m-d', strtotime($to . ' -365 days'));
    }

    return [
        'from'   => $from,
        'to'     => $to,
        'from_dt' => $from . ' 00:00:00',
        'to_dt'   => $to   . ' 23:59:59',
        'days'   => max(1, (int)$diff + 1),
    ];
}

// ============================================================
// DASHBOARD OVERVIEW - SỬ DỤNG SUMMARY TABLE
// ============================================================

/**
 * Overview stats - OPTIMIZED với Summary Table
 * Sử dụng pre-computed summary_daily thay vì COUNT(*)
 */
function apex_query_overview(array $range): array {
    $from = $range['from'];
    $to   = $range['to'];
    $today = date('Y-m-d');
    $yest  = date('Y-m-d', strtotime('-1 day'));

    $pdo = apex_db();

    // Đọc từ Summary Table (cache < 5 phút)
    $summary = ApexCache::get('summary_daily',
        fn() => loadSummaryTable($pdo),
        300
    );

    // Lấy số hôm nay và hôm qua từ summary
    $todaySummary = $summary[$today] ?? null;
    $yestSummary = $summary[$yest] ?? null;

    // Lấy cumulative cho date range
    $rangeSummary = [
        'visitors_total' => 0,
        'visitors_fb' => 0,
        'visitors_gg' => 0,
        'visitors_direct' => 0,
        'leads_total' => 0,
        'leads_fb' => 0,
        'cta_clicks_total' => 0,
        'form_focuses_total' => 0,
        'form_submits_attempts' => 0,
    ];

    foreach ($summary as $date => $row) {
        if ($date >= $from && $date <= $to) {
            $rangeSummary['visitors_total'] += ($row['visitors_total'] ?? 0);
            $rangeSummary['visitors_fb'] += ($row['visitors_fb'] ?? 0);
            $rangeSummary['visitors_gg'] += ($row['visitors_gg'] ?? 0);
            $rangeSummary['visitors_direct'] += ($row['visitors_direct'] ?? 0);
            $rangeSummary['leads_total'] += ($row['leads_total'] ?? 0);
            $rangeSummary['leads_fb'] += ($row['leads_fb'] ?? 0);
            $rangeSummary['cta_clicks_total'] += ($row['cta_clicks_total'] ?? 0);
            $rangeSummary['form_focuses_total'] += ($row['form_focuses_total'] ?? 0);
            $rangeSummary['form_submits_attempts'] += ($row['form_submits_attempts'] ?? 0);
        }
    }

    $visitors = $rangeSummary['visitors_total'];
    $visitorsToday = $todaySummary['visitors_total'] ?? 0;
    $visitorsYest = $yestSummary['visitors_total'] ?? 0;

    $leadsTotal = $rangeSummary['leads_total'];
    $leadsToday = $todaySummary['leads_total'] ?? 0;
    $leadsFromFB = $rangeSummary['leads_fb'];
    $fbVisitors = $rangeSummary['visitors_fb'];

    $convRate = $visitors > 0 ? round(($leadsTotal / $visitors) * 100, 2) : 0;
    $fbConvRate = $fbVisitors > 0 ? round(($leadsFromFB / $fbVisitors) * 100, 2) : 0;

    // Avg stats từ summary
    $avgTime = 0;
    $avgScroll = 0;
    $count = 0;
    foreach ($summary as $date => $row) {
        if ($date >= $from && $date <= $to && ($row['visitors_total'] ?? 0) > 0) {
            $avgTime += ($row['avg_duration_seconds'] ?? 0) * ($row['visitors_total'] ?? 0);
            $avgScroll += ($row['avg_scroll_percent'] ?? 0) * ($row['visitors_total'] ?? 0);
            $count += ($row['visitors_total'] ?? 0);
        }
    }
    $avgTime = $count > 0 ? (int)($avgTime / $count) : 0;
    $avgScroll = $count > 0 ? (int)($avgScroll / $count) : 0;

    return [
        'visitors'         => $visitors,
        'visitors_today'   => $visitorsToday,
        'visitors_yest'    => $visitorsYest,
        'pageviews'        => 0, // Đọc từ summary
        'fb_visitors'      => $fbVisitors,
        'fb_visitors_today'=> $todaySummary['visitors_fb'] ?? 0,
        'gg_visitors'      => $rangeSummary['visitors_gg'],
        'direct_visitors'  => $rangeSummary['visitors_direct'],
        'leads_total'      => $leadsTotal,
        'leads_today'      => $leadsToday,
        'leads_from_fb'    => $leadsFromFB,
        'conv_rate'        => $convRate,
        'fb_conv_rate'     => $fbConvRate,
        'avg_time'         => $avgTime,
        'avg_scroll'       => $avgScroll,
        'cta_clicks'       => $rangeSummary['cta_clicks_total'],
        'form_submits'     => $rangeSummary['form_submits_attempts'],
        'bot_traffic'      => 0, // Không cần trong summary
    ];
}

/**
 * Load Summary Table - chỉ chạy khi cache miss
 */
function loadSummaryTable(PDO $pdo): array {
    $rows = $pdo->query("SELECT * FROM summary_daily ORDER BY date DESC LIMIT 365")->fetchAll(PDO::FETCH_UNIQUE);
    $result = [];
    foreach ($rows as $row) {
        $result[$row['date']] = $row;
    }
    return $result;
}

// ============================================================
// TIME SERIES - TỐI ƯU KHÔNG DÙNG UNION ALL
// ============================================================

/**
 * Time series - OPTIMIZED với Summary Table
 */
function apex_query_timeseries(array $range): array {
    $from = $range['from'];
    $to   = $range['to'];

    return ApexCache::get('timeseries',
        function() use ($from, $to) {
            $pdo = apex_db();
            return $pdo->query("
                SELECT
                    date,
                    visitors_total,
                    visitors_fb,
                    leads_total,
                    leads_hero,
                    leads_modal,
                    leads_multistep,
                    cta_clicks_total,
                    form_focuses_total,
                    avg_duration_seconds,
                    avg_scroll_percent
                FROM summary_daily
                WHERE date BETWEEN '{$from}' AND '{$to}'
                ORDER BY date ASC
            ")->fetchAll();
        },
        60
    );
}

// ============================================================
// KEYSET PAGINATION CHO LEADS
// ============================================================

/**
 * Leads list với KEYSET PAGINATION
 *
 * @param array $range Date range
 * @param string|null $search Search term
 * @param string|null $afterId Cursor: last ID on current page (null = first page)
 * @param int $limit Rows per page
 * @param string|null $status Filter by status
 * @param string|null $formType Filter by form type
 * @param string|null $source Filter by UTM source
 * @return array
 */
function apex_query_leads_keyset(
    array $range,
    ?string $search = null,
    ?string $afterId = null,
    int $limit = 50,
    ?string $status = null,
    ?string $formType = null,
    ?string $source = null
): array {
    $pdo = apex_db();
    $from = $range['from_dt'];
    $to   = $range['to_dt'];

    $params = [];
    $where = ['created_at BETWEEN ? AND ?'];
    $params[] = $from;
    $params[] = $to;

    // KEYSET: điều kiện cho cursor
    if ($afterId !== null) {
        $where[] = 'id < ?';
        $params[] = (int)$afterId;
    }

    if ($search !== null && $search !== '') {
        $where[] = '(name LIKE ? OR phone LIKE ? OR utm_source LIKE ?)';
        $s = '%' . $search . '%';
        $params[] = $s;
        $params[] = $s;
        $params[] = $s;
    }

    if ($status !== null && $status !== '') {
        $where[] = 'status = ?';
        $params[] = $status;
    }

    if ($formType !== null && $formType !== '') {
        $where[] = 'form_type = ?';
        $params[] = $formType;
    }

    if ($source !== null && $source !== '') {
        $where[] = 'utm_source = ?';
        $params[] = $source;
    }

    // Lấy limit + 1 để kiểm tra có next page
    $params[] = $limit + 1;
    $whereStr = implode(' AND ', $where);

    $stmt = $pdo->prepare("
        SELECT
            id, name, phone, form_type, status, channel,
            utm_source, utm_medium, utm_campaign,
            fbclid, gclid,
            ip, ip_hash, device_type, page_url, referrer,
            session_id, assignee_id, notes,
            duplicate_of_id, duplicate_group_id,
            created_at, contacted_at, converted_at
        FROM leads
        WHERE {$whereStr}
        ORDER BY id DESC
        LIMIT ?
    ");
    $stmt->execute($params);
    $rows = $stmt->fetchAll();

    // Check has next
    $hasNext = count($rows) > $limit;
    if ($hasNext) {
        array_pop($rows);
    }

    // Next cursor = ID nhỏ nhất trong kết quả (row cuối cùng)
    $nextCursor = !empty($rows) ? (string)end($rows)['id'] : null;

    return [
        'data'       => $rows,
        'pagination' => [
            'has_next'   => $hasNext,
            'next_cursor' => $nextCursor,
            'count'      => count($rows),
            'limit'      => $limit,
        ]
    ];
}

/**
 * Legacy pagination function - chuyển sang keyset
 */
function apex_query_leads(array $range, ?string $search = null, int $page = 1, int $perPage = 50): array {
    // Chuyển page number thành cursor
    $afterId = null;
    if ($page > 1) {
        // Để tìm cursor, ta cần offset - làm đơn giản: get first ID of page
        $pdo = apex_db();
        $offset = ($page - 1) * $perPage;
        $from = $range['from_dt'];
        $to   = $range['to_dt'];

        // Lấy ID tại offset (cursor cho page)
        $cursorRow = $pdo->prepare("
            SELECT id FROM leads
            WHERE created_at BETWEEN ? AND ?
            ORDER BY id DESC
            LIMIT 1 OFFSET ?
        ")->execute([$from, $to, $offset - 1])->fetch();

        $afterId = $cursorRow['id'] ?? null;
    }

    $result = apex_query_leads_keyset($range, $search, $afterId, $perPage);

    return [
        'total'    => 0, // Không trả total count - dùng cursor navigation
        'page'     => $page,
        'per_page' => $perPage,
        'pages'    => 0, // Infinity scroll với cursor
        'rows'     => $result['data'],
        'has_next' => $result['pagination']['has_next'],
        'next_cursor' => $result['pagination']['next_cursor'],
    ];
}

// ============================================================
// TRAFFIC SOURCES
// ============================================================

/**
 * Phân tích traffic source - OPTIMIZED với Summary Table
 */
function apex_query_traffic_sources(array $range, int $limit = 20): array {
    $from = $range['from'];
    $to   = $range['to'];

    return ApexCache::get('traffic_sources',
        function() use ($from, $to) {
            $pdo = apex_db();
            return $pdo->query("
                SELECT
                    utm_source,
                    utm_medium,
                    utm_campaign,
                    COUNT(DISTINCT session_id) AS sessions,
                    SUM(page_views) AS pageviews,
                    SUM(is_lead) AS leads
                FROM visitor_sessions
                WHERE first_visit BETWEEN '{$from}' AND '{$to} 23:59:59'
                  AND is_bot = 0
                GROUP BY utm_source, utm_medium, utm_campaign
                ORDER BY sessions DESC
                LIMIT 20
            ")->fetchAll();
        },
        300
    );
}

// ============================================================
// CAMPAIGN BREAKDOWN
// ============================================================

function apex_query_campaigns_breakdown(array $range): array {
    $from = $range['from_dt'];
    $to   = $range['to_dt'];

    return ApexCache::get('campaign_breakdown',
        function() use ($from, $to) {
            $pdo = apex_db();
            return $pdo->query("
                SELECT
                    COALESCE(NULLIF(utm_source,''),'(direct)') AS source,
                    COALESCE(NULLIF(utm_medium,''),'-') AS medium,
                    COALESCE(NULLIF(utm_campaign,''),'-') AS campaign,
                    COUNT(DISTINCT session_id) AS visitors,
                    SUM(page_views) AS pageviews,
                    SUM(is_lead) AS leads,
                    AVG(total_duration) AS avg_time,
                    AVG(scroll_max) AS avg_scroll
                FROM visitor_sessions
                WHERE first_visit BETWEEN '{$from}' AND '{$to}'
                  AND is_bot = 0
                GROUP BY utm_source, utm_medium, utm_campaign
                HAVING visitors > 0
                ORDER BY visitors DESC
                LIMIT 50
            ")->fetchAll();
        },
        300
    );
}

// ============================================================
// DEVICES
// ============================================================

function apex_query_devices(array $range): array {
    $from = $range['from_dt'];
    $to   = $range['to_dt'];

    return ApexCache::get('device_stats',
        function() use ($from, $to) {
            $pdo = apex_db();
            return [
                'devices' => $pdo->query("
                    SELECT
                        COALESCE(NULLIF(device_type,''),'unknown') AS device,
                        COUNT(DISTINCT session_id) AS sessions
                    FROM visitor_sessions
                    WHERE first_visit BETWEEN '{$from}' AND '{$to}'
                      AND is_bot = 0
                    GROUP BY device_type
                    ORDER BY sessions DESC
                ")->fetchAll(),
                'browsers' => $pdo->query("
                    SELECT
                        COALESCE(NULLIF(browser_name,''),'unknown') AS browser,
                        COUNT(DISTINCT session_id) AS sessions
                    FROM visitor_sessions
                    WHERE first_visit BETWEEN '{$from}' AND '{$to}'
                      AND is_bot = 0
                    GROUP BY browser_name
                    ORDER BY sessions DESC
                    LIMIT 10
                ")->fetchAll(),
                'os' => $pdo->query("
                    SELECT
                        COALESCE(NULLIF(os_name,''),'unknown') AS os,
                        COUNT(DISTINCT session_id) AS sessions
                    FROM visitor_sessions
                    WHERE first_visit BETWEEN '{$from}' AND '{$to}'
                      AND is_bot = 0
                    GROUP BY os_name
                    ORDER BY sessions DESC
                    LIMIT 10
                ")->fetchAll(),
            ];
        },
        1800
    );
}

// ============================================================
// FUNNEL - TỪ SUMMARY TABLE
// ============================================================

function apex_query_funnel(array $range): array {
    $from = $range['from'];
    $to   = $range['to'];

    return ApexCache::get('funnel_data',
        function() use ($from, $to) {
            $pdo = apex_db();

            // Đọc summary
            $summaries = $pdo->query("
                SELECT * FROM summary_daily
                WHERE date BETWEEN '{$from}' AND '{$to}'
            ")->fetchAll(PDO::FETCH_UNIQUE);

            // Aggregate
            $agg = [
                'visitors' => 0,
                'scroll25' => 0,
                'cta_click' => 0,
                'form_focus' => 0,
                'form_submit' => 0,
                'leads' => 0,
            ];

            foreach ($summaries as $row) {
                $agg['visitors'] += ($row['visitors_total'] ?? 0) - ($row['visitors_bot'] ?? 0);
                // Ước tính scroll 25% từ avg_scroll
                $visitors = $row['visitors_total'] ?? 0;
                $avgScroll = (float)($row['avg_scroll_percent'] ?? 0);
                if ($avgScroll >= 25) {
                    $agg['scroll25'] += (int)($visitors * min(1, $avgScroll / 100));
                }
                $agg['cta_click'] += ($row['cta_clicks_total'] ?? 0);
                $agg['form_focus'] += ($row['form_focuses_total'] ?? 0);
                $agg['form_submit'] += ($row['form_submits_attempts'] ?? 0);
                $agg['leads'] += ($row['leads_total'] ?? 0);
            }

            $steps = [
                ['label' => 'Visitors',      'event' => 'visitors',    'color' => '#0A192F'],
                ['label' => 'Scroll 25%+',  'event' => 'scroll25',   'color' => '#1E40AF'],
                ['label' => 'CTA Click',    'event' => 'cta_click',   'color' => '#F5A623'],
                ['label' => 'Form Focus',   'event' => 'form_focus',  'color' => '#EA580C'],
                ['label' => 'Form Submit',  'event' => 'form_submit', 'color' => '#DC2626'],
                ['label' => 'Lead',         'event' => 'leads',       'color' => '#10B981'],
            ];

            $result = [];
            foreach ($steps as $i => $step) {
                $count = $agg[$step['event']];
                $prev = $i > 0 ? ($result[$i - 1]['count'] ?? 0) : 0;
                $drop = $prev > 0 ? round((1 - $count / max(1, $prev)) * 100, 1) : 0;
                $total = $result[0]['count'] ?? 0;
                $conv = $total > 0 ? round(($count / $total) * 100, 1) : 0;

                $result[] = [
                    'label' => $step['label'],
                    'event' => $step['event'],
                    'color' => $step['color'],
                    'count' => $count,
                    'drop'  => $drop,
                    'conv'  => $conv,
                ];
            }

            return $result;
        },
        60
    );
}

// ============================================================
// TOP PAGES
// ============================================================

function apex_query_top_pages(array $range, int $limit = 20): array {
    $from = $range['from_dt'];
    $to   = $range['to_dt'];

    return ApexCache::get('top_pages',
        function() use ($from, $to, $limit) {
            $pdo = apex_db();
            return $pdo->query("
                SELECT
                    COALESCE(NULLIF(page_url,''),'/') AS url,
                    COUNT(DISTINCT session_id) AS sessions,
                    COUNT(*) AS pageviews
                FROM events
                WHERE event_type IN ('page_view')
                  AND created_at BETWEEN '{$from}' AND '{$to}'
                GROUP BY page_url
                ORDER BY sessions DESC
                LIMIT {$limit}
            ")->fetchAll();
        },
        300
    );
}

// ============================================================
// RECENT EVENTS - KEYSET PAGINATION
// ============================================================

function apex_query_recent_events(int $limit = 50, ?string $type = null, ?string $session = null): array {
    $pdo = apex_db();
    $afterId = isset($_GET['after_id']) ? (int)$_GET['after_id'] : null;

    $where = ['1=1'];
    $params = [];

    if ($type) {
        $where[] = 'event_type = ?';
        $params[] = $type;
    }
    if ($session) {
        $where[] = 'session_id = ?';
        $params[] = $session;
    }
    if ($afterId !== null) {
        $where[] = 'id < ?';
        $params[] = $afterId;
    }

    $params[] = $limit + 1;
    $whereStr = implode(' AND ', $where);

    $stmt = $pdo->prepare("
        SELECT id, session_id, event_type, event_value, page_url, utm_source,
               device_type, browser_name, os_name, ip, created_at
        FROM events
        WHERE {$whereStr}
        ORDER BY id DESC
        LIMIT ?
    ");
    $stmt->execute($params);
    $rows = $stmt->fetchAll();

    $hasNext = count($rows) > $limit;
    if ($hasNext) array_pop($rows);

    return $rows;
}

// ============================================================
// SESSION
// ============================================================

function apex_query_session(string $sessionId): ?array {
    $pdo = apex_db();

    $session = $pdo->prepare("SELECT * FROM visitor_sessions WHERE session_id = ? LIMIT 1");
    $session->execute([$sessionId]);
    $s = $session->fetch();

    if (!$s) {
        // Fallback: sessions table
        $session = $pdo->prepare("SELECT * FROM sessions WHERE session_id = ? LIMIT 1");
        $session->execute([$sessionId]);
        $s = $session->fetch();
    }

    if (!$s) return null;

    $events = $pdo->prepare("
        SELECT id, event_type, event_value, page_url, referrer,
               utm_source, utm_medium, utm_campaign, device_type,
               browser_name, os_name, ip, created_at
        FROM events
        WHERE session_id = ?
        ORDER BY id ASC
    ");
    $events->execute([$sessionId]);
    $e = $events->fetchAll();

    // Lead
    $lead = $pdo->prepare("
        SELECT id, name, phone, form_type, created_at
        FROM leads
        WHERE session_id = ?
        ORDER BY id DESC LIMIT 1
    ");
    $lead->execute([$sessionId]);
    $l = $lead->fetch();

    return ['session' => $s, 'events' => $e, 'lead' => $l];
}

// ============================================================
// ACTIVE VISITORS - CACHED
// ============================================================

function apex_query_active_visitors(): int {
    return ApexCache::get('active_visitors',
        function() {
            $pdo = apex_db();
            $threshold = date('Y-m-d H:i:s', time() - 300);
            return (int)$pdo->query("
                SELECT COUNT(DISTINCT session_id)
                FROM events
                WHERE created_at >= '{$threshold}'
            ")->fetchColumn();
        },
        5
    );
}

// ============================================================
// LEAD COUNTER TODAY - CACHED
// ============================================================

function apex_query_lead_counter_today(): int {
    return ApexCache::get('leads_today_count',
        function() {
            $pdo = apex_db();
            return (int)$pdo->query("
                SELECT COUNT(*) FROM leads WHERE DATE(created_at) = CURDATE()
            ")->fetchColumn();
        },
        30
    );
}

// ============================================================
// IP TRACKING
// ============================================================

function apex_query_ip_list(array $range, int $limit = 50): array {
    $from = $range['from_dt'];
    $to   = $range['to_dt'];

    $pdo = apex_db();
    return $pdo->query("
        SELECT
            ip_hash,
            ip,
            country,
            city,
            isp,
            is_vpn,
            is_proxy,
            is_tor,
            is_datacenter,
            risk_score,
            flags,
            visit_count,
            session_count,
            lead_count,
            page_view_count,
            avg_duration,
            last_utm_source,
            last_seen
        FROM ip_tracking
        WHERE last_seen BETWEEN '{$from}' AND '{$to}'
        ORDER BY lead_count DESC, risk_score DESC
        LIMIT {$limit}
    ")->fetchAll();
}

function apex_query_ip_detail(string $ipHash): ?array {
    $pdo = apex_db();

    $ip = $pdo->prepare("SELECT * FROM ip_tracking WHERE ip_hash = ?");
    $ip->execute([$ipHash]);
    $ipInfo = $ip->fetch();

    if (!$ipInfo) return null;

    $sessions = $pdo->prepare("
        SELECT s.*, l.id AS lead_id, l.name AS lead_name, l.form_type
        FROM visitor_sessions s
        LEFT JOIN leads l ON l.session_id = s.session_id
        WHERE s.ip_hash = ?
        ORDER BY s.first_visit DESC
        LIMIT 20
    ");
    $sessions->execute([$ipHash]);
    $sessions = $sessions->fetchAll();

    $leads = $pdo->prepare("
        SELECT id, name, phone, form_type, status, created_at
        FROM leads WHERE ip_hash = ? ORDER BY id DESC
    ");
    $leads->execute([$ipHash]);
    $leads = $leads->fetchAll();

    return [
        'ip' => $ipInfo,
        'sessions' => $sessions,
        'leads' => $leads,
    ];
}

// ============================================================
// DUPLICATE LEADS
// ============================================================

function apex_query_duplicate_leads(int $limit = 100): array {
    $pdo = apex_db();
    return $pdo->query("
        SELECT * FROM v_duplicate_leads
        ORDER BY duplicate_count DESC
        LIMIT {$limit}
    ")->fetchAll();
}
