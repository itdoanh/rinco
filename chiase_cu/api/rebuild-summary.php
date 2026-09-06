<?php
/**
 * Rebuild summary_daily từ events + visitor_sessions + leads
 * Chạy 1 lần để fix dashboard trống khi cron chưa chạy
 *
 * URL: /chiase/api/rebuild-summary.php
 */

error_reporting(E_ALL);
ini_set('display_errors', '1');

header('Content-Type: text/plain; charset=utf-8');

try {
    require_once __DIR__ . '/../includes/config.php';
    require_once __DIR__ . '/../admin/includes/db.php';

    $pdo = apex_db();

    echo "=== REBUILD SUMMARY_DAILY ===\n\n";

    // Kiểm tra inputs
    $totalEvents = (int)$pdo->query("SELECT COUNT(*) FROM events")->fetchColumn();
    $totalSessions = (int)$pdo->query("SELECT COUNT(*) FROM visitor_sessions")->fetchColumn();
    $totalLeads = (int)$pdo->query("SELECT COUNT(*) FROM leads")->fetchColumn();
    $totalSummary = (int)$pdo->query("SELECT COUNT(*) FROM summary_daily")->fetchColumn();

    echo "INPUTS:\n";
    echo "  events: {$totalEvents}\n";
    echo "  visitor_sessions: {$totalSessions}\n";
    echo "  leads: {$totalLeads}\n";
    echo "  summary_daily (before): {$totalSummary}\n\n";

    if ($totalEvents === 0 && $totalSessions === 0 && $totalLeads === 0) {
        echo "KHONG CO DATA DE REBUILD\n";
        echo "Truy cap landing page va doi 30s roi chay lai file nay.\n";
        exit;
    }

    // Lấy schema của summary_daily
    $cols = $pdo->query("SHOW COLUMNS FROM summary_daily")->fetchAll(PDO::FETCH_COLUMN);
    echo "summary_daily columns: " . implode(', ', $cols) . "\n\n";

    // Lấy danh sách ngày có data
    $dates = $pdo->query("
        SELECT DISTINCT date_key FROM (
            SELECT DATE(created_at) AS date_key FROM events
            UNION
            SELECT DATE(first_visit) AS date_key FROM visitor_sessions
            UNION
            SELECT DATE(created_at) AS date_key FROM leads
        ) AS d
        ORDER BY date_key DESC
    ")->fetchAll(PDO::FETCH_COLUMN);

    echo "Dates to rebuild: " . count($dates) . " (latest: " . ($dates[0] ?? 'none') . ")\n\n";

    // Rebuild từng ngày
    $rebuilt = 0;
    foreach ($dates as $date) {
        // Tính các metric cho ngày $date

        // Visitors (unique sessions, không tính bot)
        $visitors_total = (int)$pdo->prepare("
            SELECT COUNT(DISTINCT session_id) FROM visitor_sessions
            WHERE DATE(first_visit) = ? AND is_bot = 0
        ")->execute([$date]) ?: 0;

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

        // Visitors theo UTM source
        $stmt = $pdo->prepare("
            SELECT
                SUM(CASE WHEN utm_source IN ('fb', 'facebook', 'ig', 'meta') THEN 1 ELSE 0 END) AS fb_visitors,
                SUM(CASE WHEN utm_source IN ('gg', 'google', 'adwords') THEN 1 ELSE 0 END) AS gg_visitors,
                SUM(CASE WHEN COALESCE(NULLIF(utm_source,''),'(direct)') = '(direct)' THEN 1 ELSE 0 END) AS direct_visitors
            FROM (
                SELECT session_id, utm_source FROM visitor_sessions
                WHERE DATE(first_visit) = ? AND is_bot = 0
                GROUP BY session_id
            ) AS uniq
        ");
        $stmt->execute([$date]);
        $utmRow = $stmt->fetch();
        $visitors_fb = (int)($utmRow['fb_visitors'] ?? 0);
        $visitors_gg = (int)($utmRow['gg_visitors'] ?? 0);
        $visitors_direct = (int)($utmRow['direct_visitors'] ?? 0);

        // Leads
        $stmt = $pdo->prepare("
            SELECT
                COUNT(*) AS total,
                SUM(CASE WHEN form_type = 'hero' THEN 1 ELSE 0 END) AS hero,
                SUM(CASE WHEN form_type = 'modal' THEN 1 ELSE 0 END) AS modal,
                SUM(CASE WHEN form_type NOT IN ('hero','modal') AND form_type IS NOT NULL THEN 1 ELSE 0 END) AS multistep
            FROM leads
            WHERE DATE(created_at) = ?
        ");
        $stmt->execute([$date]);
        $leadRow = $stmt->fetch();
        $leads_total = (int)($leadRow['total'] ?? 0);
        $leads_hero = (int)($leadRow['hero'] ?? 0);
        $leads_modal = (int)($leadRow['modal'] ?? 0);
        $leads_multistep = (int)($leadRow['multistep'] ?? 0);

        // Leads FB (co utm_source fb)
        $stmt = $pdo->prepare("
            SELECT COUNT(*) FROM leads
            WHERE DATE(created_at) = ?
              AND utm_source IN ('fb','facebook','ig','meta')
        ");
        $stmt->execute([$date]);
        $leads_fb = (int)$stmt->fetchColumn();

        // CTA clicks (event_type = 'cta_click')
        $stmt = $pdo->prepare("
            SELECT COUNT(*) FROM events
            WHERE DATE(created_at) = ? AND event_type = 'cta_click'
        ");
        $stmt->execute([$date]);
        $cta_clicks_total = (int)$stmt->fetchColumn();

        // Form focuses
        $stmt = $pdo->prepare("
            SELECT COUNT(*) FROM events
            WHERE DATE(created_at) = ? AND event_type = 'form_focus'
        ");
        $stmt->execute([$date]);
        $form_focuses_total = (int)$stmt->fetchColumn();

        // Form submit attempts
        $stmt = $pdo->prepare("
            SELECT COUNT(*) FROM events
            WHERE DATE(created_at) = ? AND event_type = 'form_submit'
        ");
        $stmt->execute([$date]);
        $form_submits_attempts = (int)$stmt->fetchColumn();

        // Avg duration
        $stmt = $pdo->prepare("
            SELECT AVG(total_duration) FROM visitor_sessions
            WHERE DATE(first_visit) = ? AND is_bot = 0 AND total_duration > 0
        ");
        $stmt->execute([$date]);
        $avg_duration = (int)$stmt->fetchColumn();

        // Avg scroll
        $stmt = $pdo->prepare("
            SELECT AVG(scroll_max) FROM visitor_sessions
            WHERE DATE(first_visit) = ? AND is_bot = 0 AND scroll_max > 0
        ");
        $stmt->execute([$date]);
        $avg_scroll = (int)$stmt->fetchColumn();

        // Upsert
        $pdo->prepare("
            INSERT INTO summary_daily
                (date, visitors_total, visitors_bot, visitors_fb, visitors_gg, visitors_direct,
                 leads_total, leads_hero, leads_modal, leads_multistep, leads_fb,
                 cta_clicks_total, form_focuses_total, form_submits_attempts,
                 avg_duration_seconds, avg_scroll_percent)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
                cta_clicks_total = VALUES(cta_clicks_total),
                form_focuses_total = VALUES(form_focuses_total),
                form_submits_attempts = VALUES(form_submits_attempts),
                avg_duration_seconds = VALUES(avg_duration_seconds),
                avg_scroll_percent = VALUES(avg_scroll_percent)
        ")->execute([
            $date, $visitors_total, $visitors_bot, $visitors_fb, $visitors_gg, $visitors_direct,
            $leads_total, $leads_hero, $leads_modal, $leads_multistep, $leads_fb,
            $cta_clicks_total, $form_focuses_total, $form_submits_attempts,
            $avg_duration, $avg_scroll
        ]);

        $rebuilt++;
        echo "  [{$date}] visitors={$visitors_total} (fb={$visitors_fb}) leads={$leads_total} cta={$cta_clicks_total}\n";
    }

    // Xóa cache để dashboard refresh
    echo "\n=== XOA CACHE ===\n";
    try {
        if (function_exists('apex_cache_clear_all')) {
            apex_cache_clear_all();
            echo "  apex_cache_clear_all() OK\n";
        } else {
            echo "  Khong co ham apex_cache_clear_all - can xoa cache khac\n";
        }
    } catch (Throwable $e) {
        echo "  Cache clear warning: " . $e->getMessage() . "\n";
    }

    // Verify
    $totalAfter = (int)$pdo->query("SELECT COUNT(*) FROM summary_daily")->fetchColumn();
    echo "\n=== KET QUA ===\n";
    echo "  Rebuilt: {$rebuilt} days\n";
    echo "  summary_daily (after): {$totalAfter}\n";
    echo "\nReload dashboard ngay bay gio se thay data!\n";

} catch (Throwable $e) {
    echo "\nLOI PHP:\n";
    echo "Message: " . $e->getMessage() . "\n";
    echo "File: " . $e->getFile() . "\n";
    echo "Line: " . $e->getLine() . "\n";
}
