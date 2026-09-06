<?php
/**
 * APEX Admin - Leads Management v2
 * List + search + filter + Keyset Pagination + Polling + Status
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';
require_once __DIR__ . '/includes/queries.php';
require_once __DIR__ . '/../includes/cache.php';

$user = apex_admin_require();
$pdo = apex_db();

// Handle actions
$action = $_POST['action'] ?? $_GET['action'] ?? '';
$csrf   = $_POST['csrf']   ?? $_GET['csrf']   ?? '';

if ($action && !hash_equals(apex_admin_csrf_token(), $csrf)) {
    http_response_code(419);
    die('CSRF mismatch');
}

$flash = '';
if ($action === 'delete' && !empty($_POST['id'])) {
    try {
        $id = (int)$_POST['id'];
        $stmt = $pdo->prepare("DELETE FROM leads WHERE id = ?");
        $stmt->execute([$id]);
        $flash = 'Đã xóa lead #' . $id;
        apex_admin_log('lead_delete', ['id' => $id]);
        ApexCache::invalidateNamespace('lead_list');
    } catch (Throwable $e) {
        $flash = 'Lỗi: ' . $e->getMessage();
    }
} elseif ($action === 'delete_bulk' && !empty($_POST['ids']) && is_array($_POST['ids'])) {
    try {
        $ids = array_map('intval', $_POST['ids']);
        $placeholders = implode(',', array_fill(0, count($ids), '?'));
        $stmt = $pdo->prepare("DELETE FROM leads WHERE id IN ($placeholders)");
        $stmt->execute($ids);
        $flash = 'Đã xóa ' . count($ids) . ' leads';
        apex_admin_log('lead_delete_bulk', ['count' => count($ids)]);
        ApexCache::invalidateNamespace('lead_list');
    } catch (Throwable $e) {
        $flash = 'Lỗi: ' . $e->getMessage();
    }
}

$range    = apex_parse_date_range();
$search   = trim($_GET['q'] ?? '');
$afterId  = $_GET['after'] ?? null;  // Keyset cursor
$status   = trim($_GET['status'] ?? '');
$formType = trim($_GET['form_type'] ?? '');
$source   = trim($_GET['source'] ?? '');

$leadsData = apex_query_leads_keyset($range, $search, $afterId, 50, $status, $formType, $source);

$pageTitle = 'Quản lý Leads';
$pageSubtitle = number_format(count($leadsData['data'])) . ' leads (Cursor: ' . ($afterId ?? 'first') . ')';
$showDateFilter = true;

require __DIR__ . '/includes/header.php';
?>

<?php if ($flash): ?>
<div class="alert alert--success mb-4" data-autohide>✅ <?= h($flash) ?></div>
<?php endif; ?>

<div class="panel">
  <div class="panel__head">
    <div>
      <div class="panel__title">👥 Danh sách Leads</div>
      <div class="text-sm text-muted mt-2">
        Hiển thị: <strong><?= count($leadsData['data']) ?></strong> leads
        <?php if ($search): ?>
          | Tìm: <strong><?= h($search) ?></strong>
          <a href="?" class="text-sm" style="margin-left:8px">[x]</a>
        <?php endif; ?>
        <?php if ($status): ?>
          | Status: <strong><?= h($status) ?></strong>
          <a href="?status=<?= h($status) ?>&amp;action=clear" class="text-sm" style="margin-left:8px">[x]</a>
        <?php endif; ?>
      </div>
    </div>
    <div class="panel__actions">
      <div class="search-bar">
        <input type="search"
               placeholder="Tìm tên, SĐT, nguồn..."
               value="<?= h($search) ?>"
               data-table-filter="#leadsTable"
               autocomplete="off">
      </div>
      <a href="<?= h(apex_admin_url('api/export.php')) ?>?type=leads&from=<?= h($range['from']) ?>&to=<?= h($range['to']) ?><?= $search ? '&q=' . urlencode($search) : '' ?>"
         class="btn btn--sm btn--ghost">📥 CSV</a>
      <button class="btn btn--sm btn--primary" onclick="window.openStatusModal && window.openStatusModal()">📋 Status</button>
    </div>
  </div>

  <!-- Filters -->
  <div class="flex gap-3 mb-4" style="flex-wrap:wrap;align-items:center">
    <span class="text-sm text-muted">Lọc:</span>
    <select class="select-sm" onchange="location.href='?status='+this.value+'&amp;after='">
      <option value="">Tất cả status</option>
      <?php foreach (['new','contacted','qualified','converted','invalid','duplicate'] as $s): ?>
        <option value="<?= $s ?>" <?= $status === $s ? 'selected' : '' ?>><?= ucfirst($s) ?></option>
      <?php endforeach; ?>
    </select>
    <select class="select-sm" onchange="location.href='?form_type='+this.value">
      <option value="">Tất cả form</option>
      <option value="hero" <?= $formType === 'hero' ? 'selected' : '' ?>>Hero</option>
      <option value="modal" <?= $formType === 'modal' ? 'selected' : '' ?>>Modal</option>
      <option value="multistep" <?= $formType === 'multistep' ? 'selected' : '' ?>>Multi-step</option>
    </select>
  </div>

  <form method="POST" id="bulkForm">
    <input type="hidden" name="csrf" value="<?= h(apex_admin_csrf_token()) ?>">
    <div class="flex gap-3 mb-3" style="display:none" id="bulkActions">
      <span class="text-sm" style="align-self:center"><strong id="selectedCount">0</strong> đã chọn</span>
      <button type="submit" name="action" value="delete_bulk"
              data-confirm="Bạn có chắc muốn XÓA các leads đã chọn?"
              class="btn btn--sm" style="background:var(--red)">
        🗑️ Xóa đã chọn
      </button>
      <button type="button" class="btn btn--sm btn--primary"
              onclick="window.openBulkStatusModal && window.openBulkStatusModal()">
        📋 Đổi status
      </button>
    </div>

    <div class="table-wrap">
      <table class="data" id="leadsTable" data-live>
        <thead>
          <tr>
            <th style="width:32px"><input type="checkbox" id="selectAll"></th>
            <th>Thời gian</th>
            <th>Họ tên</th>
            <th>SĐT</th>
            <th>Form</th>
            <th>Status</th>
            <th>Duplicate</th>
            <th>Nguồn</th>
            <th>Device</th>
            <th>IP</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <?php foreach ($leadsData['data'] as $l): ?>
            <?php
              // Determine duplicate/risk display
              $isDuplicate = !empty($l['duplicate_of_id']) || !empty($l['duplicate_group_id']);
              $dupIcon = '';
              $dupClass = '';
              if ($isDuplicate) {
                $dupIcon = '🔄';
                $dupClass = 'text-orange';
                if ($l['status'] === 'duplicate') {
                  $dupIcon = '🔁';
                  $dupClass = 'text-muted';
                }
              }
            ?>
            <tr data-lead-id="<?= (int)$l['id'] ?>" class="<?= $isDuplicate ? 'row-duplicate' : '' ?>">
              <td><input type="checkbox" name="ids[]" value="<?= (int)$l['id'] ?>" class="row-check"></td>
              <td>
                <div><?= date('d/m/Y H:i', strtotime($l['created_at'])) ?></div>
                <small class="text-muted"><?= apex_time_ago($l['created_at']) ?></small>
              </td>
              <td><strong><?= h($l['name']) ?></strong></td>
              <td>
                <a href="tel:<?= h($l['phone']) ?>" style="color:var(--orange);font-weight:600"><?= h($l['phone']) ?></a>
                <button type="button" class="btn btn--xs btn--ghost" onclick="navigator.clipboard.writeText('<?= h($l['phone']) ?>')">📋</button>
              </td>
              <td>
                <?php $cls = ($l['form_type'] === 'hero') ? 'pill--hero' : 'pill--footer'; ?>
                <span class="pill <?= $cls ?>"><?= h($l['form_type'] ?: 'unknown') ?></span>
              </td>
              <td>
                <select class="select-xs status-select"
                        data-lead-id="<?= (int)$l['id'] ?>"
                        onchange="updateLeadStatus(<?= (int)$l['id'] ?>, this.value)">
                  <?php foreach (['new','contacted','qualified','converted','invalid','duplicate'] as $s): ?>
                    <option value="<?= $s ?>" <?= ($l['status'] ?? 'new') === $s ? 'selected' : '' ?>><?= ucfirst($s) ?></option>
                  <?php endforeach; ?>
                </select>
              </td>
              <td>
                <?php if ($isDuplicate): ?>
                  <span class="<?= $dupClass ?>" title="Trùng với lead #<?= $l['duplicate_of_id'] ?? 'N/A' ?>">
                    <?= $dupIcon ?>
                    <?php if (!empty($l['duplicate_of_id'])): ?>
                      <a href="#" onclick="showLeadDetail(<?= (int)$l['duplicate_of_id'] ?>)" class="text-xs">(#<?= $l['duplicate_of_id'] ?>)</a>
                    <?php endif; ?>
                  </span>
                <?php else: ?>
                  <span class="text-muted">—</span>
                <?php endif; ?>
              </td>
              <td class="text-sm">
                <?= h($l['utm_source'] ?: '(direct)') ?>
                <?php if (!empty($l['utm_medium'])): ?>
                  <div class="text-muted"><?= h($l['utm_medium']) ?><?= !empty($l['utm_campaign']) ? ' / ' . h($l['utm_campaign']) : '' ?></div>
                <?php endif; ?>
              </td>
              <td class="text-sm text-muted"><?= h($l['device_type'] ?: '—') ?></td>
              <td class="text-sm text-muted" style="font-family:monospace">
                <?= isset($l['ip_hash']) ? substr($l['ip_hash'], 0, 8) . '...' : (isset($l['ip']) ? preg_replace('/\d+(\d{3})$/', '***$1', $l['ip']) : '***') ?>
              </td>
              <td class="text-right">
                <a href="https://zalo.me/<?= preg_replace('/^0/', '+84', $l['phone']) ?>"
                   target="_blank" class="btn btn--xs btn--ghost" title="Chat Zalo">💬</a>
                <button type="button" class="btn btn--xs btn--ghost"
                        onclick="showLeadDetail(<?= (int)$l['id'] ?>)" title="Xem chi tiết">👁️</button>
                <button type="submit" name="action" value="delete"
                        data-confirm="Xóa lead #<?= (int)$l['id'] ?> - <?= h($l['name']) ?>?"
                        class="btn btn--xs btn--ghost" style="color:var(--red)">🗑️</button>
                <input type="hidden" name="id" value="<?= (int)$l['id'] ?>">
              </td>
            </tr>
          <?php endforeach; ?>
          <?php if (empty($leadsData['data'])): ?>
            <tr><td colspan="11" class="empty">
              <?= $search ? 'Không tìm thấy leads nào khớp "' . h($search) . '"' : 'Chưa có leads nào trong khoảng thời gian này.' ?>
            </td></tr>
          <?php endif; ?>
        </tbody>
      </table>
    </div>

    <!-- Keyset Pagination -->
    <div class="flex-between mt-4">
      <div class="text-sm text-muted">
        <?php if ($afterId): ?>
          <a href="?from=<?= h($range['from']) ?>&amp;to=<?= h($range['to']) ?><?= $search ? '&amp;q=' . urlencode($search) : '' ?><?= $status ? '&amp;status=' . urlencode($status) : '' ?>"
             class="btn btn--sm btn--ghost">← Đầu tiên</a>
        <?php endif; ?>
      </div>
      <div class="flex gap-2">
        <?php if ($leadsData['pagination']['has_next']): ?>
          <span class="text-sm text-muted">Có thêm leads</span>
          <a href="?after=<?= $leadsData['pagination']['next_cursor'] ?>&amp;from=<?= h($range['from']) ?>&amp;to=<?= h($range['to']) ?><?= $search ? '&amp;q=' . urlencode($search) : '' ?><?= $status ? '&amp;status=' . urlencode($status) : '' ?>"
             class="btn btn--sm btn--ghost">Tiếp theo →</a>
        <?php else: ?>
          <span class="text-sm text-muted">— Hết —</span>
        <?php endif; ?>
      </div>
    </div>
  </form>
</div>

<!-- Lead Detail Modal -->
<div id="leadDetailModal" style="display:none;position:fixed;inset:0;z-index:9999;background:rgba(0,0,0,0.6);align-items:center;justify-content:center;padding:20px" onclick="if(event.target===this)closeLeadDetail()">
  <div style="background:var(--bg-card);border-radius:16px;max-width:900px;width:100%;max-height:90vh;overflow:hidden;display:flex;flex-direction:column;box-shadow:0 25px 50px rgba(0,0,0,0.5)">
    <div style="padding:20px 24px;border-bottom:1px solid rgba(255,255,255,0.1);display:flex;align-items:center;justify-content:space-between;background:linear-gradient(135deg,#0A192F,#1a2f4f)">
      <div>
        <h2 style="margin:0;font-size:18px;color:#fff">🔍 Chi tiết Lead</h2>
        <div id="leadDetailName" style="color:#94a3b8;font-size:13px;margin-top:4px"></div>
      </div>
      <button onclick="closeLeadDetail()" style="background:rgba(255,255,255,0.1);border:none;color:#fff;width:36px;height:36px;border-radius:50%;cursor:pointer;font-size:18px;display:flex;align-items:center;justify-content:center">×</button>
    </div>
    <div id="leadDetailContent" style="overflow-y:auto;flex:1;padding:24px;color:#e2e8f0;font-size:14px">
      <div style="text-align:center;padding:40px;color:#94a3b8">
        <div style="font-size:24px;margin-bottom:8px">⏳</div>
        <div>Đang tải dữ liệu...</div>
      </div>
    </div>
  </div>
</div>

<script>
// Bulk select
const selectAll = document.getElementById('selectAll');
const rowChecks = document.querySelectorAll('.row-check');
const bulkActions = document.getElementById('bulkActions');
const selectedCount = document.getElementById('selectedCount');

function updateBulkUI() {
  const checked = document.querySelectorAll('.row-check:checked').length;
  selectedCount.textContent = checked;
  bulkActions.style.display = checked > 0 ? 'flex' : 'none';
}

if (selectAll) {
  selectAll.addEventListener('change', () => {
    rowChecks.forEach(c => c.checked = selectAll.checked);
    updateBulkUI();
  });
}
rowChecks.forEach(c => c.addEventListener('change', updateBulkUI));

// Auto-hide flash
document.querySelectorAll('[data-autohide]').forEach(el => {
  setTimeout(() => { el.style.transition = 'opacity .3s'; el.style.opacity = '0'; }, 4000);
});

// Lead Status API - đúng path: /api/lead-status.php (không phải /admin/api/)
const leadStatusApi = '<?= defined("BASE_PATH") ? rtrim(BASE_PATH, "/") : "" ?>/api/lead-status.php';

async function updateLeadStatus(leadId, status) {
  try {
    const res = await fetch(leadStatusApi, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: leadId, status: status })
    });
    console.log('lead-status API status:', res.status, res.statusText);

    if (!res.ok) {
      const text = await res.text();
      console.error('lead-status API error:', res.status, text);
      showToast('Lỗi ' + res.status + ': ' + text.substring(0, 200), 'error');
      return;
    }

    const data = await res.json();
    console.log('lead-status API response:', data);

    if (data.ok) {
      showToast(`Đã cập nhật status thành: ${status}`);
    } else {
      showToast('Lỗi: ' + (data.error || 'Không rõ'), 'error');
    }
  } catch (err) {
    console.error('lead-status fetch error:', err);
    showToast('Lỗi kết nối: ' + err.message, 'error');
  }
}

// Toast helper
function showToast(msg, type = 'success') {
  let toast = document.getElementById('apex-toast');
  if (!toast) {
    toast = document.createElement('div');
    toast.id = 'apex-toast';
    toast.style.cssText = 'position:fixed;top:20px;right:20px;z-index:99999;padding:12px 24px;border-radius:8px;font-weight:700;box-shadow:0 4px 12px rgba(0,0,0,0.15);transition:opacity .3s';
    document.body.appendChild(toast);
  }
  toast.style.background = type === 'error' ? 'var(--red)' : 'var(--green)';
  toast.style.color = '#fff';
  toast.textContent = msg;
  toast.style.opacity = '1';
  setTimeout(() => { toast.style.opacity = '0'; }, 3000);
}

// Lead Detail Modal
function showLeadDetail(id) {
  const modal = document.getElementById('leadDetailModal');
  const content = document.getElementById('leadDetailContent');
  const nameEl = document.getElementById('leadDetailName');
  modal.style.display = 'flex';
  document.body.style.overflow = 'hidden';

  content.innerHTML = '<div style="text-align:center;padding:40px;color:#94a3b8"><div style="font-size:24px;margin-bottom:8px">⏳</div><div>Đang tải dữ liệu...</div></div>';

  fetch(adminBase + '/api/lead-detail.php?id=' + id)
    .then(r => r.json())
    .then(data => {
      if (!data.ok) {
        content.innerHTML = '<div style="color:var(--red);padding:20px">Lỗi: ' + (data.error || 'Không thể tải dữ liệu') + '</div>';
        return;
      }
      const lead = data.lead;
      nameEl.textContent = 'ID: ' + lead.id + ' | SĐT: ' + (lead.phone || '—');

      let html = '<div style="display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:16px">';

      // Lead Info Card
      html += '<div style="background:rgba(255,255,255,0.05);border-radius:12px;padding:16px">';
      html += '<div style="font-weight:700;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid rgba(255,255,255,0.1);color:#00d4ff">📋 THÔNG TIN LEAD</div>';
      html += renderField('ID', lead.id);
      html += renderField('Họ tên', lead.name);
      html += renderField('SĐT', lead.phone);
      html += renderField('Form', lead.form_type);
      html += renderField('Status', lead.status);
      html += renderField('Thời gian', formatDate(lead.created_at));
      html += renderField('IP', lead.ip_hash ? lead.ip_hash.substring(0,8) + '...' : lead.ip);
      html += renderField('Device', lead.device_type);
      html += '</div>';

      // UTM Card
      html += '<div style="background:rgba(255,255,255,0.05);border-radius:12px;padding:16px">';
      html += '<div style="font-weight:700;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid rgba(255,255,255,0.1);color:#f59e0b">🎯 UTM TRACKING</div>';
      html += renderField('Source', lead.utm_source || lead.source);
      html += renderField('Medium', lead.utm_medium || lead.medium);
      html += renderField('Campaign', lead.utm_campaign || lead.campaign);
      html += renderField('Content', lead.utm_content || lead.content);
      html += renderField('Term', lead.utm_term || lead.term);
      html += renderField('FB Click ID', lead.fbclid);
      html += renderField('G Click ID', lead.gclid);
      html += '</div>';

      // Session & Stats Card
      if (data.session) {
        html += '<div style="background:rgba(255,255,255,0.05);border-radius:12px;padding:16px">';
        html += '<div style="font-weight:700;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid rgba(255,255,255,0.1);color:#10b981">📊 SESSION</div>';
        html += renderField('Session Start', formatDate(data.session.first_visit));
        html += renderField('Last Activity', formatDate(data.session.last_activity));
        html += renderField('Duration', data.session.total_duration ? formatDuration(data.session.total_duration) : '—');
        html += renderField('Pages Visited', data.session.page_views);
        html += renderField('Device', data.session.device_type);
        html += renderField('Browser', data.session.browser_name);
        html += renderField('OS', data.session.os_name);
        html += renderField('Country', data.session.country);
        html += renderField('City', data.session.city);
        html += '</div>';
      }

      // Events Timeline
      if (data.events && data.events.length > 0) {
        html += '<div style="grid-column:1/-1;background:rgba(255,255,255,0.05);border-radius:12px;padding:16px">';
        html += '<div style="font-weight:700;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid rgba(255,255,255,0.1);color:#ffd700">📜 TRACKING EVENTS (' + data.events.length + ')</div>';
        html += '<div style="background:rgba(0,0,0,0.3);border-radius:8px;padding:8px;max-height:300px;overflow-y:auto">';

        data.events.forEach((ev, i) => {
          const typeColor = ev.event_type === 'page_view' ? '#3b82f6' : ev.event_type === 'lead' ? '#10b981' : '#f59e0b';
          html += '<div style="padding:8px 12px;border-bottom:1px solid rgba(255,255,255,0.05)">';
          html += '<span style="background:' + typeColor + ';color:#fff;padding:2px 8px;border-radius:4px;font-size:11px;margin-right:8px">' + ev.event_type + '</span>';
          html += '<span style="color:#94a3b8;font-size:12px">' + formatDate(ev.created_at) + '</span>';
          html += '</div>';
        });

        html += '</div></div>';
      }

      html += '</div>';
      content.innerHTML = html;
    })
    .catch(err => {
      content.innerHTML = '<div style="color:var(--red);padding:20px">Lỗi kết nối: ' + err.message + '</div>';
    });
}

function closeLeadDetail() {
  document.getElementById('leadDetailModal').style.display = 'none';
  document.body.style.overflow = '';
}

function renderField(label, value) {
  if (!value && value !== 0) return '';
  return '<div style="margin-bottom:8px"><span style="color:#94a3b8;font-size:12px">' + label + ':</span> <span style="color:#e2e8f0;font-size:13px;word-break:break-all">' + escapeHtml(String(value)) + '</span></div>';
}

function escapeHtml(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}

function formatDate(dt) {
  if (!dt) return '—';
  try {
    const d = new Date(String(dt).replace(' ', 'T'));
    return d.toLocaleString('vi-VN', {day:'2-digit',month:'2-digit',year:'numeric',hour:'2-digit',minute:'2-digit'});
  } catch(e) { return dt; }
}

function formatDuration(seconds) {
  if (!seconds) return '—';
  const h = Math.floor(seconds / 3600);
  const m = Math.floor((seconds % 3600) / 60);
  const s = seconds % 60;
  if (h > 0) return h + 'h ' + m + 'm';
  if (m > 0) return m + 'm ' + s + 's';
  return s + 's';
}

// Admin base path
const adminBase = '<?= defined("BASE_PATH") ? rtrim(BASE_PATH, "/") : "" ?>/admin';

// Expose functions
window.updateBulkUI = updateBulkUI;
window.showLeadDetail = showLeadDetail;
window.closeLeadDetail = closeLeadDetail;
window.updateLeadStatus = updateLeadStatus;
window.showToast = showToast;

// Init realtime polling
if (typeof ApexRealtimeDashboard !== 'undefined') {
  window.apexRealtime = new ApexRealtimeDashboard({ pollInterval: 3000 });
  window.apexRealtime.start();
}
</script>

<style>
.row-duplicate {
  background: rgba(245, 158, 11, 0.08) !important;
}
.row-duplicate:hover {
  background: rgba(245, 158, 11, 0.15) !important;
}
.text-orange {
  color: #f59e0b;
  font-weight: 600;
}
</style>

<?php require __DIR__ . '/includes/footer.php'; ?>
