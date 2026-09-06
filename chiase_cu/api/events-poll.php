<?php
/**
 * API: Events Polling Endpoint
 *
 * Polling endpoint cho real-time event stream
 */

declare(strict_types=1);

require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/lead-helpers.php';

header('Content-Type: application/json; charset=utf-8');
header('Cache-Control: no-store');
header('X-Poll-Version: 2');

$clientIp = lead_client_ip();
if (!lead_rate_limit($clientIp, 'poll_events', 120, 60)) {
    http_response_code(429);
    echo json_encode(['ok' => false, 'error' => 'rate_limited']);
    exit;
}

try {
    $pdo = lead_db();

    $afterId = max(0, (int)($_GET['after_id'] ?? 0));
    $limit = min(100, max(1, (int)($_GET['limit'] ?? 50)));
    $eventType = trim($_GET['type'] ?? '');

    $params = [$afterId, $limit + 1];
    $where = ['id > ?'];

    if ($eventType !== '') {
        $where[] = 'event_type = ?';
        $params[] = $eventType;
    }

    $whereStr = implode(' AND ', $where);

    $stmt = $pdo->prepare("
        SELECT
            id, session_id, event_type, event_value, page_url,
            utm_source, utm_medium, utm_campaign,
            device_type, browser_name, os_name,
            ip_hash, created_at
        FROM events
        WHERE {$whereStr}
        ORDER BY id ASC
        LIMIT ?
    ");
    $stmt->execute($params);
    $rows = $stmt->fetchAll();

    $hasMore = count($rows) > $limit;
    if ($hasMore) array_pop($rows);

    // Sanitize
    foreach ($rows as &$row) {
        $row['ip'] = $row['ip_hash'];
        unset($row['ip_hash']);
    }
    unset($row);

    echo json_encode([
        'ok' => true,
        'new_count' => count($rows),
        'has_more' => $hasMore,
        'events' => $rows,
        'server_time' => (new DateTime())->format('c'),
    ], JSON_INVALID_UTF8_IGNORE);

} catch (Throwable $e) {
    lead_log_error('EVENTS_POLL_ERROR', 'server', $e->getMessage(), $e->getTraceAsString(), null, $clientIp);

    http_response_code(500);
    echo json_encode(['ok' => false, 'error' => 'server_error']);
}
