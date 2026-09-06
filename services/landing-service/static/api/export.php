<?php
/**
 * APEX Admin API - CSV Export
 * GET /admin/api/export.php?type=leads&from=...&to=...
 * GET /admin/api/export.php?type=events&session=...
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/../includes/auth.php';

$user = apex_admin_require();

$type = $_GET['type'] ?? '';
$range = apex_parse_date_range();

header('Content-Type: text/csv; charset=utf-8');
header('Content-Disposition: attachment; filename="apex-' . $type . '-' . date('Ymd-His') . '.csv"');
header('Cache-Control: no-store');

$out = fopen('php://output', 'w');
// BOM for Excel UTF-8
fwrite($out, "\xEF\xBB\xBF");

if ($type === 'leads') {
    fputcsv($out, ['ID', 'Tên', 'SĐT', 'Form', 'Source', 'Medium', 'Campaign', 'Referrer', 'Device', 'IP', 'Page URL', 'Thời gian']);
    $rows = apex_query_leads($range)['rows'] ?? [];
    foreach ($rows as $r) {
        fputcsv($out, [
            $r['id'], $r['name'], $r['phone'], $r['form_type'],
            $r['source'], $r['medium'], $r['campaign'], $r['referrer'],
            $r['device_type'], $r['ip'], $r['page_url'], $r['created_at']
        ]);
    }
} elseif ($type === 'events') {
    fputcsv($out, ['ID', 'Session', 'Type', 'Value', 'Page URL', 'UTM Source', 'UTM Medium', 'UTM Campaign', 'Device', 'Browser', 'OS', 'IP', 'Time']);
    $session = $_GET['session'] ?? null;
    $rows = apex_query_recent_events(1000, null, $session);
    foreach ($rows as $r) {
        fputcsv($out, [
            $r['id'], $r['session_id'], $r['event_type'], $r['event_value'],
            $r['page_url'], $r['utm_source'], $r['utm_medium'], $r['utm_campaign'],
            $r['device_type'], $r['browser_name'], $r['os_name'], $r['ip'], $r['created_at']
        ]);
    }
} else {
    fputcsv($out, ['error', 'invalid_type']);
}

fclose($out);
apex_admin_log('export_csv', ['type' => $type]);
