<?php
/**
 * APEX FINTECH - Lead helpers (chia sẻ giữa các endpoint lead)
 *
 * Cung cấp:
 *   - Client IP detection (ưu tiên Cloudflare)
 *   - Phone hash (MD5) để tracking mà không lộ số
 *   - Idempotency key validation
 *   - Error logging (ghi vào lead_errors)
 *   - Rate limit (per IP)
 *   - DB connection
 *
 * Mọi endpoint lead (lead.php, lead-ack.php, lead-retry.php) đều include file này.
 */

require_once __DIR__ . '/../includes/config.php';

if (!defined('APEX_LEAD_HELPERS')) {
    define('APEX_LEAD_HELPERS', true);
}

/**
 * Lấy IP client (ưu tiên Cloudflare > X-Forwarded-For > REMOTE_ADDR)
 */
function lead_client_ip(): string {
    foreach (['HTTP_CF_CONNECTING_IP', 'HTTP_X_FORWARDED_FOR', 'REMOTE_ADDR'] as $k) {
        if (!empty($_SERVER[$k])) {
            $ip = trim(explode(',', $_SERVER[$k])[0]);
            if (filter_var($ip, FILTER_VALIDATE_IP)) return $ip;
        }
    }
    return '0.0.0.0';
}

/**
 * Hash phone để tracking không lộ PII.
 * MD5 là đủ vì chỉ dùng để dedup/group, không phải security.
 */
function lead_phone_hash(string $phone): string {
    $normalized = preg_replace('/\D/', '', $phone);
    return md5('apex_v2_' . $normalized);
}

/**
 * Validate UUID v4 format (idempotency_key)
 */
function lead_is_valid_uuid(string $key): bool {
    return (bool)preg_match('/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i', $key);
}

/**
 * DB connection (dùng chung cho mọi endpoint lead)
 */
function lead_db(): PDO {
    static $pdo = null;
    if ($pdo) return $pdo;
    $dsn = 'mysql:host=' . DB_HOST . ';dbname=' . DB_NAME . ';charset=' . DB_CHARSET;
    $pdo = new PDO($dsn, DB_USER, DB_PASS, [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_EMULATE_PREPARES => false,
        PDO::ATTR_PERSISTENT => false,
        PDO::MYSQL_ATTR_INIT_COMMAND => "SET NAMES utf8mb4",
    ]);
    return $pdo;
}

/**
 * Rate limit per IP (in-memory cache APCu, fallback DB).
 * Bucket 'lead_submit': tối đa LEAD_RATE_PER_HOUR lần/giờ.
 * Bucket 'lead_retry': tối đa 200 lần/phút.
 */
