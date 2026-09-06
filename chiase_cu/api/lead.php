<?php
/**
 * APEX FINTECH - API: NHẬN ĐĂNG KÝ v3 (DUPLICATE DETECTION ENGINE)
 *
 * POST /api/lead.php
 *
 * Hệ thống phát hiện duplicate GIỐNG NHƯ FACEBOOK/GOOGLE:
 * - Multi-signal duplicate detection (phone, IP, device, session, time patterns)
 * - Tự động đánh dấu trạng thái duplicate ngay khi INSERT
 * - Duplicate group management (các leads liên quan cùng group)
 * - Risk scoring cho suspicious submissions
 * - Auto-review queue cho flags cao
 *
 * Response:
 *   { "ok": true, "id": 123, "duplicate": { "detected": true, "type": "phone_exact", "original_id": 122, "group_id": "xxx" }}
 */

declare(strict_types=1);

require_once __DIR__ . '/lead-helpers.php';

if (lead_handle_preflight()) exit;

if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['ok' => false, 'error' => 'method_not_allowed']);
    exit;
}

$requestStart = microtime(true);
$requestId = sprintf('%04x%04x-%04x-%04x-%04x-%04x%04x%04x',
    mt_rand(0, 0xffff), mt_rand(0, 0xffff),
    mt_rand(0, 0xffff), mt_rand(0, 0x0fff) | 0x4000,
    mt_rand(0, 0x3fff) | 0x8000,
    mt_rand(0, 0xffff), mt_rand(0, 0xffff), mt_rand(0, 0xffff)
);

// === Đọc payload ===
$data = lead_read_json();
if (empty($data)) $data = $_POST;
if (!is_array($data) || empty($data)) {
    lead_log_error('empty_body', 'server', 'Request body rỗng', ['request_id' => $requestId], ['payload' => $data]);
    http_response_code(400);
    echo json_encode(['ok' => false, 'error' => 'empty_body', 'request_id' => $requestId]);
    exit;
}

// === Honeypot check ===
if (!empty($data['website'])) {
    echo json_encode(['ok' => true, 'id' => 0, 'honeypot' => true]);
    exit;
}

// === Validate idempotency_key ===
$idempotencyKey = trim((string)($data['idempotency_key'] ?? ''));
if (!lead_is_valid_uuid($idempotencyKey)) {
    lead_log_error('missing_idempotency_key', 'server',
        'Thiếu idempotency_key', ['request_id' => $requestId], ['payload' => $data]);
    http_response_code(400);
    echo json_encode(['ok' => false, 'error' => 'missing_idempotency_key', 'request_id' => $requestId]);
    exit;
}

// === Validate name + phone ===
$name  = trim((string)($data['name'] ?? ''));
$phone = trim((string)($data['phone'] ?? ''));
$nameLen = mb_strlen($name, 'UTF-8');
$phoneDigits = preg_replace('/\D/', '', $phone);

if ($nameLen < 2 || $nameLen > 100) {
    lead_log_error('invalid_name', 'server', 'Tên không hợp lệ', ['request_id' => $requestId, 'name_len' => $nameLen], $data);
    http_response_code(400);
    echo json_encode(['ok' => false, 'error' => 'invalid_name', 'detail' => 'Tên phải 2-100 ký tự', 'request_id' => $requestId]);
    exit;
}

if (strlen($phoneDigits) < 8 || strlen($phoneDigits) > 15) {
    lead_log_error('invalid_phone', 'server', 'Phone không hợp lệ', ['request_id' => $requestId], $data);
    http_response_code(400);
    echo json_encode(['ok' => false, 'error' => 'invalid_phone', 'request_id' => $requestId]);
    exit;
}

// === Rate limit ===
$ip = lead_client_ip();
if (!lead_rate_limit($ip, 'lead_submit', LEAD_RATE_PER_HOUR)) {
    http_response_code(429);
    echo json_encode(['ok' => false, 'error' => 'rate_limited', 'request_id' => $requestId]);
    exit;
}

