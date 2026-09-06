<?php
/**
 * APEX Admin - Common header
 * Sidebar navigation + top bar + date range filter
 */

if (!defined('APEX_ADMIN')) {
    http_response_code(403);
    die('Forbidden');
}

require_once __DIR__ . '/auth.php';
$user = apex_admin_require();

$currentPage = basename($_SERVER['SCRIPT_NAME'], '.php');
$range = apex_parse_date_range();
$csrf = apex_admin_csrf_token();

// Set CSRF cookie for API authentication (must be set BEFORE any output)
// Cookie để API lead-status.php verify khi gọi từ admin page
if (!headers_sent()) {
    setcookie('apex_admin_csrf', $csrf, [
        'expires' => time() + 86400,
        'path' => '/',
        'httponly' => false, // Allow JS to read (needed for fetch)
        'samesite' => 'Lax',
    ]);
}
?>
<!DOCTYPE html>
<html lang="vi">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="robots" content="noindex,nofollow">
<title><?= h($pageTitle ?? 'Dashboard') ?> · APEX Admin</title>
<link rel="stylesheet" href="<?= h(apex_admin_url('assets/css/admin.css')) ?>?v=3">
<link rel="icon" href="<?= h(apex_asset_url('assets/img/apex-mark.webp')) ?>">

<!-- Chart.js for dashboard charts -->
<script src="https://cdn.jsdelivr.net/npm/chart.js@4.4.1/dist/chart.umd.min.js"></script>
<?php
  // Expose runtime config cho JS. Hỗ trợ cả BASE_PATH='' (root) và BASE_PATH='/chiase'.
  // KHÔNG phụ thuộc vào việc JS file được cache cũ hay mới.
  $apex_js_base = '';
  $bp = defined('BASE_PATH') ? (string)BASE_PATH : '';
  // Loại bỏ protocol+domain nếu lỡ config nhầm (vd 'https://domain.com/chiase/')
  $bp = preg_replace('#^https?://[^/]+#i', '', $bp);
  $bp = '/' . ltrim($bp, '/');
  $bp = rtrim($bp, '/');
  $apex_js_base = $bp;
  $apex_js_admin = $bp . '/admin';
  $apex_api_base = $bp . '/api';
?>
<script>
window.APEX_BASE = <?= json_encode($apex_js_base) ?>;
window.APEX_ADMIN_BASE = <?= json_encode($apex_js_admin) ?>;
window.APEX_API_BASE = <?= json_encode($apex_api_base) ?>;
</script>
</head>
<body>

<div class="layout">
  <aside class="sidebar">
    <div class="sidebar__brand">
      <img src="<?= h(apex_asset_url('assets/img/apex-mark.webp')) ?>" alt="APEX" width="32" height="32">
      <div>
        <div class="sidebar__brand-name">APEX Admin</div>
        <div class="sidebar__brand-sub">v3.0 Dashboard</div>
      </div>
    </div>

    <nav class="sidebar__nav">
      <a href="<?= h(apex_admin_url('')) ?>" class="nav-item <?= $currentPage === 'index' ? 'is-active' : '' ?>">
        <span class="nav-icon">📊</span>
        <span>Tổng quan</span>
      </a>
      <a href="<?= h(apex_admin_url('leads.php')) ?>" class="nav-item <?= $currentPage === 'leads' ? 'is-active' : '' ?>">
        <span class="nav-icon">👥</span>
        <span>Leads</span>
        <?php if ($leadCount = apex_query_lead_counter_today()): ?>
          <span class="nav-badge"><?= $leadCount ?></span>
        <?php endif; ?>
      </a>
      <a href="<?= h(apex_admin_url('events.php')) ?>" class="nav-item <?= $currentPage === 'events' ? 'is-active' : '' ?>">
        <span class="nav-icon">⚡</span>
        <span>Events</span>
      </a>
      <a href="<?= h(apex_admin_url('sessions.php')) ?>" class="nav-item <?= $currentPage === 'sessions' ? 'is-active' : '' ?>">
        <span class="nav-icon">🧭</span>
        <span>Sessions</span>
      </a>
      <div class="nav-divider"></div>
      <a href="<?= h(apex_admin_url('settings.php')) ?>" class="nav-item <?= $currentPage === 'settings' ? 'is-active' : '' ?>">
        <span class="nav-icon">⚙️</span>
        <span>Cài đặt</span>
      </a>
      <a href="<?= h(BASE_PATH === '' ? '/' : BASE_PATH . '/') ?>" target="_blank" class="nav-item">
        <span class="nav-icon">🌐</span>
        <span>Xem website</span>
        <span class="nav-external">↗</span>
      </a>
    </nav>

    <div class="sidebar__footer">
      <div class="sidebar__user">
        <div class="avatar"><?= strtoupper(substr($user['user'], 0, 1)) ?></div>
        <div class="sidebar__user-info">
          <div class="sidebar__user-name"><?= h($user['user']) ?></div>
          <a href="<?= h(apex_admin_url('logout.php')) ?>" class="sidebar__user-logout">Đăng xuất</a>
        </div>
      </div>
    </div>
  </aside>

  <main class="main">
    <header class="topbar">
      <div>
        <h1 class="topbar__title"><?= h($pageTitle ?? 'Dashboard') ?></h1>
        <?php if (!empty($pageSubtitle)): ?>
          <div class="topbar__subtitle"><?= h($pageSubtitle) ?></div>
        <?php endif; ?>
      </div>

      <div class="topbar__right">
        <?php if (!empty($showDateFilter)): ?>
          <form method="GET" class="date-filter">
            <input type="date" name="from" value="<?= h($range['from']) ?>" max="<?= date('Y-m-d') ?>">
            <span>→</span>
            <input type="date" name="to" value="<?= h($range['to']) ?>" max="<?= date('Y-m-d') ?>">
            <button type="submit" class="btn btn--sm btn--ghost">Áp dụng</button>
            <?php
              $presets = [
                'today' => 'Hôm nay',
                '7d'    => '7 ngày',
                '30d'   => '30 ngày',
                '90d'   => '90 ngày',
              ];
              foreach ($presets as $key => $label):
                if ($key === 'today') {
                  $pFrom = $pTo = date('Y-m-d');
                } else {
                  $pFrom = date('Y-m-d', strtotime('-' . $key));
                  $pTo   = date('Y-m-d');
                }
                $active = ($range['from'] === $pFrom && $range['to'] === $pTo) ? 'is-active' : '';
                echo '<button type="button" class="btn btn--xs btn--ghost ' . $active . '" onclick="window.location.href=\'?from=' . $pFrom . '&to=' . $pTo . '\'">' . $label . '</button>';
              endforeach;
            ?>
          </form>
        <?php endif; ?>

        <div class="live-indicator" id="liveIndicator" title="Tự động refresh mỗi 30s">
          <span class="live-dot"></span>
          <span class="live-text">LIVE</span>
        </div>
      </div>
    </header>

    <div class="content">
