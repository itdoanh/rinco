<?php
error_reporting(0);

require_once __DIR__ . '/../admin/includes/db.php';
require_once __DIR__ . '/../admin/includes/auth.php';

if (!apex_admin_authenticated()) {
    http_response_code(401);
    echo json_encode(['error' => 'unauthorized']);
    exit;
}

// SSE headers
header('Content-Type: text/event-stream');
header('Cache-Control: no-cache');
header('Connection: keep-alive');
header('X-Accel-Buffering: no');

$lastId = (int)($_GET['after_id'] ?? 0);
$pdo = apex_db();

// Send current leads
try {
    $stmt = $pdo->prepare("SELECT id, name, phone, form_type, created_at, ip, device_type FROM leads WHERE id > ? ORDER BY id DESC LIMIT 30");
    $stmt->execute([$lastId]);
    $leads = $stmt->fetchAll();
    
    echo "data: " . json_encode(['leads' => $leads, 'ts' => time()]) . "\n\n";
    @ob_flush();
    @flush();
    
    if ($leads) $lastId = (int)$leads[0]['id'];
} catch (Throwable $e) {
    echo "data: " . json_encode(['error' => 'db']) . "\n\n";
    @ob_flush();
    @flush();
    exit;
}

// Loop for new leads - skip initial flush above
$timeout = 45;
$start = time();
$pollInterval = 3;
$lastCheck = 0;
while (time() - $start < $timeout) {
    if (connection_aborted()) break;

    try {
        // Lightweight MAX(id) poll every 3s
        $stmt = $pdo->query("SELECT MAX(id) as m FROM leads");
        $maxId = (int)$stmt->fetch()['m'];

        if ($maxId > $lastId) {
            $stmt = $pdo->prepare("SELECT id, name, phone, form_type, created_at, ip, device_type FROM leads WHERE id > ? ORDER BY id ASC LIMIT 30");
            $stmt->execute([$lastId]);
            $newLeads = $stmt->fetchAll();

            echo "data: " . json_encode(['leads' => $newLeads, 'ts' => time()]) . "\n\n";
            @ob_flush();
            @flush();

            if ($newLeads) $lastId = (int)$newLeads[count($newLeads) - 1]['id'];
        }
    } catch (Throwable $e) {
        error_log('SSE error: ' . $e->getMessage());
        break;
    }

    echo ": h\n\n";
    @ob_flush();
    @flush();

    // Sleep in small chunks to detect client disconnect faster
    for ($i = 0; $i < $pollInterval; $i++) {
        if (connection_aborted()) break 2;
        sleep(1);
    }
}
