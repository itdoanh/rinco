<?php
/**
 * APEX FINTECH - API: LEAD ERRORS (Admin xem log lỗi)
 *
 * GET /api/lead-errors.php?range=24h|7d|30d&stage=db|client|server|network&code=invalid_phone&limit=50&offset=0
 *
 * Yêu cầu: Đăng nhập admin (Basic Auth qua session cookie hoặc direct admin login)
 *
 * Response:
 *   {
 *     "ok": true,
 *     "summary": {
 *       "total": 123,
 *       "by_code": [{ "code": "...", "count": 10, "recovered": 5 }, ...],
 *       "by_stage": [{ "stage": "db", "count": 50 }, ...]
 *     },
 *     "errors": [
 *       {
 *         "id": 1,
 *         "error_code": "invalid_phone",
 *         "error_stage": "server",
 *         "error_message": "...",
 *         "error_detail": {...},
 *         "idempotency_key": "uuid",
 *         "form_type": "hero",
 *         "ip": "1.2.3.4",
 *         "recovered": 0,
 *         "retry_count": 0,
 *         "created_at": "2026-09-04 10:00:00"
 *       }
 *     ]
 *   }
 */

require_once __DIR__ . '/lead-helpers.php';

// === AUTH: yêu cầu đăng nhập admin ===
// Chấp nhận 2 cách:
//   1) Có session admin (cookie PHPSESSID hợp lệ - qua /admin/login.php)
//   2) Bearer token = admin password (cho cron job / external monitoring)
$authed = false;

// Cách 1: session
if (isset($_COOKIE['apex_admin_sid']) && !empty($_COOKIE['apex_admin_sid'])) {
    try {
        $pdo = lead_db();
        $stmt = $pdo->prepare("
            SELECT a.id, a.username FROM admins a
            INNER JOIN admin_remember_tokens t ON t.admin_id = a.id
            WHERE t.token_hash = ? AND t.expires_at > NOW()
            LIMIT 1
        ");
        $stmt->execute([hash('sha256', $_COOKIE['apex_admin_sid'])]);
        if ($stmt->fetch()) $authed = true;
    } catch (Throwable $e) {}
}

// Cách 2: Bearer token
if (!$authed) {
    $authHeader = $_SERVER['HTTP_AUTHORIZATION'] ?? '';
    if (preg_match('/Bearer\s+(.+)/i', $authHeader, $m)) {
        $token = trim($m[1]);
        if (defined('TRACK_SECRET') && TRACK_SECRET !== '' && hash_equals(TRACK_SECRET, $token)) {
            $authed = true;
        }
        // Fallback: hash của admin password (ít an toàn hơn, nhưng tiện)
        if (!$authed && defined('ADMIN_PASS_HASH')) {
            if (password_verify($token, ADMIN_PASS_HASH)) $authed = true;
        }
    }
}

if (!$authed) {
    header('Content-Type: application/json; charset=utf-8');
    http_response_code(401);
    echo json_encode(['ok' => false, 'error' => 'unauthorized', 'detail' => 'Cần đăng nhập admin']);
    exit;
}

header('Content-Type: application/json; charset=utf-8');

// === Parse params ===
$range = $_GET['range'] ?? '24h';
$stage = $_GET['stage'] ?? null;
$code  = $_GET['code'] ?? null;
$limit = min(200, max(1, (int)($_GET['limit'] ?? 50)));
$offset = max(0, (int)($_GET['offset'] ?? 0));

$from = match($range) {
    '1h'  => date('Y-m-d H:i:s', time() - 3600),
    '24h' => date('Y-m-d H:i:s', time() - 86400),
    '7d'  => date('Y-m-d H:i:s', time() - 7 * 86400),
    '30d' => date('Y-m-d H:i:s', time() - 30 * 86400),
    'all' => '2000-01-01 00:00:00',
    default => date('Y-m-d H:i:s', time() - 86400),
};

$where = ['created_at >= ?'];
$params = [$from];
if ($stage) { $where[] = 'error_stage = ?'; $params[] = $stage; }
if ($code)  { $where[] = 'error_code = ?';  $params[] = $code; }
$whereSql = implode(' AND ', $where);

try {
    $pdo = lead_db();

    // === SUMMARY ===
    $total = (int)$pdo->prepare("SELECT COUNT(*) FROM lead_errors WHERE $whereSql")->execute($params) ? // dummy
        $pdo->prepare("SELECT COUNT(*) FROM lead_errors WHERE $whereSql")->execute($params) : 0;
    $stmt = $pdo->prepare("SELECT COUNT(*) FROM lead_errors WHERE $whereSql");
    $stmt->execute($params);
    $total = (int)$stmt->fetchColumn();

    $byCode = $pdo->prepare("
        SELECT error_code, error_stage, COUNT(*) as cnt,
               SUM(CASE WHEN recovered=1 THEN 1 ELSE 0 END) as recovered
        FROM lead_errors
        WHERE $whereSql
        GROUP BY error_code, error_stage
        ORDER BY cnt DESC LIMIT 20
    ");
    $byCode->execute($params);

    $byStage = $pdo->prepare("
        SELECT error_stage, COUNT(*) as cnt
        FROM lead_errors
        WHERE $whereSql
        GROUP BY error_stage
    ");
    $byStage->execute($params);

    // === ERRORS LIST ===
    $stmt = $pdo->prepare("
        SELECT id, error_code, error_stage, error_message, error_detail,
               http_status, idempotency_key, form_type, ip, retry_count,
               recovered, created_at, phone_hash
        FROM lead_errors
        WHERE $whereSql
        ORDER BY id DESC
        LIMIT $limit OFFSET $offset
    ");
    $stmt->execute($params);
    $errors = $stmt->fetchAll(PDO::FETCH_ASSOC);

    // Parse error_detail JSON
    foreach ($errors as &$err) {
        if (!empty($err['error_detail'])) {
            $decoded = json_decode($err['error_detail'], true);
            if (json_last_error() === JSON_ERROR_NONE) {
                $err['error_detail'] = $decoded;
            }
        }
        // Mask phone (chỉ giữ 3 số đầu)
        $err['id'] = (int)$err['id'];
        $err['http_status'] = $err['http_status'] ? (int)$err['http_status'] : null;
        $err['retry_count'] = (int)$err['retry_count'];
        $err['recovered'] = (int)$err['recovered'];
    }

    echo json_encode([
        'ok' => true,
        'range' => $range,
        'from' => $from,
        'summary' => [
            'total' => $total,
            'by_code' => $byCode->fetchAll(PDO::FETCH_ASSOC),
            'by_stage' => $byStage->fetchAll(PDO::FETCH_ASSOC)
        ],
        'pagination' => [
            'limit' => $limit,
            'offset' => $offset,
            'returned' => count($errors)
        ],
        'errors' => $errors
    ], JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);

} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode([
        'ok' => false,
        'error' => 'server_error',
        'detail' => $e->getMessage()
    ]);
}
