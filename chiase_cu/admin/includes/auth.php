<?php
/**
 * APEX Admin - Auth guard
 * Yêu cầu login cho tất cả page trong /admin/
 * Argon2ID verification với rehash tự động
 * Hỗ trợ Remember Me token (persistent login)
 */

if (!defined('APEX_ADMIN')) {
    http_response_code(403);
    die('Forbidden');
}

require_once __DIR__ . '/db.php';

// Base path (subfolder trên cPanel) - dùng cho redirect
if (!defined('ADMIN_BASE')) {
    define('ADMIN_BASE', rtrim(BASE_PATH, '/') . '/admin');
}

/**
 * Khởi tạo session an toàn với thời gian sống dài
 * Để hỗ trợ remember me qua cookie
 */
function apex_session_start(): void {
    if (session_status() === PHP_SESSION_ACTIVE) return;

    $secure = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off')
           || (($_SERVER['SERVER_PORT'] ?? '') == '443');

    // Default lifetime: 7 ngày cho phép remember me flow
    $lifetime = 7 * 24 * 60 * 60;

    // PHP 7.3+ supports array syntax for session_set_cookie_params
    if (PHP_VERSION_ID >= 70300) {
        @session_set_cookie_params([
            'lifetime' => $lifetime,
            'path'     => '/',
            'domain'   => '',
            'secure'   => $secure,
            'httponly' => true,
            'samesite' => 'Lax',
        ]);
    } else {
        // PHP < 7.3 fallback
        @session_set_cookie_params($lifetime, '/', '', $secure, true);
    }

    @session_name('apex_admin_sid');

    // PHP 7.1+ supports options array for session_start
    if (PHP_VERSION_ID >= 70100) {
        @session_start([
            'use_strict_mode'   => true,
            'use_only_cookies'  => true,
        ]);
    } else {
        @session_start();
    }
}

apex_session_start();

header('X-Frame-Options: DENY');
header('X-Content-Type-Options: nosniff');
header('Referrer-Policy: same-origin');

/**
 * Lấy base URL admin đầy đủ, ví dụ: /chiase/admin
 */
function apex_admin_base(): string {
    return ADMIN_BASE;
}

/**
 * URL tuyệt đối trong admin (kèm base path)
 * Ví dụ: apex_admin_url('login.php') => '/chiase/admin/login.php'
 */
function apex_admin_url(string $path = ''): string {
    return ADMIN_BASE . ($path ? '/' . ltrim($path, '/') : '');
}

/**
 * URL cho static assets (images, css, js) - kèm BASE_PATH
 * Ví dụ: apex_asset_url('assets/img/logo.webp') => '/chiase/assets/img/logo.webp'
 */
function apex_asset_url(string $path): string {
    $base = rtrim(BASE_PATH ?? '', '/');
    return $base . '/' . ltrim($path, '/');
}

/**
 * Redirect về admin dashboard (kèm BASE_PATH)
 */
function apex_admin_redirect(string $path = ''): void {
    header('Location: ' . apex_admin_url($path));
    exit;
}

/**
 * Kiểm tra đã login chưa (cả session lẫn remember token)
 */
function apex_admin_authenticated(): bool {
    // 1. Session thường
    if (!empty($_SESSION['apex_admin_id']) && !empty($_SESSION['apex_admin_user'])) {
        // Idle timeout 8 giờ (dài hơn để dùng trong ngày)
        if (isset($_SESSION['apex_admin_last_seen']) && (time() - $_SESSION['apex_admin_last_seen']) > 28800) {
            apex_admin_logout();
            return false;
        }
        $_SESSION['apex_admin_last_seen'] = time();
        return true;
    }

    // 2. Remember token cookie
    $token = $_COOKIE['apex_remember'] ?? '';
    if ($token && preg_match('/^[a-f0-9]{64}$/', $token)) {
        return apex_admin_try_remember_token($token);
    }

    return false;
}

/**
 * Thử xác thực qua remember token
 */