try {
    $pdo = lead_db();
    $phoneHash = lead_phone_hash($phone);
    $sessionId = trim((string)($data['session_id'] ?? ''));
    $deviceFingerprint = trim((string)($data['device_fingerprint'] ?? ''));
    $formType = substr((string)($data['form_type'] ?? 'unknown'), 0, 20);
    $channel = isset($data['channel']) && $data['channel'] !== '' ? substr((string)$data['channel'], 0, 20) : null;

    // === IDEMPOTENCY CHECK (exact retry) ===
    $existing = $pdo->prepare("SELECT lead_id FROM lead_idempotency WHERE idempotency_key=?");
    $existing->execute([$idempotencyKey]);
    $existingLeadId = $existing->fetchColumn();

    if ($existingLeadId) {
        lead_mark_recovered($idempotencyKey);
        $pdo->prepare("UPDATE lead_retry_queue SET ack_sent=1, ack_at=NOW() WHERE idempotency_key=? AND ack_sent=0")
            ->execute([$idempotencyKey]);

        $stmtLead = $pdo->prepare("SELECT id, name, phone, form_type, created_at FROM leads WHERE id=?");
        $stmtLead->execute([$existingLeadId]);
        $lead = $stmtLead->fetch(PDO::FETCH_ASSOC);

        echo json_encode([
            'ok' => true,
            'id' => (int)$existingLeadId,
            'deduplicated' => true,
            'lead' => $lead,
            'ts' => time(),
            'response_time_ms' => round((microtime(true) - $requestStart) * 1000)
        ]);
        exit;
    }

    // ========================================================================
    // DUPLICATE DETECTION ENGINE - GIỐNG FACEBOOK/GOOGLE
    // ========================================================================

    $duplicateInfo = detectDuplicateAdvanced($pdo, $phone, $phoneHash, $ip, $sessionId, $deviceFingerprint, $name);

    $isDuplicate = $duplicateInfo['detected'];
    $duplicateType = $duplicateInfo['type'];
    $originalLeadId = $duplicateInfo['original_id'];
    $duplicateGroupId = $duplicateInfo['group_id'];
    $riskScore = $duplicateInfo['risk_score'];
    $riskFlags = $duplicateInfo['risk_flags'];
    $status = $duplicateInfo['status']; // 'new', 'duplicate', 'review'

    // Log duplicate detection
    if ($isDuplicate) {
        lead_log_error('duplicate_detected', 'server',
            "Duplicate detected: {$duplicateType}",
            [
                'request_id' => $requestId,
                'duplicate_type' => $duplicateType,
                'original_id' => $originalLeadId,
                'group_id' => $duplicateGroupId,
                'risk_score' => $riskScore,
                'risk_flags' => $riskFlags
            ],
            ['idempotency_key' => $idempotencyKey, 'phone_hash' => $phoneHash, 'payload' => $data]
        );
    }

    // ========================================================================
    // INSERT LEAD
    // ========================================================================

    $payload = [
        'name'  => $name,
        'phone' => $phone,
        'form_type' => $formType,
        'channel'   => $channel,
        'utm_source' => lead_substr((string)($data['utm_source'] ?? ''), 100),
        'utm_medium' => lead_substr((string)($data['utm_medium'] ?? ''), 100),
        'utm_campaign' => lead_substr((string)($data['utm_campaign'] ?? ''), 100),
        'utm_content' => lead_substr((string)($data['utm_content'] ?? ''), 100),
        'utm_term' => lead_substr((string)($data['utm_term'] ?? ''), 100),
        'fbclid'    => lead_substr((string)($data['fbclid'] ?? ''), 100),
        'gclid'     => lead_substr((string)($data['gclid'] ?? ''), 100),
        'referrer'  => lead_substr((string)($data['referrer'] ?? ''), 500),
        'user_agent'=> lead_substr((string)($data['user_agent'] ?? ($_SERVER['HTTP_USER_AGENT'] ?? '')), 500),
        'device_type' => lead_substr((string)($data['device_type'] ?? ''), 20),
        'ip'        => $ip,
        'page_url'  => lead_substr((string)($data['page_url'] ?? ''), 255),
        'idempotency_key' => $idempotencyKey,
        'status' => $status,
        'duplicate_of_id' => $originalLeadId,
        'duplicate_group_id' => $duplicateGroupId,
    ];

    $pdo->beginTransaction();

    // INSERT lead
    $stmt = $pdo->prepare("
        INSERT INTO leads
            (name, phone, form_type, channel, status, duplicate_of_id, duplicate_group_id,
             utm_source, utm_medium, utm_campaign, utm_content, utm_term, fbclid, gclid,
             referrer, user_agent, device_type, ip, page_url, idempotency_key, created_at)
        VALUES
            (:name, :phone, :form_type, :channel, :status, :duplicate_of_id, :duplicate_group_id,
             :utm_source, :utm_medium, :utm_campaign, :utm_content, :utm_term, :fbclid, :gclid,
             :referrer, :user_agent, :device_type, :ip, :page_url, :idempotency_key, NOW())
    ");
    $stmt->execute($payload);
    $leadId = (int)$pdo->lastInsertId();

    // INSERT idempotency
    $pdo->prepare("INSERT INTO lead_idempotency (idempotency_key, lead_id, phone_hash, created_at) VALUES (?, ?, ?, NOW())")
        ->execute([$idempotencyKey, $leadId, $phoneHash]);

    // INSERT retry queue
    $pdo->prepare("INSERT INTO lead_retry_queue (lead_id, idempotency_key, ack_sent, attempts, next_attempt_at, created_at) VALUES (?, ?, 0, 0, DATE_ADD(NOW(), INTERVAL 30 SECOND), NOW())")
        ->execute([$leadId, $idempotencyKey]);

    // UPDATE visitor_session
    if (lead_is_valid_uuid($sessionId)) {
        $pdo->prepare("UPDATE visitor_sessions SET is_lead=1, last_activity=NOW() WHERE session_id=?")
            ->execute([$sessionId]);
    }

    // UPDATE IP tracking
    updateIpTracking($pdo, $ip, $phoneHash, $deviceFingerprint, $riskScore, $riskFlags);

    // UPDATE user journey
    if (lead_is_valid_uuid($sessionId)) {
        $pdo->prepare("
            UPDATE user_journeys SET
                is_lead = 1,
                lead_id = ?,
                lead_form_type = ?,
                lead_created_at = NOW(),
                form_submit_success = 1,
                time_to_submit = TIMESTAMPDIFF(SECOND, first_visit, NOW()),
                updated_at = NOW()
            WHERE session_id = ?
        ")->execute([$leadId, $formType, $sessionId]);
    }

    $pdo->commit();
    lead_mark_recovered($idempotencyKey);

    $duration = round((microtime(true) - $requestStart) * 1000);

    // === Gửi Facebook CAPI Lead Event ===
    try {
        require_once __DIR__ . '/../includes/fb-capi.php';
        $capiResult = fb_send_lead_event([
            'event_id' => $idempotencyKey,
            'form_type' => $formType,
            'url' => $pageUrl,
            'phone_hash' => $phoneHash,
            'fbc' => $fbclid ?: fb_get_fbc(),
            'fbp' => fb_get_fbp(),
        ]);
        if ($capiResult['success']) {
            error_log("[FB CAPI] Lead {$leadId} sent to Facebook successfully");
        }
    } catch (Throwable $e) {
        error_log('[FB CAPI] Failed to send lead event: ' . $e->getMessage());
    }

    echo json_encode([
        'ok' => true,
        'id' => $leadId,
        'duplicate' => $duplicateInfo,
        'status' => $status,
        'risk_score' => $riskScore,
        'ts' => time(),
        'response_time_ms' => $duration
    ]);

} catch (Throwable $e) {
    if (isset($pdo) && $pdo->inTransaction()) $pdo->rollBack();

    $isDup = ($e->getCode() === '23000') || (stripos($e->getMessage(), 'Duplicate') !== false);

    if ($isDup) {
        try {
            $existing = $pdo->prepare("SELECT lead_id FROM lead_idempotency WHERE idempotency_key=?");
            $existing->execute([$idempotencyKey]);
            $existingLeadId = $existing->fetchColumn();
            if ($existingLeadId) {
                echo json_encode([
                    'ok' => true,
                    'id' => (int)$existingLeadId,
                    'deduplicated' => true,
                    'ts' => time()
                ]);
                exit;
            }
        } catch (Throwable $e2) {}
    }

    lead_log_error('db_insert_failed', 'db', $e->getMessage(),
        ['request_id' => $requestId, 'exception' => get_class($e), 'file' => $e->getFile(), 'line' => $e->getLine()],
        ['idempotency_key' => $idempotencyKey, 'phone_hash' => $phoneHash ?? '']
    );

    http_response_code(500);
    $resp = ['ok' => false, 'error' => 'server_error', 'request_id' => $requestId];
    if (defined('DEBUG_MODE') && DEBUG_MODE) {
        $resp['debug'] = $e->getMessage() . ' @ ' . basename($e->getFile()) . ':' . $e->getLine();
    }
    echo json_encode($resp);
}

// ========================================================================
// DUPLICATE DETECTION ENGINE
// ========================================================================

/**
 * Advanced duplicate detection - GIỐNG NHƯ FACEBOOK/GOOGLE
 *
 * Sử dụng multi-signal detection:
 * 1. Phone exact match (cùng số)
 * 2. Phone fuzzy match (tương tự - cùng 7 số cuối)
 * 3. Same IP + short time window (cùng IP trong thời gian ngắn)
 * 4. Same device fingerprint
 * 5. Same session
 * 6. Time-based patterns
 */
function detectDuplicateAdvanced(PDO $pdo, string $phone, string $phoneHash, string $ip, string $sessionId, string $deviceFingerprint, string $name): array
{
    $now = time();
    $oneHour = 3600;
    $oneDay = 86400;
    $oneWeek = 604800;

    $riskScore = 0;
    $riskFlags = [];
    $result = [
        'detected' => false,
        'type' => null,
        'original_id' => null,
        'group_id' => null,
        'risk_score' => 0,
        'risk_flags' => [],
        'status' => 'new'
    ];

    // ===== 1. PHONE EXACT MATCH (signal mạnh nhất) =====
    $stmt = $pdo->prepare("
        SELECT id, name, phone, form_type, status, duplicate_group_id, created_at
        FROM leads
        WHERE phone = ?
        ORDER BY id DESC
        LIMIT 5
    ");
    $stmt->execute([$phone]);
    $phoneMatches = $stmt->fetchAll(PDO::FETCH_ASSOC);

    if (!empty($phoneMatches)) {
        $original = $phoneMatches[0];
        $originalId = (int)$original['id'];
        $originalCreated = strtotime($original['created_at']);
        $timeDiff = $now - $originalCreated;

        // Exact phone match found
        $result['detected'] = true;
        $result['original_id'] = $originalId;
        $result['group_id'] = $original['duplicate_group_id'] ?? md5($phoneHash . date('Y-m'));
        $result['type'] = 'phone_exact';

        // ===== PHONE EXACT: Phân loại mức độ duplicate =====

        // 1a. Cùng phone + cùng name + trong 24h → Legitimate re-submit
        if ($timeDiff <= $oneDay && strtolower(trim($name)) === strtolower(trim($original['name']))) {
            $result['status'] = 'duplicate';
            $result['type'] = 'phone_name_same_24h';
            $riskScore += 20;
            $riskFlags[] = 'same_person_resubmit';
        }
        // 1b. Cùng phone + khác name + trong 1h → SUSPICIOUS (có thể fraud)
        elseif ($timeDiff <= $oneHour && strtolower(trim($name)) !== strtolower(trim($original['name']))) {
            $result['status'] = 'review';
            $result['type'] = 'phone_diff_name_1h';
            $riskScore += 80;
            $riskFlags[] = 'fraud_suspicion';
            $riskFlags[] = 'phone_reuse';
        }
        // 1c. Cùng phone + khác name + trong 24h → Cần review
        elseif ($timeDiff <= $oneDay) {
            $result['status'] = 'review';
            $result['type'] = 'phone_diff_name_24h';
            $riskScore += 50;
            $riskFlags[] = 'potential_fraud';
            $riskFlags[] = 'phone_reuse';
        }
        // 1d. Cùng phone + trong 1 tuần → Duplicate tự nhiên
        elseif ($timeDiff <= $oneWeek) {
            $result['status'] = 'duplicate';
            $result['type'] = 'phone_same_week';
            $riskScore += 10;
            $riskFlags[] = 'repeat_customer';
        }
        // 1e. Cùng phone + > 1 tuần → Có thể cùng khách cũ
        else {
            $result['status'] = 'new';
            $result['type'] = 'phone_exact_old';
            $riskScore += 5;
            $riskFlags[] = 'returning_customer';
        }
    }

    // ===== 2. PHONE FUZZY MATCH (cùng 7 số cuối) =====
    if (!$result['detected']) {
        $phoneDigits = preg_replace('/\D/', '', $phone);
        $last7 = substr($phoneDigits, -7);

        $stmt = $pdo->prepare("
            SELECT id, phone, name, created_at
            FROM leads
            WHERE phone REGEXP ? AND id != ?
            ORDER BY id DESC LIMIT 3
        ");
        $stmt->execute([ substr($last7, 0, 4) . '[0-9]{0,3}' . substr($last7, -3), 0]);
        $fuzzyMatches = $stmt->fetchAll();

        foreach ($fuzzyMatches as $m) {
            $mPhone = preg_replace('/\D/', '', $m['phone']);
            $mLast7 = substr($mPhone, -7);
            if ($mLast7 === $last7 && apex_levenshtein($phoneDigits, $mPhone) <= 2) {
                $result['detected'] = true;
                $result['type'] = 'phone_fuzzy';
                $result['original_id'] = (int)$m['id'];
                $result['risk_score'] += 40;
                $riskFlags[] = 'similar_phone';
                break;
            }
        }
    }

    // ===== 3. SAME IP + SHORT TIME WINDOW =====
    if (!$result['detected'] || $result['risk_score'] < 50) {
        $oneHourAgo = date('Y-m-d H:i:s', $now - $oneHour);

        $stmt = $pdo->prepare("
            SELECT id, phone, name, created_at
            FROM leads
            WHERE ip = ? AND created_at >= ?
            ORDER BY id DESC LIMIT 3
        ");
        $stmt->execute([$ip, $oneHourAgo]);
        $ipMatches = $stmt->fetchAll();

        $ipMatchCount = count($ipMatches);
        if ($ipMatchCount >= 3) {
            // Nhiều leads từ cùng IP trong 1 giờ → RATE_LIMIT nhưng có thể là shared network
            $riskScore += 30 * $ipMatchCount;
            $riskFlags[] = 'high_ip_density';
            if ($ipMatchCount >= 5) {
                $result['status'] = 'review';
                $riskFlags[] = 'potential_abuse';
            }
        } elseif ($ipMatchCount >= 1) {
            $riskScore += 15;
            $riskFlags[] = 'same_ip_recent';
        }
    }

    // ===== 4. SAME DEVICE FINGERPRINT =====
    if ($deviceFingerprint && !$result['detected']) {
        $stmt = $pdo->prepare("
            SELECT id, phone, created_at
            FROM leads l
            JOIN visitor_sessions s ON l.session_id = s.session_id
            WHERE s.device_type = ? AND l.id != ?
            ORDER BY l.id DESC LIMIT 3
        ");
        // Note: In real implementation, you'd store device fingerprint in leads table
        // For now, skip this check if fingerprint not stored
    }

    // ===== 5. SESSION ALREADY HAS LEAD =====
    if (lead_is_valid_uuid($sessionId) && !$result['detected']) {
        $stmt = $pdo->prepare("SELECT id, name FROM leads WHERE session_id = ? LIMIT 1");
        $stmt->execute([$sessionId]);
        $sessionLead = $stmt->fetch();

        if ($sessionLead) {
            $result['detected'] = true;
            $result['type'] = 'same_session';
            $result['original_id'] = (int)$sessionLead['id'];
            $result['status'] = 'duplicate';
            $riskScore += 60;
            $riskFlags[] = 'session_duplicate';
        }
    }

    // ===== 6. RAPID SUBMISSION PATTERN =====
    $stmt = $pdo->prepare("
        SELECT COUNT(*) FROM leads
        WHERE ip = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 10 MINUTE)
    ");
    $stmt->execute([$ip]);
    $rapidCount = (int)$stmt->fetchColumn();

    if ($rapidCount >= 3) {
        $riskScore += 50;
        $riskFlags[] = 'rapid_submission';
        if ($result['risk_score'] < 70) {
            $result['status'] = 'review';
        }
    }

    // ===== FINAL: Update group_id nếu cần =====
    if ($result['detected'] && !$result['group_id']) {
        $result['group_id'] = md5($phoneHash . date('Y-m'));
    }

    // Cap risk score at 100
    $riskScore = min(100, $riskScore);
    $result['risk_score'] = $riskScore;
    $result['risk_flags'] = $riskFlags;

    return $result;
}

/**
 * Update IP tracking table
 */
function updateIpTracking(PDO $pdo, string $ip, string $phoneHash, string $deviceFingerprint, int $riskScore, array $riskFlags): void
{
    $ipHash = md5($ip);

    // Upsert IP tracking
    $stmt = $pdo->prepare("
        INSERT INTO ip_tracking (
            ip_hash, ip, visit_count, session_count, lead_count,
            page_view_count, first_seen, last_seen, last_utm_source, last_page_url,
            risk_score, flags, updated_at
        ) VALUES (
            ?, ?, 1, 1, 1, 1, NOW(), NOW(), NULL, NULL,
            ?, ?, NOW()
        )
        ON DUPLICATE KEY UPDATE
            visit_count = visit_count + 1,
            session_count = session_count + 1,
            lead_count = lead_count + 1,
            page_view_count = page_view_count + 1,
            last_seen = NOW(),
            risk_score = GREATEST(risk_score, ?),
            flags = ?,
            updated_at = NOW()
    ");

    // Fetch existing flags trước (tránh nested query conflict)
    $existingFlagsJson = '[]';
    try {
        $flagStmt = $pdo->prepare("SELECT flags FROM ip_tracking WHERE ip_hash = ?");
        $flagStmt->execute([$ipHash]);
        $existingFlagsJson = $flagStmt->fetchColumn() ?: '[]';
    } catch (Throwable $e) {
        $existingFlagsJson = '[]';
    }

    $flagsJson = json_encode(array_unique(array_merge(
        json_decode($existingFlagsJson, true) ?: [],
        $riskFlags
    )));

    $stmt->execute([$ipHash, $ip, $riskScore, $flagsJson, $riskScore, $flagsJson]);
}

/**
 * Levenshtein distance for fuzzy phone matching
 * Đổi tên để tránh redeclare với built-in PHP function
 */
function apex_levenshtein(string $s1, string $s2): int {
    $len1 = strlen($s1);
    $len2 = strlen($s2);
    $matrix = [];

    for ($i = 0; $i <= $len1; $i++) $matrix[$i][0] = $i;
    for ($j = 0; $j <= $len2; $j++) $matrix[0][$j] = $j;

    for ($i = 1; $i <= $len1; $i++) {
        for ($j = 1; $j <= $len2; $j++) {
            $cost = $s1[$i-1] === $s2[$j-1] ? 0 : 1;
            $matrix[$i][$j] = min(
                $matrix[$i-1][$j] + 1,
                $matrix[$i][$j-1] + 1,
                $matrix[$i-1][$j-1] + $cost
            );
        }
    }

    return $matrix[$len1][$len2];
}
