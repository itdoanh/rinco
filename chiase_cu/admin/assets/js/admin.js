/* =================================================================
   APEX Admin - Dashboard JS
   ================================================================= */
(function () {
  'use strict';

  const APEX = window.APEX_ADMIN || {};
  // CSRF token: ưu tiên cookie đã set bởi header.php, fallback window.APEX_ADMIN.csrf
  function readCookie(name) {
    const m = document.cookie.match(new RegExp('(?:^|; )' + name + '=([^;]*)'));
    return m ? decodeURIComponent(m[1]) : '';
  }
  const csrfToken = readCookie('apex_admin_csrf') || APEX.csrf || '';
  const refreshInterval = APEX.refreshInterval || 30000;

  // ===== TOAST =====
  function toast(msg, type = 'info', duration = 3500) {
    const c = document.getElementById('toastContainer');
    if (!c) return alert(msg);
    const el = document.createElement('div');
    el.className = 'toast toast--' + type;
    el.textContent = msg;
    c.appendChild(el);
    setTimeout(() => {
      el.style.opacity = '0';
      el.style.transform = 'translateX(100%)';
      el.style.transition = 'all 0.3s';
      setTimeout(() => el.remove(), 300);
    }, duration);
  }
  window.apexToast = toast;

  // ===== AUTO-REFRESH DASHBOARD =====
  function autoRefresh() {
    const ind = document.getElementById('liveIndicator');
    if (!ind || !document.querySelector('[data-autorefresh]')) return;

    let last = Date.now();
    const tick = () => {
      const idle = Date.now() - last;
      if (idle > refreshInterval * 2) {
        // User inactive > 2x interval - reload
        window.location.reload();
        return;
      }
    };
    // Reset idle timer on user interaction
    ['click', 'keydown', 'scroll', 'mousemove'].forEach(ev => {
      document.addEventListener(ev, () => { last = Date.now(); }, { passive: true });
    });
    setInterval(tick, refreshInterval);
  }
  autoRefresh();

  // ===== LIVE STATS PULLER =====
  async function pullLiveStats() {
    try {
      const adminBase = (window.APEX_ADMIN_BASE || '/chiase/admin');
      const res = await fetch(adminBase + '/api/stats.php?live=1', {
        credentials: 'same-origin',
        headers: { 'X-CSRF-Token': csrfToken }
      });
      if (!res.ok) {
        console.debug('[pullLiveStats] non-OK status:', res.status);
        return;
      }
      const data = await res.json();
      if (data && data.ok === false) {
        console.warn('[pullLiveStats] API error:', data.error, data);
        return;
      }

      // Update active visitors badge
      const activeEl = document.querySelector('[data-active-visitors]');
      if (activeEl && data.active_visitors != null) {
        activeEl.textContent = data.active_visitors;
        activeEl.classList.add('pulse-anim');
        setTimeout(() => activeEl.classList.remove('pulse-anim'), 800);
      }

      // Update today's leads badge in nav
      const navBadge = document.querySelector('.sidebar .nav-badge');
      if (navBadge && data.leads_today != null) {
        navBadge.textContent = data.leads_today;
        navBadge.style.display = data.leads_today > 0 ? '' : 'none';
      }
    } catch (_) { /* silent fail */ }
  }
  if (document.querySelector('[data-live]')) {
    pullLiveStats();
    setInterval(pullLiveStats, refreshInterval);
  }

  // ===== TABLE FILTERS / SEARCH =====
  document.querySelectorAll('[data-table-filter]').forEach(input => {
    const target = document.querySelector(input.dataset.tableFilter);
    if (!target) return;
    input.addEventListener('input', () => {
      const q = input.value.trim().toLowerCase();
      target.querySelectorAll('tbody tr').forEach(row => {
        row.style.display = row.textContent.toLowerCase().includes(q) ? '' : 'none';
      });
    });
  });

  // ===== COPY-TO-CLIPBOARD =====
  document.querySelectorAll('[data-copy]').forEach(el => {
    el.style.cursor = 'pointer';
    el.title = 'Click để copy';
    el.addEventListener('click', () => {
      navigator.clipboard.writeText(el.dataset.copy || el.textContent)
        .then(() => toast('Đã copy: ' + el.textContent, 'success', 1500))
        .catch(() => toast('Copy thất bại', 'error'));
    });
  });

  // ===== CONFIRM DELETE =====
  document.querySelectorAll('[data-confirm]').forEach(el => {
    el.addEventListener('click', (e) => {
      const msg = el.dataset.confirm || 'Bạn có chắc?';
      if (!confirm(msg)) { e.preventDefault(); e.stopPropagation(); }
    });
  });

  // ===== DRAW LINE CHART (inline SVG) =====
  window.apexLineChart = function (canvas, data, options = {}) {
    const opts = Object.assign({
      padding: 30,
      strokeColor: '#FF6B00',
      fillColor: 'rgba(255, 107, 0, 0.08)',
      dotColor: '#FF6B00',
      gridColor: '#E2E8F0',
      textColor: '#94A3B8',
      formatValue: (v) => v.toLocaleString(),
      formatLabel: (l) => l,
    }, options);

    if (!canvas || !data || data.length === 0) return;

    const W = canvas.clientWidth;
    const H = canvas.clientHeight;
    const dpr = window.devicePixelRatio || 1;
    canvas.width = W * dpr;
    canvas.height = H * dpr;
    canvas.style.width = W + 'px';
    canvas.style.height = H + 'px';
    const ctx = canvas.getContext('2d');
    ctx.scale(dpr, dpr);

    const pad = opts.padding;
    const w = W - pad * 2;
    const h = H - pad * 2;

    const max = Math.max(...data.map(d => d.value), 1);
    const min = 0;
    const stepX = data.length > 1 ? w / (data.length - 1) : 0;

    // Grid
    ctx.strokeStyle = opts.gridColor;
    ctx.lineWidth = 1;
    ctx.font = '10px -apple-system, sans-serif';
    ctx.fillStyle = opts.textColor;
    ctx.textAlign = 'right';
    for (let i = 0; i <= 4; i++) {
      const y = pad + (h / 4) * i;
      const val = max - (max / 4) * i;
      ctx.beginPath();
      ctx.moveTo(pad, y);
      ctx.lineTo(W - pad, y);
      ctx.stroke();
      ctx.fillText(opts.formatValue(val), pad - 5, y + 3);
    }

    // X labels (every nth to avoid overlap)
    ctx.textAlign = 'center';
    const labelStep = Math.max(1, Math.ceil(data.length / 7));
    data.forEach((d, i) => {
      if (i % labelStep === 0 || i === data.length - 1) {
        const x = pad + stepX * i;
        ctx.fillText(opts.formatLabel(d.label), x, H - 8);
      }
    });

    // Area
    ctx.beginPath();
    ctx.moveTo(pad, pad + h);
    data.forEach((d, i) => {
      const x = pad + stepX * i;
      const y = pad + h - (d.value / max) * h;
      if (i === 0) ctx.lineTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.lineTo(pad + stepX * (data.length - 1), pad + h);
    ctx.closePath();
    ctx.fillStyle = opts.fillColor;
    ctx.fill();

    // Line
    ctx.beginPath();
    data.forEach((d, i) => {
      const x = pad + stepX * i;
      const y = pad + h - (d.value / max) * h;
      if (i === 0) ctx.moveTo(x, y);
      else ctx.lineTo(x, y);
    });
    ctx.strokeStyle = opts.strokeColor;
    ctx.lineWidth = 2.5;
    ctx.stroke();

    // Dots
    data.forEach((d, i) => {
      const x = pad + stepX * i;
      const y = pad + h - (d.value / max) * h;
      ctx.beginPath();
      ctx.arc(x, y, 3, 0, Math.PI * 2);
      ctx.fillStyle = '#fff';
      ctx.fill();
      ctx.strokeStyle = opts.dotColor;
      ctx.lineWidth = 2;
      ctx.stroke();
    });
  };

  // ===== DRAW FUNNEL (handled by CSS, JS just animates) =====
  document.querySelectorAll('.funnel-step__fill').forEach(el => {
    const w = el.dataset.width || '0%';
    el.style.width = '0%';
    setTimeout(() => { el.style.width = w; }, 100);
  });

  // ===== KEYBOARD SHORTCUTS =====
  document.addEventListener('keydown', (e) => {
    // Ctrl+K or / - focus search
    if ((e.ctrlKey && e.key === 'k') || (e.key === '/' && document.activeElement.tagName !== 'INPUT')) {
      const search = document.querySelector('[data-table-filter]');
      if (search) { e.preventDefault(); search.focus(); }
    }
    // Esc - clear search
    if (e.key === 'Escape') {
      const search = document.querySelector('[data-table-filter]');
      if (search && document.activeElement === search) {
        search.value = '';
        search.dispatchEvent(new Event('input'));
      }
    }
  });

  // ===== SIDEBAR TOGGLE (mobile) =====
  const menuBtn = document.querySelector('[data-menu-btn]');
  const sidebar = document.querySelector('.sidebar');
  if (menuBtn && sidebar) {
    menuBtn.addEventListener('click', () => sidebar.classList.toggle('is-open'));
  }
})();
