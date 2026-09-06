<?php
/**
 * APEX Admin - Events Timeline
 * Xem chi tiết các sự kiện theo thời gian thực
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';
require_once __DIR__ . '/includes/queries.php';

$user = apex_admin_require();

$type    = $_GET['type'] ?? null;
$session = $_GET['session'] ?? null;
$page    = max(1, (int)($_GET['page'] ?? 1));
$perPage = 100;
$offset  = ($page - 1) * $perPage;

$pdo = apex_db();

// Stats overview
$eventTypes = $pdo->query("
    SELECT event_type, COUNT(*) AS cnt
    FROM events
    WHERE DATE(created_at) >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)
    GROUP BY event_type
    ORDER BY cnt DESC
")->fetchAll();

$eventsData = apex_query_recent_events($perPage + 1, $type, $session);
$hasMore = count($eventsData) > $perPage;
$eventsData = array_slice($eventsData, 0, $perPage);

$pageTitle = '⚡ Events Timeline';
$pageSubtitle = !empty($type) ? 'Type: ' . h($type) : ($session ? 'Session: ' . h($session) : 'Tất cả events (30 ngày gần nhất)');
$showDateFilter = true;

require __DIR__ . '/includes/header.php';
?>

<!-- Event type filter -->
<div class="panel" style="margin-bottom:20px">
  <div class="panel__head">
    <div class="panel__title">Bộ lọc event type</div>
    <a href="<?= h(apex_admin_url('events.php')) ?>" class="btn btn--xs btn--ghost <?= !$type ? 'is-active' : '' ?>">Tất cả</a>
    <?php foreach ($eventTypes as $et): ?>
      <?php
        $icons = [
          'page_view'     => '📄',
          'cta_click'     => '🖱️',
          'form_focus'    => '📝',
          'form_submit'   => '✅',
          'scroll_depth'  => '⬇️',
          'lead'          => '🎯',
          'time_tick'     => '⏱️',
          'session_start' => '🚀',
          'form_idle_15s' => '💤',
        ];
        $icon = $icons[$et['event_type']] ?? '📌';
      ?>
      <a href="?type=<?= urlencode($et['event_type']) ?>"
         class="btn btn--xs btn--ghost <?= $type === $et['event_type'] ? 'is-active' : '' ?>">
        <?= $icon ?> <?= h($et['event_type']) ?> (<?= number_format($et['cnt']) ?>)
      </a>
    <?php endforeach; ?>
  </div>
</div>

<!-- Events table -->
<div class="panel">
  <div class="panel__head">
    <div class="panel__title">
      <?php if ($type): ?>
        <?= $icons[$type] ?? '📌' ?> <?= h($type) ?>
      <?php elseif ($session): ?>
        🧭 Session: <?= h($session) ?>
      <?php else: ?>
        ⚡ Tất cả Events
      <?php endif; ?>
      <span class="text-muted" style="font-weight:400;font-size:12px;margin-left:8px">
        Hiển thị <?= count($eventsData) ?> events
      </span>
    </div>
    <div class="panel__actions">
      <?php if ($type || $session): ?>
        <a href="<?= h(apex_admin_url('api/export.php')) ?>?type=events<?= $type ? '&type=' . urlencode($type) : '' ?><?= $session ? '&session=' . urlencode($session) : '' ?>"
           class="btn btn--sm btn--ghost">📥 Export CSV</a>
      <?php endif; ?>
    </div>
  </div>

  <div class="table-wrap">
    <table class="data">
      <thead>
        <tr>
          <th>ID</th>
          <th>Thời gian</th>
          <th>Event</th>
          <th>Value</th>
          <th>Session</th>
          <th>UTM</th>
          <th>Device</th>
          <th>IP</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <?php
          $icons = [
            'page_view'     => '📄',
            'cta_click'     => '🖱️',
            'form_focus'    => '📝',
            'form_submit'   => '✅',
            'scroll_depth'  => '⬇️',
            'lead'          => '🎯',
            'time_tick'     => '⏱️',
            'session_start' => '🚀',
            'form_idle_15s' => '💤',
          ];
          foreach ($eventsData as $e):
            $icon = $icons[$e['event_type']] ?? '📌';
            $eventType = $e['event_type'];
            if ($eventType === 'lead') $color = 'var(--green)';
            elseif ($eventType === 'cta_click') $color = 'var(--orange)';
            elseif ($eventType === 'form_submit') $color = 'var(--blue)';
            elseif ($eventType === 'form_focus') $color = 'var(--purple)';
            elseif ($eventType === 'scroll_depth') $color = 'var(--text-2)';
            else $color = 'var(--text)';
        ?>
          <tr>
            <td class="text-muted" style="font-family:monospace">#<?= $e['id'] ?></td>
            <td>
              <div><?= date('d/m H:i:s', strtotime($e['created_at'])) ?></div>
              <small class="text-muted"><?= apex_time_ago($e['created_at']) ?></small>
            </td>
            <td>
              <span style="color:<?= $color ?>"><?= $icon ?> <?= h($e['event_type']) ?></span>
            </td>
            <td>
              <?php if ($e['event_value']): ?>
                <code style="font-size:11px;background:var(--bg);padding:2px 6px;border-radius:4px">
                  <?= h(mb_substr($e['event_value'], 0, 60)) ?>
                </code>
              <?php else: ?>
                <span class="text-muted">—</span>
              <?php endif; ?>
            </td>
            <td>
              <a href="<?= h(apex_admin_url('sessions.php')) ?>?id=<?= h($e['session_id']) ?>"
                 style="font-family:monospace;font-size:11px;color:var(--orange)">
                <?= h(mb_substr($e['session_id'], 0, 8)) ?>...
              </a>
            </td>
            <td class="text-sm">
              <?php if ($e['utm_source']): ?>
                <span class="pill"><?= h($e['utm_source']) ?></span>
                <?php if ($e['utm_medium']): ?>
                  <div class="text-muted"><?= h($e['utm_medium']) ?></div>
                <?php endif; ?>
              <?php else: ?>
                <span class="text-muted">—</span>
              <?php endif; ?>
            </td>
            <td class="text-sm text-muted">
              <?= h($e['device_type'] ?: '—') ?>
              <div><?= h($e['browser_name'] ?: '') ?></div>
            </td>
            <td style="font-family:monospace;font-size:11px" class="text-muted"><?= h($e['ip']) ?></td>
            <td>
              <a href="<?= h(apex_admin_url('sessions.php')) ?>?id=<?= h($e['session_id']) ?>" class="btn btn--xs btn--ghost">🧭</a>
            </td>
          </tr>
        <?php endforeach; ?>
        <?php if (empty($eventsData)): ?>
          <tr><td colspan="9" class="empty">Không có events nào.</td></tr>
        <?php endif; ?>
      </tbody>
    </table>
  </div>

  <?php if ($hasMore): ?>
    <div class="text-center mt-4">
      <a href="?type=<?= urlencode($type ?? '') ?>&session=<?= urlencode($session ?? '') ?>&page=<?= $page + 1 ?>"
         class="btn">Tải thêm →</a>
    </div>
  <?php endif; ?>
</div>

<?php require __DIR__ . '/includes/footer.php'; ?>
