<?php
/**
 * APEX Admin - Sessions Management
 * Danh sách sessions + chi tiết từng session
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';
require_once __DIR__ . '/includes/queries.php';

$user = apex_admin_require();
$pdo  = apex_db();

// Single session detail
if (!empty($_GET['id'])) {
    $sessionId = trim($_GET['id']);
    $data = apex_query_session($sessionId);

    if (!$data) {
        http_response_code(404);
        $pageTitle = 'Session not found';
        require __DIR__ . '/includes/header.php';
        echo '<div class="panel"><div class="empty">Session không tìm thấy.</div></div>';
        require __DIR__ . '/includes/footer.php';
        exit;
    }

    $s    = $data['session'];
    $evts = $data['events'];
    $lead = $data['lead'];

    $pageTitle = '🧭 Session Detail';
    $pageSubtitle = 'ID: ' . h($s['session_id']);

    require __DIR__ . '/includes/header.php';
?>

<div class="panel">
  <div class="panel__head">
    <div class="panel__title">Thông tin Session</div>
    <a href="<?= h(apex_admin_url('sessions.php')) ?>" class="btn btn--sm btn--ghost">← Quay lại danh sách</a>
  </div>

  <div class="grid-2">
    <div>
      <h4 class="text-sm text-muted mb-2" style="text-transform:uppercase">Session Info</h4>
      <table class="data" style="margin-bottom:0">
        <tr><td style="width:40%;color:var(--muted)">Session ID</td><td><code style="font-size:11px" data-copy="<?= h($s['session_id']) ?>"><?= h($s['session_id']) ?></code></td></tr>
        <tr><td>First Visit</td><td><?= date('d/m/Y H:i:s', strtotime($s['first_visit'])) ?></td></tr>
        <tr><td>Last Activity</td><td><?= date('d/m/Y H:i:s', strtotime($s['last_activity'])) ?></td></tr>
        <tr><td>Duration</td><td><?= apex_format_duration($s['total_duration']) ?></td></tr>
        <tr><td>Page Views</td><td><?= $s['page_views'] ?></td></tr>
        <tr><td>Max Scroll</td><td><?= $s['scroll_max'] ?>%</td></tr>
        <tr>
          <td>Lead Status</td>
          <td>
            <?php if ($s['is_lead']): ?>
              <span class="pill-status pill-status--new">Converted to Lead</span>
            <?php else: ?>
              <span class="pill-status pill-status--seen">No conversion</span>
            <?php endif; ?>
          </td>
        </tr>
      </table>
    </div>
    <div>
      <h4 class="text-sm text-muted mb-2" style="text-transform:uppercase">User Info</h4>
      <table class="data" style="margin-bottom:0">
        <tr><td style="width:40%;color:var(--muted)">Device</td><td><?= h(ucfirst($s['device_type'] ?? 'unknown')) ?></td></tr>
        <tr><td>Browser</td><td><?= h($s['browser_name'] ?? '—') ?></td></tr>
        <tr><td>OS</td><td><?= h($s['os_name'] ?? '—') ?></td></tr>
        <tr><td>UTM Source</td><td><?= h($s['utm_source'] ?: '—') ?></td></tr>
      </table>

      <?php if ($lead): ?>
        <h4 class="text-sm text-muted mb-2 mt-4" style="text-transform:uppercase">Lead (nếu có)</h4>
        <table class="data" style="margin-bottom:0">
          <tr><td style="width:40%">Name</td><td><strong><?= h($lead['name']) ?></strong></td></tr>
          <tr><td>Phone</td><td><a href="tel:<?= h($lead['phone']) ?>" style="color:var(--orange)"><?= h($lead['phone']) ?></a></td></tr>
          <tr><td>Form</td><td><span class="pill pill--hero"><?= h($lead['form_type']) ?></span></td></tr>
          <tr><td>Time</td><td><?= date('d/m/Y H:i', strtotime($lead['created_at'])) ?></td></tr>
        </table>
      <?php endif; ?>
    </div>
  </div>
</div>

<!-- Event Timeline -->
<div class="panel">
  <div class="panel__head">
    <div class="panel__title">⚡ Event Timeline (<?= count($evts) ?> events)</div>
    <a href="<?= h(apex_admin_url('api/export.php')) ?>?type=events&session=<?= urlencode($sessionId) ?>" class="btn btn--sm btn--ghost">📥 Export</a>
  </div>

  <div style="max-height:500px;overflow-y:auto">
    <table class="data">
      <thead>
        <tr>
          <th>Thời gian</th>
          <th>Event</th>
          <th>Value</th>
          <th>Page</th>
          <th>UTM</th>
        </tr>
      </thead>
      <tbody>
        <?php
          $icons = [
            'page_view'      => '📄',
            'cta_click'     => '🖱️',
            'form_focus'    => '📝',
            'form_submit'   => '✅',
            'scroll_depth'  => '⬇️',
            'lead'          => '🎯',
            'time_tick'     => '⏱️',
            'session_start' => '🚀',
          ];
          $prevTime = null;
          foreach ($evts as $e):
            $icon = $icons[$e['event_type']] ?? '📌';
            $elapsed = '';
            if ($prevTime) {
              $elapsed = round((strtotime($e['created_at']) - strtotime($prevTime)) / 1000);
              if ($elapsed > 0) $elapsed = '+' . $elapsed . 's';
            }
            $prevTime = $e['created_at'];
        ?>
          <tr>
            <td>
              <div><?= date('H:i:s', strtotime($e['created_at'])) ?></div>
              <?php if ($elapsed): ?>
                <small style="color:var(--orange);font-weight:600"><?= $elapsed ?></small>
              <?php endif; ?>
            </td>
            <td>
              <span style="font-size:12px"><?= $icon ?> <?= h($e['event_type']) ?></span>
            </td>
            <td>
              <?php if ($e['event_value']): ?>
                <code style="font-size:10px;background:var(--bg);padding:1px 5px;border-radius:3px">
                  <?= h(mb_substr($e['event_value'], 0, 80)) ?>
                </code>
              <?php else: ?>
                <span class="text-muted">—</span>
              <?php endif; ?>
            </td>
            <td class="text-sm"><?= h(mb_substr($e['page_url'] ?: '', 0, 50)) ?></td>
            <td class="text-sm">
              <?php if ($e['utm_source']): ?>
                <span class="pill"><?= h($e['utm_source']) ?></span>
              <?php else: ?>
                <span class="text-muted">—</span>
              <?php endif; ?>
            </td>
          </tr>
        <?php endforeach; ?>
      </tbody>
    </table>
  </div>
</div>

<script>
// Visual timeline bar
document.addEventListener('DOMContentLoaded', function() {
  // Scroll tracking visualization
  const rows = document.querySelectorAll('tbody tr');
  let minTime = null, maxTime = null;
  rows.forEach(row => {
    const td = row.querySelector('td:first-child');
    if (!td) return;
    const timeMatch = td.textContent.match(/(\d{2}:\d{2}:\d{2})/);
    if (!timeMatch) return;
  });
});
</script>

<?php
    require __DIR__ . '/includes/footer.php';
    exit;
}

// ===== SESSION LIST =====
$range = apex_parse_date_range();
$page    = max(1, (int)($_GET['page'] ?? 1));
$perPage = 50;
$offset  = ($page - 1) * $perPage;

$total = (int)$pdo->query("
    SELECT COUNT(*) FROM sessions
    WHERE last_activity BETWEEN '{$range['from_dt']}' AND '{$range['to_dt']}'
")->fetchColumn();

$sessions = $pdo->query("
    SELECT s.*,
           (SELECT COUNT(*) FROM events WHERE session_id=s.session_id) AS event_count,
           (SELECT name FROM leads WHERE session_id=s.session_id ORDER BY id DESC LIMIT 1) AS lead_name,
           (SELECT id   FROM leads WHERE session_id=s.session_id ORDER BY id DESC LIMIT 1) AS lead_id
    FROM sessions s
    WHERE s.last_activity BETWEEN '{$range['from_dt']}' AND '{$range['to_dt']}'
    ORDER BY s.last_activity DESC
    LIMIT $perPage OFFSET $offset
")->fetchAll();

$pageTitle = '🧭 Sessions';
$pageSubtitle = number_format($total) . ' sessions trong khoảng thời gian đã chọn';
$showDateFilter = true;

require __DIR__ . '/includes/header.php';
?>

<div class="panel">
  <div class="panel__head">
    <div>
      <div class="panel__title">Danh sách Sessions</div>
      <div class="text-sm text-muted mt-2"><?= number_format($total) ?> sessions (<?= number_format($range['days']) ?> ngày)</div>
    </div>
  </div>

  <div class="table-wrap">
    <table class="data">
      <thead>
        <tr>
          <th>Session ID</th>
          <th>First Visit</th>
          <th>Last Activity</th>
          <th class="text-right">Duration</th>
          <th class="text-right">Pages</th>
          <th class="text-right">Scroll</th>
          <th>Device</th>
          <th>UTM Source</th>
          <th>Lead</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <?php foreach ($sessions as $s):
          $icons = ['mobile' => '📱', 'desktop' => '💻', 'tablet' => '📱', 'unknown' => '❓'];
          $icon = $icons[$s['device_type']] ?? '❓';
        ?>
          <tr>
            <td><code style="font-size:11px" data-copy="<?= h($s['session_id']) ?>"><?= h(mb_substr($s['session_id'], 0, 8)) ?>...</code></td>
            <td><small><?= date('d/m H:i', strtotime($s['first_visit'])) ?></small></td>
            <td><small><?= date('d/m H:i', strtotime($s['last_activity'])) ?></small></td>
            <td class="text-right"><?= apex_format_duration($s['total_duration']) ?></td>
            <td class="text-right num"><?= $s['page_views'] ?></td>
            <td class="text-right">
              <?php if ($s['scroll_max'] > 0): ?>
                <div style="display:flex;align-items:center;gap:6px;justify-content:flex-end">
                  <div style="width:50px;height:4px;background:var(--bg-2);border-radius:2px;overflow:hidden">
                    <div style="width:<?= $s['scroll_max'] ?>%;height:100%;background:var(--orange);border-radius:2px"></div>
                  </div>
                  <span style="font-size:11px"><?= $s['scroll_max'] ?>%</span>
                </div>
              <?php else: ?>
                <span class="text-muted">—</span>
              <?php endif; ?>
            </td>
            <td class="text-sm"><?= $icon ?> <?= h($s['device_type']) ?></td>
            <td class="text-sm"><?= h($s['utm_source'] ?: '—') ?></td>
            <td>
              <?php if ($s['lead_id']): ?>
                <span class="pill-status pill-status--new">
                  🎯 <?= h(mb_substr($s['lead_name'], 0, 15)) ?>
                </span>
              <?php else: ?>
                <span class="text-muted">—</span>
              <?php endif; ?>
            </td>
            <td>
              <a href="?id=<?= h($s['session_id']) ?>" class="btn btn--xs btn--ghost">Xem</a>
            </td>
          </tr>
        <?php endforeach; ?>
        <?php if (empty($sessions)): ?>
          <tr><td colspan="10" class="empty">Chưa có session nào.</td></tr>
        <?php endif; ?>
      </tbody>
    </table>
  </div>

  <?php if ($total > $perPage): ?>
    <?php
      $pages = max(1, (int)ceil($total / $perPage));
      $baseUrl = '?from=' . urlencode($range['from']) . '&to=' . urlencode($range['to']);
    ?>
    <div class="flex-between mt-4">
      <div class="text-sm text-muted">Trang <?= $page ?> / <?= $pages ?></div>
      <div class="flex gap-2">
        <?php if ($page > 1): ?>
          <a href="<?= $baseUrl ?>&page=<?= $page - 1 ?>" class="btn btn--sm btn--ghost">← Trước</a>
        <?php endif; ?>
        <?php if ($page < $pages): ?>
          <a href="<?= $baseUrl ?>&page=<?= $page + 1 ?>" class="btn btn--sm btn--ghost">Sau →</a>
        <?php endif; ?>
      </div>
    </div>
  <?php endif; ?>
</div>

<?php require __DIR__ . '/includes/footer.php'; ?>
