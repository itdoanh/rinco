<?php
/**
 * Lead-status.php TEST - simulate same loading chain
 */
error_reporting(E_ALL);
ini_set('display_errors', '0');
ini_set('html_errors', '0');

set_exception_handler(function (Throwable $e) {
    error_log('[test-status] FATAL: ' . $e->getMessage() . ' | ' . $e->getFile() . ':' . $e->getLine());
    while (ob_get_level()) ob_end_clean();
    header('Content-Type: application/json', true, 500);
    echo json_encode([
        'ok' => false,
        'fatal' => $e->getMessage(),
        'file' => basename($e->getFile()),
        'line' => $e->getLine(),
    ]);
    exit;
});

ob_start();
require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/../admin/includes/auth.php';
require_once __DIR__ . '/../admin/includes/db.php';
require_once __DIR__ . '/../includes/cache.php';
require_once __DIR__ . '/lead-helpers.php';
ob_end_clean();

// Check state
$result = [
    'session_name' => session_name(),
    'session_id' => session_id(),
    'apex_admin_id' => $_SESSION['apex_admin_id'] ?? null,
    'apex_admin_user' => $_SESSION['apex_admin_user'] ?? null,
    'auth_result' => apex_admin_authenticated(),
    'apex_admin_log_exists' => function_exists('apex_admin_log'),
    'ApexCache_exists' => class_exists('ApexCache'),
    'lead_db_exists' => function_exists('lead_db'),
    'display_errors' => ini_get('display_errors'),
];

header('Content-Type: application/json');
echo json_encode($result, JSON_INVALID_UTF8_IGNORE);
