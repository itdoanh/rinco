<?php
/**
 * Xem log track.php để debug
 */
error_reporting(E_ALL);
ini_set('display_errors', '1');

header('Content-Type: text/plain; charset=utf-8');

$logFile = '/tmp/apex_track_debug.log';

echo "=== TRACK LOG ===\n\n";

if (!file_exists($logFile)) {
    echo "Log file not exists: {$logFile}\n";
    echo "→ Tracker có thể chưa chạy hoặc isBot luôn = 0 (không skip)\n";
    
    // List các file log có thể có
    echo "\nTìm các file log khác:\n";
    $patterns = [
        '/tmp/apex_*.log',
        '/tmp/*.log',
        '/var/log/apache2/*.log',
        '/var/log/httpd/*.log',
        sys_get_temp_dir() . '/apex_*',
    ];
    foreach ($patterns as $p) {
        $files = glob($p);
        if ($files) {
            foreach ($files as $f) {
                $size = filesize($f);
                echo "  - {$f} ({$size} bytes)\n";
            }
        }
    }
} else {
    echo "File: {$logFile}\n";
    echo "Size: " . filesize($logFile) . " bytes\n";
    echo "Modified: " . date('Y-m-d H:i:s', filemtime($logFile)) . "\n\n";
    
    echo "--- NỘI DUNG (50 dòng cuối) ---\n";
    $lines = file($logFile);
    $last = array_slice($lines, -50);
    foreach ($last as $l) {
        echo $l;
    }
}
