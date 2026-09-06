<?php
/**
 * APEX Admin API - Live stats endpoint
 * GET /admin/api/stats.php?live=1
 *  Trả về active_visitors + leads_today cho live badge
 * GET /admin/api/stats.php?range=1&from=...&to=...
 *  Trả về full stats cho dashboard refresh
 *
 * Returns JSON only - KHÔNG redirect. 401 nếu chưa login.
 */

define('APEX_ADMIN', true);

// Bắt buộc: tắt error display, chỉ log
ini_set('display_errors', '0');
ini_set('log_errors', '1');

require_once __DIR__ . '/../includes/db.php';
require_once __DIR__ . '/../includes/auth.php';
require_once __DIR__ . '/../includes/queries.php';

// Khởi tạo session (cần thiết cho auth check)
apex_session_start();

header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: no-store');
header('X-Content-Type-Options: nosniff');

// Auth check - trả 401 JSON nếu chưa login (không redirect để JS xử lý được)
if (!apex_admin_authenticated()) {
    http_response_code(401);
    echo json_encode([
        'ok' => false,
        'error' => 'Unauthorized',
        'admin_url' => apex_admin_url('login.php'),
    ]);
    exit;
}

$mode = $_GET['live'] ?? $_GET['range'] ?? 'live';

try {
    if ($mode === 'live') {
        // Bypass cache để debug 500: chạy query trực tiếp
        $pdo = apex_db();
        $threshold = date('Y-m-d H:i:s', time() - 300);
        $visitors = (int)$pdo->query("
            SELECT COUNT(DISTINCT session_id) FROM events WHERE created_at >= '{$threshold}'
        ")->fetchColumn();
        $leads = (int)$pdo->query("
            SELECT COUNT(*) FROM leads WHERE DATE(created_at) = CURDATE()
        ")->fetchColumn();
        echo json_encode([
            'ok'              => true,
            'ts'              => time(),
            'active_visitors' => $visitors,
            'leads_today'     => $leads,
            'debug'           => 'bypass_cache',
        ]);
    } else {
        $range = apex_parse_date_range();
        $pdo = apex_db();
        $threshold = date('Y-m-d H:i:s', time() - 300);
        $visitors = (int)$pdo->query("
            SELECT COUNT(DISTINCT session_id) FROM events WHERE created_at >= '{$threshold}'
        ")->fetchColumn();
        $leads = (int)$pdo->query("
            SELECT COUNT(*) FROM leads WHERE DATE(created_at) = CURDATE()
        ")->fetchColumn();
        echo json_encode([
            'ok'              => true,
            'ts'              => time(),
            'overview'        => apex_query_overview($range),
            'active_visitors' => $visitors,
            'leads_today'     => $leads,
            'debug'           => 'bypass_cache',
        ]);
    }
} catch (Throwable $e) {
    http_response_code(500);
    $msg = $e->getMessage();
    $loc = $e->getFile() . ':' . $e->getLine();
    error_log("[admin/api/stats.php] {$msg} @ {$loc}");
    @file_put_contents(sys_get_temp_dir() . '/apex_debug.log', date('c') . " stats.php 500: {$msg} @ {$loc}\n", FILE_APPEND);
    echo json_encode(['ok' => false, 'error' => 'server_error', 'detail' => $msg, 'loc' => basename($loc)]);
}