function lead_rate_limit(string $ip, string $bucket, int $max): bool {
    if (LEAD_RATE_PER_HOUR <= 0 && $bucket === 'lead_submit') return true; // disabled

    $windowSec = ($bucket === 'lead_submit') ? 3600 : 60;

    if (function_exists('apcu_enabled') && apcu_enabled()) {
        $key = "apex_rl_{$bucket}_" . md5($ip);
        $current = apcu_inc($key, 1, $success, $windowSec);
        if (!$success) {
            apcu_store($key, 1, $windowSec);
            $current = 1;
        }
        return $current <= $max;
    }

    try {
        $pdo = lead_db();
        $now = time();
        $reset = $now + $windowSec;
        $stmt = $pdo->prepare("
            INSERT INTO rate_limits (ip, bucket, count, reset_at)
            VALUES (?, ?, 1, ?)
            ON DUPLICATE KEY UPDATE
                count = IF(reset_at < ?, 1, count + 1),
                reset_at = IF(reset_at < ?, VALUES(reset_at), reset_at)
        ");
        $stmt->execute([$ip, $bucket, $reset, $now, $reset]);
        $count = (int)$pdo->query("SELECT count FROM rate_limits WHERE ip='" . addslashes($ip) . "' AND bucket='" . addslashes($bucket) . "'")->fetchColumn();
        return $count <= $max;
    } catch (Throwable $e) {
        // Fail open nếu DB lỗi (không block user thật)
        error_log('[APEX] rate_limit DB error: ' . $e->getMessage());
        return true;
    }
}

/**
 * Ghi log lỗi vào bảng lead_errors.
 * Tất cả lỗi (validation fail, network fail, DB error, ...) đều phải ghi log.
 *
 * @param string $code       Mã lỗi ngắn gọn (vd: 'invalid_phone', 'db_timeout')
 * @param string $stage      'client' | 'server' | 'db' | 'network'
 * @param string $msg        Mô tả ngắn
 * @param array  $detail     Chi tiết (stack trace, response body...)
 * @param array  $ctx        Context: idempotency_key, phone_hash, form_type, payload
 */
function lead_log_error(string $code, string $stage, string $msg, array $detail = [], array $ctx = []): void {
    try {
        $pdo = lead_db();
        $stmt = $pdo->prepare("
            INSERT INTO lead_errors
                (error_code, error_stage, error_message, error_detail, http_status,
                 idempotency_key, payload_json, phone_hash, form_type, ip, user_agent, retry_count, created_at)
            VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
        ");
        $stmt->execute([
            substr($code, 0, 50),
            substr($stage, 0, 30),
            substr($msg, 0, 65535),
            !empty($detail) ? substr(json_encode($detail, JSON_UNESCAPED_UNICODE), 0, 65535) : null,
            $ctx['http_status'] ?? null,
            $ctx['idempotency_key'] ?? null,
            !empty($ctx['payload']) ? substr(json_encode($ctx['payload'], JSON_UNESCAPED_UNICODE), 0, 65535) : null,
            $ctx['phone_hash'] ?? null,
            $ctx['form_type'] ?? null,
            lead_client_ip(),
            substr($_SERVER['HTTP_USER_AGENT'] ?? '', 0, 500),
            $ctx['retry_count'] ?? 0,
        ]);
    } catch (Throwable $e) {
        // Không throw - không để lỗi logging làm crash request
        error_log('[APEX] lead_log_error failed: ' . $e->getMessage() . ' | original: ' . $code . ' / ' . $msg);
    }
}

/**
 * Đánh dấu lỗi đã được recovered (sau khi retry thành công).
 */
function lead_mark_recovered(string $idempotencyKey): void {
    if (!lead_is_valid_uuid($idempotencyKey)) return;
    try {
        $pdo = lead_db();
        $stmt = $pdo->prepare("UPDATE lead_errors SET recovered=1 WHERE idempotency_key=? AND recovered=0");
        $stmt->execute([$idempotencyKey]);
    } catch (Throwable $e) {
        error_log('[APEX] lead_mark_recovered failed: ' . $e->getMessage());
    }
}

/**
 * Đọc & parse JSON body an toàn.
 * Trả về array (rỗng nếu parse fail).
 */
function lead_read_json(): array {
    $raw = file_get_contents('php://input');
    if (!$raw) return [];
    $data = json_decode($raw, true);
    return is_array($data) ? $data : [];
}

/**
 * Truncate string an toàn (UTF-8).
 */
function lead_substr(?string $s, int $len): ?string {
    if ($s === null) return null;
    return function_exists('mb_substr') ? mb_substr($s, 0, $len) : substr($s, 0, $len);
}

/**
 * Ghi CORS headers.
 */
function lead_cors(): void {
    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store');
    header('X-Content-Type-Options: nosniff');
    header('Access-Control-Allow-Origin: *');
    header('Access-Control-Allow-Methods: POST, OPTIONS');
    header('Access-Control-Allow-Headers: Content-Type, X-Requested-With, X-Idempotency-Key');
}

/**
 * Pre-flight CORS.
 */
function lead_handle_preflight(): bool {
    if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') {
        http_response_code(204);
        lead_cors();
        return true;
    }
    lead_cors();
    return false;
}
