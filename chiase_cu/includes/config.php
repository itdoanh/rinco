<?php
/**
 * APEX FINTECH - CONFIG
 * Đọc từ biến môi trường nếu có (cho Docker), nếu không dùng giá trị mặc định.
 * ⚠️ TRÊN CPANEL: Sửa các giá trị bên dưới cho phù hợp, hoặc set biến môi trường.
 */

// Force display errors (chỉ để debug, xóa sau khi xong)
// Tạm thời bật để xem lỗi 500 trên hosting
if (!defined('APEX_CONFIG_LOADED')) {
    define('APEX_CONFIG_LOADED', true);
    // Only enable display_errors for HTML pages (not for API endpoints)
    // API endpoints will explicitly disable display_errors in their own files
    @ini_set('display_startup_errors', '1');
    @error_reporting(E_ALL);
    // display_errors is set below based on DEBUG_MODE
}

// Admin panel guard
if (!defined('APEX_ADMIN')) {
    // Only define if not already defined by admin pages
    define('APEX_ADMIN', true);
}

// ========== DATABASE ==========
// ⚠️ TRÊN CPANEL: Sửa 4 dòng dưới cho đúng DB hosting
define('DB_HOST', getenv('APEX_DB_HOST') ?: 'localhost');
define('DB_NAME', getenv('APEX_DB_NAME') ?: 'hanghoap_apex');
define('DB_USER', getenv('APEX_DB_USER') ?: 'hanghoap_apex');
define('DB_PASS', getenv('APEX_DB_PASS') ?: '10032022Lam@');
define('DB_CHARSET', 'utf8mb4');

// ========== ADMIN DASHBOARD ==========
define('ADMIN_USER', getenv('APEX_ADMIN_USER') ?: 'admin');
// Mặc định bcrypt hash cho "apexadmin2026" - ĐỔI NGAY sau khi đăng nhập đầu tiên
define('ADMIN_PASS_HASH', getenv('APEX_ADMIN_PASS_HASH') ?: '$2y$10$8K1p/a0dRhX3oSZQAk5O9.J5mABqdxKKk7PzFxlP1fH8pBvTLrhOa');

// ========== TRACKING SECRET ==========
define('TRACK_SECRET', getenv('APEX_TRACK_SECRET') ?: '');

// ========== BRAND ==========
define('BRAND_NAME', 'APEX Fintech');
define('BRAND_DOMAIN', 'https://hanghoaphaisinh.net');
define('BRAND_EMAIL', 'contact@hanghoaphaisinh.net');

// ========== BASE PATH ==========
// Đường dẫn gốc của ứng dụng (thay đổi nếu deploy vào thư mục khác)
// Local dev: '' (rỗng - serve từ root, vd: http://localhost:8080/)
// Production cPanel '/chiase': '/chiase' (vd: https://hanghoaphaisinh.net/chiase/)
define('BASE_PATH', '/chiase');

// ========== RATE LIMIT ==========
// Per IP per hour (lead form submissions) - 0 = disable
define('LEAD_RATE_PER_HOUR', 0);
// Per IP per minute (page tracking events)  
define('TRACK_RATE_PER_MIN', 200);

// ========== DEBUG ==========
define('DEBUG_MODE', getenv('APEX_DEBUG') === '1');

// ========== TIMEZONE ==========
date_default_timezone_set('Asia/Ho_Chi_Minh');

// ========== ERROR REPORTING ==========
if (DEBUG_MODE) {
    error_reporting(E_ALL);
    ini_set('display_errors', '1');
} else {
    error_reporting(E_ERROR | E_WARNING | E_PARSE);
    ini_set('display_errors', '0');
    ini_set('log_errors', '1');
}

// ========== SESSION ==========
// Luôn start session nếu chưa có
if (session_status() === PHP_SESSION_NONE) {
    // Nếu có apex_session_start (admin context đã load auth.php) thì dùng nó
    if (function_exists('apex_session_start')) {
        apex_session_start();
    } else {
        @session_start();
    }
}

// ========== ADMIN AUTH HELPER ==========
// Admin auth helper được định nghĩa trong admin/includes/auth.php
// (Stub đã xóa vì xung đột fatal error với admin/includes/auth.php)
