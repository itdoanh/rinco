<?php
/**
 * APEX Admin - Database helper
 * Singleton PDO connection + utility helpers
 */

require_once __DIR__ . '/../../includes/config.php';

function apex_db(): PDO {
    static $pdo = null;
    if ($pdo === null) {
        $dsn = 'mysql:host=' . DB_HOST . ';dbname=' . DB_NAME . ';charset=' . DB_CHARSET;
        try {
            $pdo = new PDO($dsn, DB_USER, DB_PASS, [
                PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
                PDO::ATTR_EMULATE_PREPARES   => false,
                PDO::ATTR_PERSISTENT         => false,
                PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
                PDO::MYSQL_ATTR_INIT_COMMAND => "SET NAMES utf8mb4, time_zone = '+07:00'",
            ]);
        } catch (Throwable $e) {
            apex_admin_die('Database connection failed: ' . $e->getMessage());
        }
    }
    return $pdo;
}

function apex_admin_die(string $msg): void {
    if (DEBUG_MODE) {
        die('<pre style="font-family:monospace;padding:20px;background:#fee;border:1px solid #fbb;color:#900">' . htmlspecialchars($msg) . '</pre>');
    }
    http_response_code(500);
    die('<h1 style="font-family:sans-serif;color:#666;text-align:center;margin-top:100px">Service unavailable</h1>');
}

function h(?string $s): string {
    return htmlspecialchars($s ?? '', ENT_QUOTES, 'UTF-8');
}

function apex_admin_log(string $action, array $context = []): void {
    // Minimal audit log - silent fail (don't break admin UX)
    $dir = sys_get_temp_dir() . '/apex_admin_log/';
    if (!is_dir($dir)) @mkdir($dir, 0700, true);
    $entry = [
        'time'    => date('c'),
        'ip'      => $_SERVER['REMOTE_ADDR'] ?? '',
        'ua'      => $_SERVER['HTTP_USER_AGENT'] ?? '',
        'user'    => $_SESSION['apex_admin'] ?? null,
        'action'  => $action,
        'context' => $context,
    ];
    @file_put_contents($dir . date('Y-m-d') . '.jsonl', json_encode($entry) . "\n", FILE_APPEND | LOCK_EX);
}

function apex_format_duration(?int $sec): string {
    $sec = (int)$sec;
    if ($sec <= 0) return '0s';
    if ($sec < 60) return $sec . 's';
    if ($sec < 3600) return floor($sec / 60) . 'm ' . ($sec % 60) . 's';
    return floor($sec / 3600) . 'h ' . floor(($sec % 3600) / 60) . 'm';
}

function apex_format_number(int $n): string {
    if ($n >= 1000000) return round($n / 1000000, 1) . 'M';
    if ($n >= 1000) return number_format($n);
    return (string)$n;
}

function apex_time_ago(string $datetime): string {
    $ts = strtotime($datetime);
    if (!$ts) return $datetime;
    $diff = time() - $ts;
    if ($diff < 60)        return $diff . 's ago';
    if ($diff < 3600)      return floor($diff / 60) . 'm ago';
    if ($diff < 86400)     return floor($diff / 3600) . 'h ago';
    if ($diff < 86400 * 7) return floor($diff / 86400) . 'd ago';
    return date('d/m/Y', $ts);
}

function apex_percent($a, $b, int $decimals = 1): string {
    if ($b == 0) return '0%';
    return round(($a / $b) * 100, $decimals) . '%';
}
