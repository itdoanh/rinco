<?php
/**
 * APEX Admin - Dashboard v2 (PHP 8.4 Optimized)
 *
 * SỬ DỤNG:
 * - Summary Table (summary_daily) cho KPIs
 * - APCu caching với TTL ngắn
 * - Stats API endpoint cho data loading
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';
require_once __DIR__ . '/includes/queries.php';
require_once __DIR__ . '/../includes/cache.php';

$user = apex_admin_require();
$range = apex_parse_date_range();

$pageTitle = 'Tổng quan';
$pageSubtitle = sprintf('%s → %s (%d ngày)',
    date('d/m/Y', strtotime($range['from'])),
    date('d/m/Y', strtotime($range['to'])),
    $range['days']);
$showDateFilter = true;

require __DIR__ . '/includes/header.php';
?>

<div data-dashboard-stats data-live>
  <input type="hidden" data-stats-from="<?= h($range['from']) ?>" data-stats-to="<?= h($range['to']) ?>">

  <!-- ===== KPI TOP ===== -->
  <div class="metric-grid">
    <!-- Real visitors -->
    <div class="metric metric--blue metric--featured">
      <div class="metric__icon">👥</div>
      <div class="metric__label">Visitors thực (<?= $range['days'] ?> ngày)</div>
      <div class="metric__value" data-stat="visitors_total">
        <span class="metric-loader">...</span>
      </div>
      <span class="metric__delta metric__delta--up" data-stat="visitors_today">
        <span class="metric-loader">...</span> hôm nay
      </span>
    </div>

    <!-- FB Ads visitors -->
    <div class="metric metric--purple">
      <div class="metric__icon">📘</div>
      <div class="metric__label">Từ Facebook Ads</div>
      <div class="metric__value" data-stat="visitors_fb">
        <span class="metric-loader">...</span>
      </div>
      <span class="metric__delta metric__delta--up">
        <span class="metric-loader">...</span> hôm nay
      </span>
    </div>

    <!-- Leads -->
    <div class="metric metric--green">
      <div class="metric__icon">🎯</div>
      <div class="metric__label">Leads</div>
      <div class="metric__value" data-stat="leads_total">
        <span class="metric-loader">...</span>
      </div>
      <span class="metric__delta metric__delta--up" data-stat="leads_today">
        <span class="metric-loader">...</span> hôm nay
      </span>
    </div>

    <!-- Conversion Rate -->
    <div class="metric metric--blue">
      <div class="metric__icon">📊</div>
      <div class="metric__label">Conv. Rate</div>
      <div class="metric__value" data-stat="conv_rate">
        <span class="metric-loader">...</span>%
      </div>
      <span class="metric__delta metric__delta--neutral">
        <span class="metric-loader">...</span> từ FB Ads
      </span>
    </div>
  </div>

  <!-- ===== KPI ROW 2 ===== -->
  <div class="metric-grid">
    <div class="metric">
      <div class="metric__icon">📄</div>
      <div class="metric__label">Page Views</div>
      <div class="metric__value" data-stat="pageviews_total">
        <span class="metric-loader">...</span>
      </div>
    </div>

    <div class="metric">
      <div class="metric__icon">🔍</div>
      <div class="metric__label">Từ Google Ads</div>
      <div class="metric__value" data-stat="visitors_gg">
        <span class="metric-loader">...</span>
      </div>
    </div>

    <div class="metric">
      <div class="metric__icon">🔗</div>
      <div class="metric__label">Direct / Organic</div>
      <div class="metric__value" data-stat="visitors_direct">
        <span class="metric-loader">...</span>
      </div>
    </div>

    <div class="metric">
      <div class="metric__icon">⚡</div>
      <div class="metric__label">Active (5 phút)</div>
      <div class="metric__value" data-active-visitors>
        <?= apex_format_number(apex_query_active_visitors()) ?>
      </div>
    </div>

    <div class="metric">
      <div class="metric__icon">⏱️</div>
      <div class="metric__label">Avg Time</div>
      <div class="metric__value" data-stat="avg_duration">
        <span class="metric-loader">...</span>
      </div>
    </div>

    <div class="metric">
      <div class="metric__icon">📝</div>
      <div class="metric__label">Form Submits</div>
      <div class="metric__value" data-stat="form_submits_success">
        <span class="metric-loader">...</span>
      </div>
    </div>

    <div class="metric">
      <div class="metric__icon">📋</div>
      <div class="metric__label">Form Focuses</div>
      <div class="metric__value" data-stat="form_focuses_total">
        <span class="metric-loader">...</span>
      </div>
    </div>

    <div class="metric metric--red">
      <div class="metric__icon">⚠️</div>
      <div class="metric__label">Errors</div>
      <div class="metric__value" data-stat="errors_total">
        <span class="metric-loader">...</span>
      </div>
    </div>
  </div>

  <!-- ===== TIMELINE CHART ===== -->
  <div class="panel">
    <div class="panel__head">
      <div class="panel__title">📈 Biểu đồ truy cập & leads</div>
      <div class="panel__actions text-sm text-muted">
        Real visitors · <?= $range['days'] ?> ngày
      </div>
    </div>
    <div style="height:280px;position:relative">
      <canvas id="chartTimeline"></canvas>
    </div>
  </div>

  <!-- ===== FUNNEL ===== -->
  <div class="panel">
    <div class="panel__head">
      <div class="panel__title">🔻 Phễu chuyển đổi</div>
      <div class="panel__actions text-sm text-muted">
        <?= $range['days'] ?> ngày qua
      </div>
    </div>
    <div class="funnel" id="funnelChart">
      <!-- Loaded via JS from API -->
    </div>
  </div>

  <!-- ===== TRAFFIC SOURCES ===== -->
  <div class="grid-2">
    <div class="panel">
      <div class="panel__head">
        <div class="panel__title">🍩 Phân bổ nguồn traffic</div>
      </div>
      <div style="height:280px;position:relative">
        <canvas id="chartSources"></canvas>
      </div>
    </div>

    <div class="panel">
      <div class="panel__head">
        <div class="panel__title">🌐 Traffic Sources</div>
        <a href="<?= h(apex_admin_url('leads.php')) ?>" class="text-sm">Chi tiết →</a>
      </div>
      <div id="trafficSourcesList">
        <!-- Loaded via JS -->
        <div class="empty">Đang tải...</div>
      </div>
    </div>
  </div>

  <!-- ===== RECENT LEADS ===== -->
  <div class="panel">
    <div class="panel__head">
      <div class="panel__title">📋 Leads mới nhất</div>
      <div class="panel__actions">
        <a href="<?= h(apex_admin_url('leads.php')) ?>" class="btn btn--sm btn--ghost">📥 Export CSV</a>
        <a href="<?= h(apex_admin_url('leads.php')) ?>" class="btn btn--sm">Xem tất cả →</a>
      </div>
    </div>
    <div class="table-wrap">
      <table class="data">
        <thead>
          <tr>
            <th>Thời gian</th>
            <th>Họ tên</th>
            <th>SĐT</th>
            <th>Form</th>
            <th>Nguồn</th>
            <th>Device</th>
            <th></th>
          </tr>
        </thead>
        <tbody id="recentLeadsBody">
          <!-- Loaded via API -->
          <tr><td colspan="7" class="empty">Đang tải...</td></tr>
        </tbody>
      </table>
    </div>
  </div>
</div>

<script>
document.addEventListener('DOMContentLoaded', async function() {
  // Load stats from API
  const params = new URLSearchParams({
    from: '<?= $range['from'] ?>',
    to: '<?= $range['to'] ?>'
  });

  try {
    const res = await fetch('<?= rtrim(BASE_PATH, '/') ?>/api/stats.php?' + params);
    console.log('Stats API status:', res.status, res.statusText);

    if (!res.ok) {
      const text = await res.text();
      console.error('Stats API error response:', text);
      document.querySelectorAll('.metric-loader').forEach(el => el.textContent = 'Lỗi ' + res.status);
      return;
    }

    const data = await res.json();
    console.log('Stats API data:', data);

    if (data.ok) {
      updateStatsUI(data);
      updateChart(data);
      updateFunnel(data);
      updateTrafficSources(data);
      loadRecentLeads();
    } else {
      console.error('Stats API returned ok=false:', data);
    }
  } catch (err) {
    console.error('Failed to load stats:', err);
    document.querySelectorAll('.metric-loader').forEach(el => el.textContent = 'Lỗi mạng');
  }
});

function updateStatsUI(data) {
  const s = data.summary || {};
  const t = data.today || {};

  // Update metric values
  setStat('visitors_total', formatNumber(s.visitors_total || 0));
  setStat('visitors_today', formatNumber(t.visitors || 0));
  setStat('visitors_fb', formatNumber(s.visitors_fb || 0));
  setStat('visitors_gg', formatNumber(s.visitors_gg || 0));
  setStat('visitors_direct', formatNumber(s.visitors_direct || 0));
  setStat('leads_total', formatNumber(s.leads_total || 0));
  setStat('leads_today', formatNumber(t.leads || 0));
  setStat('conv_rate', (s.conv_rate || 0).toFixed(2));
  setStat('pageviews_total', formatNumber(s.pageviews_total || 0));
  setStat('avg_duration', formatDuration(s.avg_duration_seconds || 0));
  setStat('form_submits_success', formatNumber(s.form_submits_success || 0));
  setStat('form_focuses_total', formatNumber(s.form_focuses_total || 0));
  setStat('errors_total', formatNumber(s.errors_total || 0));
}

function setStat(key, value) {
  const el = document.querySelector('[data-stat="' + key + '"]');
  if (el) {
    el.innerHTML = value;
  }
}

function updateChart(data) {
  const ts = data.time_series || [];
  if (!ts.length) return;

  const ctx = document.getElementById('chartTimeline');
  if (!ctx) return;

  // Destroy existing chart
  if (window.timelineChart) {
    window.timelineChart.destroy();
  }

  window.timelineChart = new Chart(ctx, {
    type: 'line',
    data: {
      labels: ts.map(d => d.label),
      datasets: [
        {
          label: 'Visitors',
          data: ts.map(d => d.visitors),
          borderColor: '#FF6B00',
          backgroundColor: 'rgba(255, 107, 0, 0.15)',
          fill: true,
          tension: 0.35,
          borderWidth: 2,
        },
        {
          label: 'Từ Facebook',
          data: ts.map(d => d.visitors_fb),
          borderColor: '#1877F2',
          backgroundColor: 'rgba(24, 119, 242, 0.1)',
          fill: true,
          tension: 0.35,
        },
        {
          label: 'Leads',
          data: ts.map(d => d.leads),
          borderColor: '#10B981',
          fill: false,
          tension: 0.35,
          yAxisID: 'y1',
        }
      ]
    },
    options: {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: 'index', intersect: false },
      plugins: {
        legend: { position: 'top', labels: { color: '#94a3b8' } },
      },
      scales: {
        x: { ticks: { color: '#64748b' }, grid: { color: 'rgba(148, 163, 184, 0.08)' } },
        y: { ticks: { color: '#64748b' }, grid: { color: 'rgba(148, 163, 184, 0.08)' } },
        y1: { position: 'right', ticks: { color: '#10B981' }, grid: { drawOnChartArea: false } }
      }
    }
  });
}

function updateFunnel(data) {
  const f = data.funnel || {};
  const container = document.getElementById('funnelChart');
  if (!container) return;

  const steps = [
    { label: 'Visitors', key: 'visitors', color: '#0A192F' },
    { label: 'CTA Click', key: 'cta_clicks', color: '#F5A623' },
    { label: 'Form Focus', key: 'form_focuses', color: '#EA580C' },
    { label: 'Form Submit', key: 'form_submits_attempts', color: '#DC2626' },
    { label: 'Success', key: 'form_submits_success', color: '#10B981' },
    { label: 'Leads', key: 'leads', color: '#059669' },
  ];

  let html = '';
  const maxVal = f.visitors || 1;

  steps.forEach((step, i) => {
    const val = f[step.key] || 0;
    const pct = (val / maxVal) * 100;
    const conv = (val / maxVal) * 100;

    html += `
      <div class="funnel-step" style="border-left-color: ${step.color}">
        <div class="funnel-step__label">${step.label}</div>
        <div class="funnel-step__bar">
          <div class="funnel-step__fill" style="width: ${pct}%; background: ${step.color}">
            ${formatNumber(val)}
          </div>
        </div>
        <div class="funnel-step__stats">
          <strong>${conv.toFixed(1)}%</strong>
        </div>
      </div>
    `;
  });

  container.innerHTML = html;
}

function updateTrafficSources(data) {
  const traffic = data.traffic || {};
  const container = document.getElementById('trafficSourcesList');
  if (!container) return;

  const sources = [
    { label: 'Facebook Ads', value: traffic.fb_percent || 0, icon: '📘', color: '#1877F2' },
    { label: 'Google Ads', value: traffic.gg_percent || 0, icon: '🔍', color: '#4285F4' },
    { label: 'Direct', value: traffic.direct_percent || 0, icon: '🔗', color: '#94a3b8' },
  ];

  let html = '';
  sources.forEach(s => {
    html += `
      <div class="chart-bar-row">
        <div class="chart-bar-row__label">${s.icon} ${s.label}</div>
        <div class="chart-bar-row__bar">
          <div class="chart-bar-row__fill" style="width: ${s.value}%; background: ${s.color}"></div>
        </div>
        <div class="chart-bar-row__pct">${s.value.toFixed(1)}%</div>
      </div>
    `;
  });

  container.innerHTML = html;

  // Update pie chart
  const ctx = document.getElementById('chartSources');
  if (ctx && window.sourcesChart) {
    window.sourcesChart.destroy();
  }
  if (ctx) {
    window.sourcesChart = new Chart(ctx, {
      type: 'doughnut',
      data: {
        labels: sources.map(s => s.label),
        datasets: [{
          data: sources.map(s => s.value || 1),
          backgroundColor: sources.map(s => s.color),
          borderColor: '#0f172a',
          borderWidth: 2,
        }]
      },
      options: {
        responsive: true,
        maintainAspectRatio: false,
        plugins: {
          legend: { position: 'right', labels: { color: '#cbd5e1' } },
        }
      }
    });
  }
}

async function loadRecentLeads() {
  try {
    const res = await fetch('<?= rtrim(BASE_PATH, '/') ?>/api/leads-poll.php?limit=8&status=new');
    const data = await res.json();

    const tbody = document.getElementById('recentLeadsBody');
    if (!tbody || !data.ok) return;

    if (!data.leads || !data.leads.length) {
      tbody.innerHTML = '<tr><td colspan="7" class="empty">Chưa có leads nào.</td></tr>';
      return;
    }

    tbody.innerHTML = data.leads.map(lead => {
      const phoneZalo = (lead.phone || '').replace(/^0/, '+84');
      const formClass = lead.form_type === 'hero' ? 'pill--hero' : 'pill--footer';
      return `
        <tr>
          <td><small>${formatDate(lead.created_at)}</small></td>
          <td><strong>${escapeHtml(lead.name)}</strong></td>
          <td>
            <a href="tel:${escapeHtml(lead.phone)}" style="color:var(--orange);font-weight:600">${escapeHtml(lead.phone)}</a>
          </td>
          <td><span class="pill ${formClass}">${escapeHtml(lead.form_type || 'unknown')}</span></td>
          <td class="text-sm">${escapeHtml(lead.utm_source || '(direct)')}</td>
          <td class="text-sm text-muted">${escapeHtml(lead.device_type || '—')}</td>
          <td>
            <a href="https://zalo.me/${phoneZalo}" target="_blank" class="btn btn--xs btn--ghost">Zalo</a>
          </td>
        </tr>
      `;
    }).join('');

  } catch (err) {
    console.error('Failed to load recent leads:', err);
  }
}

// Utilities
function formatNumber(n) {
  return (n || 0).toLocaleString('vi-VN');
}

function formatDuration(seconds) {
  if (!seconds) return '0s';
  if (seconds < 60) return seconds + 's';
  if (seconds < 3600) return Math.floor(seconds / 60) + 'm ' + (seconds % 60) + 's';
  return Math.floor(seconds / 3600) + 'h ' + Math.floor((seconds % 3600) / 60) + 'm';
}

function formatDate(dt) {
  if (!dt) return '—';
  try {
    const d = new Date(String(dt).replace(' ', 'T'));
    return d.toLocaleString('vi-VN', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' });
  } catch (e) { return dt; }
}

function escapeHtml(str) {
  if (!str) return '';
  const d = document.createElement('div');
  d.textContent = str;
  return d.innerHTML;
}
</script>

<style>
.metric-loader {
  display: inline-block;
  width: 40px;
  height: 16px;
  background: linear-gradient(90deg, #1e293b 25%, #334155 50%, #1e293b 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
  border-radius: 4px;
}
@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>

<?php require __DIR__ . '/includes/footer.php'; ?>
