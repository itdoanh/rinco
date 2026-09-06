<?php
/**
 * APEX FINTECH - API: LEAD ACK
 *
 * POST /api/lead-ack.php
 *
 * Client gọi endpoint này SAU KHI đã nhận được response thành công từ lead.php,
 * để server đánh dấu "ACK đã gửi thành công" trong lead_retry_queue.
 * Mục đích: nếu sau này có cron job quét các lead chưa ACK, sẽ bỏ qua các lead này.
 *
 * Body:
 *   { "idempotency_key": "uuid-v4" }
 *
 * Response:
 *   { "ok": true, "marked": true }   // đã đánh dấu ACK
 *   { "ok": true, "marked": false }  // không tìm thấy (có thể lead không tồn tại)
 *   { "ok": false, "error": "..." }
 */

require_once __DIR__ . '/lead-helpers.php';

if (lead_handle_preflight()) exit;

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['ok' => false, 'error' => 'method_not_allowed']);
    exit;
}

$data = lead_read_json();
if (empty($data)) $data = $_POST;

$key = trim((string)($data['idempotency_key'] ?? ''));
if (!lead_is_valid_uuid($key)) {
    http_response_code(400);
    echo json_encode(['ok' => false, 'error' => 'missing_idempotency_key']);
    exit;
}

try {
    $pdo = lead_db();
    $stmt = $pdo->prepare("
        UPDATE lead_retry_queue
        SET ack_sent=1, ack_at=NOW()
        WHERE idempotency_key=? AND ack_sent=0
    ");
    $stmt->execute([$key]);
    $marked = $stmt->rowCount() > 0;

    echo json_encode(['ok' => true, 'marked' => $marked]);
} catch (Throwable $e) {
    lead_log_error('ack_update_failed', 'db',
        'Không update được lead_retry_queue.ack_sent: ' . $e->getMessage(),
        ['exception' => get_class($e), 'file' => $e->getFile(), 'line' => $e->getLine()],
        ['idempotency_key' => $key]
    );
    http_response_code(500);
    echo json_encode(['ok' => false, 'error' => 'server_error']);
}
