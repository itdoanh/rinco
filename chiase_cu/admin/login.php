<?php
/**
 * APEX Admin - Login page
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';

// Nếu đã login thì về dashboard
if (apex_admin_authenticated()) {
    apex_admin_redirect('');
    exit;
}

$error = '';
$username = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    apex_admin_csrf_check();
    $username = trim($_POST['username'] ?? '');
    $password = $_POST['password'] ?? '';
    $remember = !empty($_POST['remember']);

    if ($username === '' || $password === '') {
        $error = 'Vui lòng nhập tài khoản và mật khẩu';
    } else {
        $result = apex_admin_login($username, $password, $remember);
        if ($result['ok']) {
            $next = $_GET['next'] ?? '';
            // Chỉ cho redirect tới cùng admin path
            if (!$next || strpos($next, ADMIN_BASE) !== 0) {
                $next = '';
            }
            apex_admin_redirect($next);
            exit;
        } else {
            $error = $result['error'];
        }
    }
}

$csrf = apex_admin_csrf_token();
?>
<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title>Đăng nhập · APEX Admin</title>
<link rel="stylesheet" href="<?= h(apex_admin_url('assets/css/admin.css')) ?>">
<link rel="icon" href="<?= h(apex_asset_url('assets/img/apex-mark.webp')) ?>">
</head>
<body class="login-page">
  <form method="POST" class="login-card">
    <div class="login-card__brand">
      <img src="<?= h(apex_asset_url('assets/img/apex-mark.webp')) ?>" alt="APEX">
      <div class="login-card__brand-name">APEX Admin</div>
    </div>
    <div class="login-card__title">Đăng nhập vào Dashboard</div>

    <?php if ($error): ?>
      <div class="alert alert--error">⚠️ <?= h($error) ?></div>
    <?php endif; ?>

    <input type="hidden" name="csrf" value="<?= h($csrf) ?>">

    <div class="field mb-3">
      <label class="field__label">Tài khoản</label>
      <input type="text" name="username" value="<?= h($username) ?>" required autofocus autocomplete="username" placeholder="admin">
    </div>

    <div class="field mb-3">
      <label class="field__label">Mật khẩu</label>
      <input type="password" name="password" required autocomplete="current-password" placeholder="••••••••">
    </div>

    <div class="field field--checkbox mb-4">
      <label class="checkbox-label">
        <input type="checkbox" name="remember" value="1">
        <span>Ghi nhớ đăng nhập (30 ngày)</span>
      </label>
    </div>

    <button type="submit" class="btn" style="width:100%;padding:14px;font-size:14px">
      🔐 ĐĂNG NHẬP
    </button>

    <div style="text-align:center;margin-top:24px;font-size:11px;color:var(--muted)">
      Protected by Argon2ID • <?= h(BRAND_NAME ?? 'APEX Fintech') ?>
    </div>
  </form>

  <script>
    document.querySelector('form').addEventListener('submit', function () {
      const btn = this.querySelector('button[type=submit]');
      btn.disabled = true;
      btn.textContent = '⏳ Đang đăng nhập...';
    });
  </script>
</body>
</html>