function apex_admin_try_remember_token(string $token): bool {
    try {
        $pdo = apex_db();
        $hash = hash('sha256', $token);
        $stmt = $pdo->prepare("
            SELECT t.id AS token_id, t.admin_id, t.expires_at, a.username
            FROM admin_remember_tokens t
            JOIN admins a ON a.id = t.admin_id
            WHERE t.token_hash = ? AND t.expires_at > NOW()
            LIMIT 1
        ");
        $stmt->execute([$hash]);
        $row = $stmt->fetch();

        if (!$row) {
            // Token sai/hết hạn - xóa cookie
            setcookie('apex_remember', '', time() - 3600, '/', '', false, true);
            return false;
        }

        // Xác thực thành công - tái tạo session
        session_regenerate_id(true);
        $_SESSION['apex_admin_id']        = (int)$row['admin_id'];
        $_SESSION['apex_admin_user']      = $row['username'];
        $_SESSION['apex_admin_last_seen'] = time();
        $_SESSION['apex_admin_login_at']  = time();

        // Rotate token để chống replay attack
        apex_admin_issue_remember_token((int)$row['admin_id'], $token);

        return true;
    } catch (Throwable $_) {
        return false;
    }
}

/**
 * Phát hành remember token mới (lưu hash vào DB, set cookie raw token)
 */
function apex_admin_issue_remember_token(int $adminId, ?string $oldToken = null): bool {
    try {
        $pdo = apex_db();

        // Xóa token cũ nếu có
        if ($oldToken) {
            $oldHash = hash('sha256', $oldToken);
            $pdo->prepare("DELETE FROM admin_remember_tokens WHERE token_hash = ?")->execute([$oldHash]);
        }

        // Xóa tất cả token cũ của user (giữ tối đa 3 token)
        $pdo->prepare("
            DELETE FROM admin_remember_tokens
            WHERE admin_id = ?
              AND id NOT IN (
                SELECT id FROM (
                  SELECT id FROM admin_remember_tokens WHERE admin_id = ? ORDER BY issued_at DESC LIMIT 2
                ) t
              )
        ")->execute([$adminId, $adminId]);

        $rawToken = bin2hex(random_bytes(32));
        $hash = hash('sha256', $rawToken);
        $expires = date('Y-m-d H:i:s', time() + 30 * 86400); // 30 ngày

        $pdo->prepare("
            INSERT INTO admin_remember_tokens (admin_id, token_hash, issued_at, expires_at, user_agent, ip)
            VALUES (?, ?, NOW(), ?, ?, ?)
        ")->execute([
            $adminId,
            $hash,
            $expires,
            mb_substr($_SERVER['HTTP_USER_AGENT'] ?? '', 0, 255),
            $_SERVER['REMOTE_ADDR'] ?? ''
        ]);

        // Set cookie 30 ngày
        $secure = (!empty($_SERVER['HTTPS']) && $_SERVER['HTTPS'] !== 'off');
        setcookie('apex_remember', $rawToken, [
            'expires'  => time() + 30 * 86400,
            'path'     => '/',
            'domain'   => '',
            'secure'   => $secure,
            'httponly' => true,
            'samesite' => 'Lax',
        ]);

        return true;
    } catch (Throwable $_) {
        return false;
    }
}

/**
 * Xóa tất cả remember tokens của user hiện tại
 */
function apex_admin_clear_remember_tokens(int $adminId): void {
    try {
        $pdo = apex_db();
        $pdo->prepare("DELETE FROM admin_remember_tokens WHERE admin_id = ?")->execute([$adminId]);
    } catch (Throwable $_) {}
}

/**
 * Yêu cầu đăng nhập - redirect về login page nếu chưa
 */
function apex_admin_require(): array {
    if (!apex_admin_authenticated()) {
        $current = $_SERVER['REQUEST_URI'] ?? '/admin/';
        header('Location: ' . apex_admin_url('login.php?next=' . urlencode($current)));
        exit;
    }
    return [
        'id'   => (int)$_SESSION['apex_admin_id'],
        'user' => (string)$_SESSION['apex_admin_user'],
    ];
}

/**
 * Yêu cầu đăng nhập cho API JSON - trả về 401 thay vì redirect
 */
function apex_admin_require_json(): array {
    if (!apex_admin_authenticated()) {
        http_response_code(401);
        header('Content-Type: application/json');
        echo json_encode(['ok' => false, 'error' => 'Unauthorized']);
        exit;
    }
    return [
        'id'   => (int)$_SESSION['apex_admin_id'],
        'user' => (string)$_SESSION['apex_admin_user'],
    ];
}

/**
 * Trả về JSON response
 */
function json_response(array $data, int $code = 200): void {
    http_response_code($code);
    header('Content-Type: application/json');
    echo json_encode($data, JSON_UNESCAPED_UNICODE | JSON_PRETTY_PRINT);
    exit;
}

/**
 * Thử đăng nhập với username/password
 */
function apex_admin_login(string $username, string $password, bool $remember = false): array {
    try {
        $pdo = apex_db();
        $stmt = $pdo->prepare("SELECT id, username, password_hash FROM admins WHERE username = ? LIMIT 1");
        $stmt->execute([trim($username)]);
        $row = $stmt->fetch();

        if (!$row) {
            usleep(random_int(100000, 300000));
            return ['ok' => false, 'error' => 'Sai tài khoản hoặc mật khẩu'];
        }

        if (!password_verify($password, $row['password_hash'])) {
            usleep(random_int(100000, 300000));
            apex_admin_log('login_failed', ['user' => $username]);
            return ['ok' => false, 'error' => 'Sai tài khoản hoặc mật khẩu'];
        }

        // Rehash nếu algorithm/cost thay đổi
        if (password_needs_rehash($row['password_hash'], PASSWORD_ARGON2ID, ['memory_cost' => 65536, 'time_cost' => 4, 'threads' => 2])) {
            $newHash = password_hash($password, PASSWORD_ARGON2ID, ['memory_cost' => 65536, 'time_cost' => 4, 'threads' => 2]);
            $pdo->prepare("UPDATE admins SET password_hash = ? WHERE id = ?")->execute([$newHash, $row['id']]);
        }

        // Regenerate session ID chống fixation
        session_regenerate_id(true);

        $_SESSION['apex_admin_id']        = (int)$row['id'];
        $_SESSION['apex_admin_user']      = $row['username'];
        $_SESSION['apex_admin_last_seen'] = time();
        $_SESSION['apex_admin_login_at']  = time();

        // Remember me: cấp token
        if ($remember) {
            apex_admin_issue_remember_token((int)$row['id']);
        }

        // Update last_login
        $pdo->prepare("UPDATE admins SET last_login = NOW() WHERE id = ?")->execute([$row['id']]);

        apex_admin_log('login_success', ['user' => $username, 'remember' => $remember]);
        return ['ok' => true];

    } catch (Throwable $e) {
        return ['ok' => false, 'error' => 'Chưa import schema.sql hoặc sai cấu hình DB'];
    }
}

/**
 * Đăng xuất - xóa cả session và remember token cookie
 */
function apex_admin_logout(): void {
    $adminId = (int)($_SESSION['apex_admin_id'] ?? 0);
    apex_admin_log('logout');

    // Xóa token trong DB
    if ($adminId) {
        apex_admin_clear_remember_tokens($adminId);
    }

    $_SESSION = [];

    if (ini_get('session.use_cookies')) {
        $p = session_get_cookie_params();
        setcookie(session_name(), '', time() - 42000, $p['path'], $p['domain'], $p['secure'], $p['httponly']);
    }

    // Xóa remember cookie
    setcookie('apex_remember', '', time() - 3600, '/', '', false, true);

    if (session_status() === PHP_SESSION_ACTIVE) {
        session_destroy();
    }
}

/**
 * CSRF token
 */
function apex_admin_csrf_token(): string {
    if (empty($_SESSION['apex_admin_csrf'])) {
        $_SESSION['apex_admin_csrf'] = bin2hex(random_bytes(32));
    }
    return $_SESSION['apex_admin_csrf'];
}

function apex_admin_csrf_check(): void {
    $token = $_POST['csrf'] ?? $_SERVER['HTTP_X_CSRF_TOKEN'] ?? '';
    if (!hash_equals($_SESSION['apex_admin_csrf'] ?? '', $token)) {
        http_response_code(419);
        die('CSRF token mismatch');
    }
}
