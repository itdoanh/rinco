<?php
/**
 * API: Leads Polling Endpoint
 *
 * Short/Long Polling cho CRM real-time updates.
 * Nguyên tắc: "Chỉ hỏi những gì mới" - WHERE id > last_id
 *
 * Endpoint: GET /api/leads-poll.php?after_id=X&limit=20&status=new
 * Response: JSON array of new leads
 */

declare(strict_types=1);

require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/lead-helpers.php';

// Set headers
header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: no-store, no-cache, must-revalidate');
header('X-Poll-Version: 2');
header('Access-Control-Allow-Origin: *');

// Rate limit: 60 requests/minute per IP
$clientIp = lead_client_ip();
if (!lead_rate_limit($clientIp, 'poll_leads', 60, 60)) {
    http_response_code(429);
    echo json_encode([
        'ok' => false,
        'error' => 'rate_limited',
        'retry_after' => 60,
    ]);
    exit;
}

try {
    $pdo = lead_db();

    // Parse params
    $afterId = isset($_GET['after_id']) ? max(0, (int)$_GET['after_id']) : 0;
    $limit = min(100, max(1, (int)($_GET['limit'] ?? 20)));
    $status = trim($_GET['status'] ?? '');
    $formType = trim($_GET['form_type'] ?? '');
    $source = trim($_GET['source'] ?? '');

    // Build query với WHERE id > last_id (INDEX seek)
    $params = [$afterId, $limit + 1]; // +1 để check has_more
    $where = ['id > ?'];

    if ($status !== '') {
        $where[] = 'status = ?';
        $params[] = $status;
    }

    if ($formType !== '') {
        $where[] = 'form_type = ?';
        $params[] = $formType;
    }

    if ($source !== '') {
        $where[] = 'utm_source = ?';
        $params[] = $source;
    }

    $whereStr = implode(' AND ', $where);

    // Query: chỉ lấy leads MỚI HƠN last_id
    // Nhờ INDEX idx_events_polling, MySQL seek rất nhanh ~0.0001s
    $stmt = $pdo->prepare("
        SELECT
            id, name, phone, form_type, status, channel,
            utm_source, utm_medium, utm_campaign,
            utm_content, utm_term,
            fbclid, gclid,
            ip, ip_hash, device_type, page_url, referrer,
            session_id,
            created_at, contacted_at, converted_at
        FROM leads
        WHERE {$whereStr}
        ORDER BY id ASC
        LIMIT ?
    ");
    $stmt->execute($params);
    $rows = $stmt->fetchAll();

    // Check has more
    $hasMore = count($rows) > $limit;
    if ($hasMore) {
        array_pop($rows);
    }

    // Sanitize IP (chỉ trả về hash, không trả raw IP)
    foreach ($rows as &$row) {
        $row['ip'] = $row['ip_hash'];
        unset($row['ip_hash']);
    }
    unset($row);

    // Server timestamp
    $serverTime = (new DateTime())->format('c');

    echo json_encode([
        'ok' => true,
        'new_count' => count($rows),
        'has_more' => $hasMore,
        'leads' => $rows,
        'server_time' => $serverTime,
        'query' => [
            'after_id' => $afterId,
            'limit' => $limit,
            'status' => $status ?: null,
            'form_type' => $formType ?: null,
            'source' => $source ?: null,
        ],
    ], JSON_INVALID_UTF8_IGNORE);

} catch (Throwable $e) {
    // Log error
    lead_log_error(
        'POLL_ERROR',
        'server',
        $e->getMessage(),
        $e->getTraceAsString(),
        null,
        $clientIp
    );

    http_response_code(500);
    echo json_encode([
        'ok' => false,
        'error' => 'server_error',
        'message' => 'Internal server error',
    ]);
}
