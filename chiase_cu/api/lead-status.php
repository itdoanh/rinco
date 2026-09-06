<?php
/**
 * API: Lead Status Update - STANDALONE (không require auth.php)
 * Chỉ verify bằng session cookie hoặc DB token.
 * POST /api/lead-status.php
 */

// ===== 1. TẮT TẤT CẢ ERROR OUTPUT =====
error_reporting(0);
ini_set('display_errors', '0');
ini_set('display_startup_errors', '0');
ini_set('html_errors', '0');
ini_set('log_errors', '1');

// ===== 2. REQUIRE TỐI THIỂU =====
// Chỉ load config để lấy DB constants
require_once __DIR__ . '/../includes/config.php';

// ===== 3. START SESSION =====
if (session_status() === PHP_SESSION_NONE) {
    @session_name('apex_admin_sid');
    @session_start();
}

// ===== 4. AUTH =====
$adminId = (int)($_SESSION['apex_admin_id'] ?? 0);
$adminUser = $_SESSION['apex_admin_user'] ?? '';

// ===== 5. DB CONNECTION =====
$pdo = null;
try {
    $dsn = 'mysql:host=' . DB_HOST . ';dbname=' . DB_NAME . ';charset=utf8mb4';
    $pdo = new PDO($dsn, DB_USER, DB_PASS, [
        PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_EMULATE_PREPARES => false,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
    ]);
} catch (Throwable $e) {
    http_response_code(500);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'db_connection_failed', 'msg' => $e->getMessage()]);
    exit;
}

// ===== 6. INPUT =====
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'method_not_allowed']);
    exit;
}

if (!$adminId) {
    http_response_code(401);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'unauthorized', 'admin_id' => $adminId]);
    exit;
}

// ===== 7. PARSE JSON BODY =====
$raw = file_get_contents('php://input');
$data = json_decode($raw, true);

if (!$data || !isset($data['id']) || !isset($data['status'])) {
    http_response_code(400);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'missing_params']);
    exit;
}

$id = (int)$data['id'];
$status = trim($data['status']);
$notes = trim($data['notes'] ?? '');

$allowedStatuses = ['new', 'contacted', 'qualified', 'converted', 'invalid', 'duplicate'];
if (!$id || !in_array($status, $allowedStatuses, true)) {
    http_response_code(400);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'invalid_status', 'valid' => $allowedStatuses]);
    exit;
}

// ===== 8. UPDATE =====
try {
    // Get current lead
    $stmt = $pdo->prepare("SELECT id, status FROM leads WHERE id = ?");
    $stmt->execute([$id]);
    $lead = $stmt->fetch();

    if (!$lead) {
        http_response_code(404);
        header('Content-Type: application/json');
        echo json_encode(['ok' => false, 'error' => 'lead_not_found', 'id' => $id]);
        exit;
    }

    $oldStatus = $lead['status'];
    $updates = ['status = ?'];
    $params = [$status];

    if ($status === 'contacted') $updates[] = 'contacted_at = NOW()';
    if ($status === 'converted') $updates[] = 'converted_at = NOW()';

    if ($notes !== '') {
        $updates[] = 'notes = CONCAT(IFNULL(notes, ""), ?, "\n")';
        $params[] = date('d/m/Y H:i') . ' [' . $adminUser . '] ' . $notes;
    }

    $params[] = $id;
    $sql = "UPDATE leads SET " . implode(', ', $updates) . " WHERE id = ?";
    $pdo->prepare($sql)->execute($params);

    // ===== 9. LOG =====
    try {
        $logDir = sys_get_temp_dir() . '/apex_admin_log/';
        if (!is_dir($logDir)) @mkdir($logDir, 0700, true);
        $entry = [
            'time' => date('c'),
            'user' => $adminUser,
            'action' => 'lead_status_update',
            'lead_id' => $id,
            'old' => $oldStatus,
            'new' => $status,
        ];
        @file_put_contents($logDir . date('Y-m-d') . '.jsonl', json_encode($entry) . "\n", FILE_APPEND | LOCK_EX);
    } catch (Throwable $_) {}

    // ===== 10. RESPONSE =====
    header('Content-Type: application/json');
    echo json_encode([
        'ok' => true,
        'id' => $id,
        'status' => $status,
        'previous_status' => $oldStatus,
    ]);

} catch (Throwable $e) {
    error_log('[lead-status] Error: ' . $e->getMessage() . ' at ' . $e->getFile() . ':' . $e->getLine());
    http_response_code(500);
    header('Content-Type: application/json');
    echo json_encode(['ok' => false, 'error' => 'update_failed', 'msg' => $e->getMessage()]);
}
