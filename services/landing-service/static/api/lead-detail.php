<?php
/**
 * API: Lead Detail - Full tracking data (OPTIMIZED)
 */

// Suppress HTML errors for JSON response
ini_set('display_errors', 0);

$id = (int)($_GET['id'] ?? 0);

require_once __DIR__ . '/../includes/db.php';
require_once __DIR__ . '/../includes/auth.php';

header('Content-Type: application/json; charset=utf-8');

if (!apex_admin_authenticated()) {
    http_response_code(401);
    echo json_encode(['error' => 'Unauthorized']);
    exit;
}

if (!$id) {
    echo json_encode(['error' => 'Missing lead ID']);
    exit;
}

try {
    $pdo = apex_db();

    // Query 1: Lead info (indexed by id - PK)
    $stmt = $pdo->prepare("SELECT * FROM leads WHERE id = ? LIMIT 1");
    $stmt->execute([$id]);
    $lead = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$lead) {
        echo json_encode(['error' => 'Lead not found']);
        exit;
    }

    $leadIp = $lead['ip'] ?? '';
    $leadTime = $lead['created_at'] ?? null;

    // Query 2: Events for this IP (indexed by ip)
    $events = [];
    $totalEvents = 0;
    $pageViews = 0;
    if ($leadIp) {
        $stmt = $pdo->prepare("
            SELECT event_type, event_value, page_url, utm_source, utm_medium, utm_campaign, device_type, created_at
            FROM events
            WHERE ip = ?
            ORDER BY created_at DESC
            LIMIT 50
        ");
        $stmt->execute([$leadIp]);
        $events = $stmt->fetchAll(PDO::FETCH_ASSOC);
        $totalEvents = count($events);
        foreach ($events as $ev) {
            if (($ev['event_type'] ?? '') === 'page_view') $pageViews++;
        }
    }

    // Query 3: Session (indexed by ip)
    $session = null;
    if ($leadIp) {
        $stmt = $pdo->prepare("
            SELECT ip, device_type, browser_name, os_name, screen_res, language, country, city,
                   referrer, utm_source, utm_medium, utm_campaign, utm_content, fbclid, gclid,
                   first_visit, last_activity, page_views, total_duration, scroll_max
            FROM visitor_sessions
            WHERE ip = ?
            ORDER BY first_visit DESC
            LIMIT 1
        ");
        $stmt->execute([$leadIp]);
        $session = $stmt->fetch(PDO::FETCH_ASSOC) ?: null;
    }

    $response = [
        'ok' => true,
        'lead' => $lead,
        'events' => $events,
        'session' => $session,
        'stats' => [
            'total_events' => $totalEvents,
            'page_views' => $pageViews,
            'time_on_site' => $session ? ($session['total_duration'] ?? 0) : 0,
        ]
    ];

    echo json_encode($response);
} catch (Throwable $e) {
    http_response_code(500);
    echo json_encode(['error' => 'Server error']);
    error_log('lead-detail error: ' . $e->getMessage());
}