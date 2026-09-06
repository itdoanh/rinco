<?php
/**
 * APEX Admin - Settings
 * Đổi mật khẩu admin
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';
require_once __DIR__ . '/includes/queries.php';

$user = apex_admin_require();
$csrf = apex_admin_csrf_token();

$flash = '';
$error = '';

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    apex_admin_csrf_check();

    $currentPass = $_POST['current_password'] ?? '';
    $newPass     = $_POST['new_password'] ?? '';
    $confirmPass = $_POST['confirm_password'] ?? '';

    if ($currentPass === '' || $newPass === '' || $confirmPass === '') {
        $error = 'Vui lòng điền đầy đủ tất cả các trường.';
    } elseif ($newPass !== $confirmPass) {
        $error = 'Mật khẩu mới và xác nhận không khớp.';
    } elseif (strlen($newPass) < 8) {
        $error = 'Mật khẩu mới phải có ít nhất 8 ký tự.';
    } else {
        // Verify current password
        $pdo = apex_db();
        $stmt = $pdo->prepare("SELECT password_hash FROM admins WHERE id = ?");
        $stmt->execute([$user['id']]);
        $row = $stmt->fetch();

        if (!$row || !password_verify($currentPass, $row['password_hash'])) {
            $error = 'Mật khẩu hiện tại không đúng.';
            apex_admin_log('password_change_failed', ['user' => $user['user']]);
        } else {
            // Update password
            $newHash = password_hash($newPass, PASSWORD_ARGON2ID, [
                'memory_cost' => 65536,
                'time_cost'   => 4,
                'threads'     => 2,
            ]);
            $pdo->prepare("UPDATE admins SET password_hash = ? WHERE id = ?")
                ->execute([$newHash, $user['id']]);

            $flash = 'Mật khẩu đã được thay đổi thành công!';
            apex_admin_log('password_changed', ['user' => $user['user']]);
        }
    }
}

// Get admin info
$pdo = apex_db();
$admin = $pdo->prepare("SELECT * FROM admins WHERE id = ?");
$admin->execute([$user['id']]);
$admin = $admin->fetch();

$pageTitle = '⚙️ Cài đặt';
$showDateFilter = false;

require __DIR__ . '/includes/header.php';
?>

<?php if ($error): ?>
  <div class="alert alert--error mb-4">⚠️ <?= h($error) ?></div>
<?php endif; ?>
<?php if ($flash): ?>
  <div class="alert alert--success mb-4" data-apex-toast>✅ <?= h($flash) ?></div>
<?php endif; ?>

<div class="panel">
  <div class="panel__head">
    <div class="panel__title">🔐 Đổi mật khẩu</div>
  </div>

  <form method="POST" style="max-width:480px">
    <input type="hidden" name="csrf" value="<?= h($csrf) ?>">

    <div class="field mb-3">
      <label class="field__label">Tài khoản</label>
      <input type="text" value="<?= h($admin['username']) ?>" disabled style="background:#f1f5f9;color:var(--muted)">
    </div>

    <div class="field mb-3">
      <label class="field__label">Mật khẩu hiện tại</label>
      <input type="password" name="current_password" required autofocus autocomplete="current-password" placeholder="••••••••">
    </div>

    <div class="field mb-3">
      <label class="field__label">Mật khẩu mới</label>
      <input type="password" name="new_password" required autocomplete="new-password" placeholder="Ít nhất 8 ký tự"
             minlength="8">
      <small class="text-sm text-muted mt-1" style="display:block">
        Nên dùng: chữ hoa, chữ thường, số, ký tự đặc biệt
      </small>
    </div>

    <div class="field mb-4">
      <label class="field__label">Xác nhận mật khẩu mới</label>
      <input type="password" name="confirm_password" required autocomplete="new-password" placeholder="Nhập lại mật khẩu mới">
    </div>

    <button type="submit" class="btn">💾 Lưu mật khẩu mới</button>
  </form>
</div>

<div class="panel">
  <div class="panel__head">
    <div class="panel__title">ℹ️ Thông tin tài khoản</div>
  </div>
  <table class="data" style="max-width:480px;margin-bottom:0">
    <tr>
      <td style="width:40%;color:var(--muted)">Username</td>
      <td><strong><?= h($admin['username']) ?></strong></td>
    </tr>
    <tr>
      <td style="color:var(--muted)">Last login</td>
      <td>
        <?= $admin['last_login'] ? date('d/m/Y H:i:s', strtotime($admin['last_login'])) : 'Chưa có' ?>
      </td>
    </tr>
    <tr>
      <td style="color:var(--muted)">Created</td>
      <td><?= date('d/m/Y H:i', strtotime($admin['created_at'])) ?></td>
    </tr>
    <tr>
      <td style="color:var(--muted)">Security</td>
      <td><span class="pill-status pill-status--new">Argon2ID</span></td>
    </tr>
  </table>
</div>

<div class="panel">
  <div class="panel__head">
    <div class="panel__title">🔗 Liên kết nhanh</div>
  </div>
  <div class="flex gap-3">
    <a href="<?= h(apex_admin_url('')) ?>" class="btn btn--ghost">📊 Dashboard</a>
    <a href="<?= h(apex_admin_url('leads.php')) ?>" class="btn btn--ghost">👥 Leads</a>
    <a href="<?= BASE_PATH === '' ? '/' : BASE_PATH . '/' ?>" target="_blank" class="btn btn--ghost">🌐 Xem website</a>
  </div>
</div>

<?php require __DIR__ . '/includes/footer.php'; ?>
