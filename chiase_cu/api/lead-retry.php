<?php
/**
 * APEX FINTECH - API: LEAD RETRY (từ client)
 *
 * POST /api/lead-retry.php
 *
 * Client gọi endpoint này khi:
 *   - User đã submit form trước đó
 *   - Network fail / không nhận được response
 *   - Client lưu payload và idempotency_key trong queue (localStorage)
 *   - Lần visit sau / reconnect → gửi lại tất cả pending qua đây
 *
 * Body:
 *   {
 *     "items": [
 *       { "idempotency_key": "uuid", "payload": {...} },
 *       { "idempotency_key": "uuid", "payload": {...} }
 *     ]
 *   }
 *
 * Response:
 *   {
 *     "ok": true,
 *     "results": [
 *       { "idempotency_key": "uuid", "ok": true, "id": 123, "deduplicated": false },
 *       { "idempotency_key": "uuid", "ok": false, "error": "invalid_phone" }
 *     ]
 *   }
 *
 * Đặc tính:
 *   - Batch tối đa 10 items/lần
 *   - Mỗi item xử lý độc lập (1 fail không ảnh hưởng các item khác)
 *   - Idempotency đảm bảo retry không tạo duplicate
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

$items = $data['items'] ?? [];
if (!is_array($items) || empty($items)) {
    http_response_code(400);
    echo json_encode(['ok' => false, 'error' => 'empty_items']);
    exit;
}

// Giới hạn batch
$items = array_slice($items, 0, 10);

// Rate limit (chung 1 IP) - cao hơn submit thường
$ip = lead_client_ip();
if (!lead_rate_limit($ip, 'lead_retry', 200)) {
    http_response_code(429);
    echo json_encode(['ok' => false, 'error' => 'rate_limited']);
    exit;
}

$results = [];

foreach ($items as $item) {
    if (!is_array($item)) {
        $results[] = ['ok' => false, 'error' => 'invalid_item'];
        continue;
    }

    $key = trim((string)($item['idempotency_key'] ?? ''));
    $payload = $item['payload'] ?? [];

    if (!lead_is_valid_uuid($key)) {
        $results[] = ['idempotency_key' => $key, 'ok' => false, 'error' => 'missing_idempotency_key'];
        lead_log_error('retry_missing_key', 'client',
            'Retry queue thiếu idempotency_key',
            ['item_keys' => array_keys($item)],
            ['payload' => $payload]
        );
        continue;
    }

    if (empty($payload) || !is_array($payload)) {
        $results[] = ['idempotency_key' => $key, 'ok' => false, 'error' => 'missing_payload'];
        continue;
    }

    // Validate tối thiểu (server-side recheck)
    $name = trim((string)($payload['name'] ?? ''));
    $phone = trim((string)($payload['phone'] ?? ''));
    $phoneDigits = preg_replace('/\D/', '', $phone);
    $phoneHash = lead_phone_hash($phone);

    if (mb_strlen($name, 'UTF-8') < 2 || strlen($phoneDigits) < 8) {
        lead_log_error('retry_validation_failed', 'client',
            'Retry payload không vượt validation',
            ['name_len' => mb_strlen($name, 'UTF-8'), 'phone_digits' => strlen($phoneDigits)],
            ['idempotency_key' => $key, 'phone_hash' => $phoneHash, 'payload' => $payload]
        );
        $results[] = ['idempotency_key' => $key, 'ok' => false, 'error' => 'validation_failed'];
        continue;
    }

    // Check idempotency
    try {
        $pdo = lead_db();
        $existing = $pdo->prepare("SELECT lead_id FROM lead_idempotency WHERE idempotency_key=?");
        $existing->execute([$key]);
        $existingLeadId = $existing->fetchColumn();

        if ($existingLeadId) {
            // Đã tồn tại → đánh ACK
            $pdo->prepare("UPDATE lead_retry_queue SET ack_sent=1, ack_at=NOW() WHERE idempotency_key=?")
                ->execute([$key]);
            lead_mark_recovered($key);
            $results[] = ['idempotency_key' => $key, 'ok' => true, 'id' => (int)$existingLeadId, 'deduplicated' => true];
            continue;
        }
    } catch (Throwable $e) {
        lead_log_error('retry_idempotency_check_failed', 'db',
            'Retry idempotency check failed: ' . $e->getMessage(),
            ['exception' => get_class($e)],
            ['idempotency_key' => $key, 'phone_hash' => $phoneHash]
        );
        $results[] = ['idempotency_key' => $key, 'ok' => false, 'error' => 'server_error'];
        continue;
    }

    // Chưa có → INSERT
    try {
        $pdo = lead_db();
        $pdo->beginTransaction();

        // Tăng retry_count
        $payload['idempotency_key'] = $key;
        $payload['page_url'] = $payload['page_url'] ?? '';
        $payload['user_agent'] = $payload['user_agent'] ?? ($_SERVER['HTTP_USER_AGENT'] ?? '');
        $payload['referrer'] = $payload['referrer'] ?? '';
        $payload['device_type'] = $payload['device_type'] ?? null;
        $payload['ip'] = $ip;
        $payload['form_type'] = $payload['form_type'] ?? 'unknown';
        $payload['channel'] = !empty($payload['channel']) ? $payload['channel'] : null;
        $payload['source'] = $payload['utm_source'] ?? null;
        $payload['medium'] = $payload['utm_medium'] ?? null;
        $payload['campaign'] = $payload['utm_campaign'] ?? null;
        $payload['content'] = $payload['utm_content'] ?? null;
        $payload['term'] = $payload['utm_term'] ?? null;
        $payload['fbclid'] = $payload['fbclid'] ?? null;
        $payload['gclid'] = $payload['gclid'] ?? null;

        $stmt = $pdo->prepare("
            INSERT INTO leads
                (name, phone, form_type, channel, source, medium, campaign, content, term,
                 fbclid, gclid, referrer, user_agent, device_type, ip, page_url,
                 idempotency_key, created_at)
            VALUES
                (:name, :phone, :form_type, :channel, :source, :medium, :campaign, :content, :term,
                 :fbclid, :gclid, :referrer, :user_agent, :device_type, :ip, :page_url,
                 :idempotency_key, NOW())
        ");
        $stmt->execute([
            'name'  => $name,
            'phone' => $phone,
            'form_type' => lead_substr($payload['form_type'], 20) ?? 'unknown',
            'channel'   => lead_substr($payload['channel'], 20),
            'source'    => lead_substr($payload['source'], 100),
            'medium'    => lead_substr($payload['medium'], 100),
            'campaign'  => lead_substr($payload['campaign'], 100),
            'content'   => lead_substr($payload['content'], 100),
            'term'      => lead_substr($payload['term'], 100),
            'fbclid'    => lead_substr($payload['fbclid'], 100),
            'gclid'     => lead_substr($payload['gclid'], 100),
            'referrer'  => lead_substr($payload['referrer'], 500),
            'user_agent'=> lead_substr($payload['user_agent'], 500),
            'device_type' => lead_substr($payload['device_type'], 20),
            'ip'        => $ip,
            'page_url'  => lead_substr($payload['page_url'], 255),
            'idempotency_key' => $key,
        ]);
        $leadId = (int)$pdo->lastInsertId();

        // Insert idempotency
        $pdo->prepare("INSERT INTO lead_idempotency (idempotency_key, lead_id, phone_hash, created_at) VALUES (?, ?, ?, NOW())")
            ->execute([$key, $leadId, $phoneHash]);

        // Insert retry_queue (server side)
        $pdo->prepare("INSERT INTO lead_retry_queue (lead_id, idempotency_key, ack_sent, ack_at, attempts, created_at) VALUES (?, ?, 1, NOW(), 1, NOW())")
            ->execute([$leadId, $key]);

        $pdo->commit();

        // Đánh dấu các lỗi trước đó (nếu có) là recovered
        lead_mark_recovered($key);

        $results[] = ['idempotency_key' => $key, 'ok' => true, 'id' => $leadId, 'deduplicated' => false];

    } catch (Throwable $e) {
        if (isset($pdo) && $pdo->inTransaction()) $pdo->rollBack();

        $code = $e->getCode();
        $isDup = ($code === '23000') || (stripos($e->getMessage(), 'Duplicate') !== false);

        if ($isDup) {
            // Race condition
            try {
                $existing = $pdo->prepare("SELECT lead_id FROM lead_idempotency WHERE idempotency_key=?");
                $existing->execute([$key]);
                $existingLeadId = $existing->fetchColumn();
                if ($existingLeadId) {
                    $results[] = ['idempotency_key' => $key, 'ok' => true, 'id' => (int)$existingLeadId, 'deduplicated' => true];
                    continue;
                }
            } catch (Throwable $e2) {}
        }

        lead_log_error('retry_insert_failed', 'db',
            'Retry INSERT thất bại: ' . $e->getMessage(),
            [
                'exception' => get_class($e),
                'file' => $e->getFile(),
                'line' => $e->getLine(),
                'sql_state' => $code
            ],
            ['idempotency_key' => $key, 'phone_hash' => $phoneHash, 'payload' => $payload]
        );

        $results[] = ['idempotency_key' => $key, 'ok' => false, 'error' => 'server_error'];
    }
}

echo json_encode([
    'ok' => true,
    'total' => count($items),
    'succeeded' => count(array_filter($results, fn($r) => $r['ok'] ?? false)),
    'failed' => count(array_filter($results, fn($r) => !($r['ok'] ?? false))),
    'results' => $results
]);
