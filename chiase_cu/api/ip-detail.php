<?php
/**
 * API: IP Detail
 *
 * GET /api/ip-detail.php?ip_hash=abc123
 * Trả về chi tiết theo dõi của 1 IP
 */

declare(strict_types=1);

require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/lead-helpers.php';

header('Content-Type: application/json; charset=utf-8');

// Auth check
if (!apex_admin_require()) {
    http_response_code(401);
    echo json_encode(['ok' => false, 'error' => 'unauthorized']);
    exit;
}

try {
    $ipHash = trim($_GET['ip_hash'] ?? '');

    if ($ipHash === '' || strlen($ipHash) !== 32) {
        http_response_code(400);
        echo json_encode(['ok' => false, 'error' => 'invalid_ip_hash']);
        exit;
    }

    $pdo = lead_db();

    // IP info
    $stmt = $pdo->prepare("SELECT * FROM ip_tracking WHERE ip_hash = ?");
    $stmt->execute([$ipHash]);
    $ipInfo = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$ipInfo) {
        http_response_code(404);
        echo json_encode(['ok' => false, 'error' => 'ip_not_found']);
        exit;
    }

    // Sessions
    $stmt = $pdo->prepare("
        SELECT s.*, l.id AS lead_id, l.name AS lead_name, l.form_type, l.status AS lead_status
        FROM visitor_sessions s
        LEFT JOIN leads l ON l.session_id = s.session_id
        WHERE s.ip_hash = ?
        ORDER BY s.first_visit DESC
        LIMIT 20
    ");
    $stmt->execute([$ipHash]);
    $sessions = $stmt->fetchAll();

    // Leads
    $stmt = $pdo->prepare("
        SELECT id, name, phone, form_type, status, channel,
               utm_source, utm_medium, utm_campaign,
               created_at, contacted_at, converted_at
        FROM leads
        WHERE ip_hash = ?
        ORDER BY id DESC
        LIMIT 50
    ");
    $stmt->execute([$ipHash]);
    $leads = $stmt->fetchAll();

    // Sanitize - ẩn raw IP
    $ipInfo['ip_display'] = substr($ipInfo['ip'] ?? '', 0, 3) . '***';
    unset($ipInfo['ip']);

    // User journey
    $stmt = $pdo->prepare("
        SELECT * FROM user_journeys
        WHERE ip_hash = ?
        ORDER BY first_visit DESC
        LIMIT 10
    ");
    $stmt->execute([$ipHash]);
    $journeys = $stmt->fetchAll();

    // Parse flags
    $ipInfo['flags'] = json_decode($ipInfo['flags'] ?? '[]', true);

    // Risk assessment
    $riskLevel = 'low';
    $riskColor = '#10B981';
    if ($ipInfo['risk_score'] >= 70) {
        $riskLevel = 'high';
        $riskColor = '#DC2626';
    } elseif ($ipInfo['risk_score'] >= 40) {
        $riskLevel = 'medium';
        $riskColor = '#F59E0B';
    }

    echo json_encode([
        'ok' => true,
        'ip' => $ipInfo,
        'sessions' => $sessions,
        'leads' => $leads,
        'journeys' => $journeys,
        'risk' => [
            'level' => $riskLevel,
            'color' => $riskColor,
            'score' => (int)$ipInfo['risk_score'],
        ],
        'stats' => [
            'total_sessions' => (int)$ipInfo['session_count'],
            'total_leads' => (int)$ipInfo['lead_count'],
            'total_pageviews' => (int)$ipInfo['page_view_count'],
            'avg_duration' => (int)$ipInfo['avg_duration'],
            'first_seen' => $ipInfo['first_seen'],
            'last_seen' => $ipInfo['last_seen'],
        ],
    ], JSON_INVALID_UTF8_IGNORE);

} catch (Throwable $e) {
    lead_log_error('IP_DETAIL_ERROR', 'server', $e->getMessage(), $e->getTraceAsString(), null, lead_client_ip());

    http_response_code(500);
    echo json_encode(['ok' => false, 'error' => 'server_error']);
}
