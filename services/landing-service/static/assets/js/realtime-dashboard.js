/**
 * APEX Admin - Realtime Dashboard JS
 *
 * Polling class cho CRM real-time updates.
 * Sử dụng "Short Polling" với WHERE id > last_id
 *
 * Cài đặt crontab:
 * */5 * * * * php /home/hanghoap/public_html/chiase/cron/build_summary_daily.php
 */

class ApexRealtimeDashboard {
    constructor(options = {}) {
        this.pollInterval = options.pollInterval || 3000;    // 3 seconds default
        this.statsInterval = options.statsInterval || 30000; // 30 seconds for stats
        this.maxRetries = options.maxRetries || 3;
        this.retryDelay = options.retryDelay || 5000;

        this.lastLeadId = 0;
        this.lastEventId = 0;
        this.timer = null;
        this.statsTimer = null;
        this.isConnected = false;
        this.retryCount = 0;

        // Ưu tiên window.APEX_BASE được PHP render, fallback '/chiase' cho hosting
        this.basePath = window.APEX_BASE || '/chiase';
        this.adminPath = (window.APEX_ADMIN_BASE || (this.basePath + '/admin'));
    }

    // Get current max lead ID from DOM
    getCurrentMaxLeadId() {
        const inputs = document.querySelectorAll('#leadsTable input[name="ids[]"]');
        if (!inputs.length) return 0;
        return Math.max(...Array.from(inputs).map(i => parseInt(i.value) || 0));
    }

    getCurrentMaxEventId() {
        const rows = document.querySelectorAll('#eventsTable tbody tr[data-event-id]');
        if (!rows.length) return 0;
        return Math.max(...Array.from(rows).map(r => parseInt(r.dataset.eventId) || 0));
    }

    // Start polling
    start() {
        console.log('[ApexRealtime] Starting...');
        this.lastLeadId = this.getCurrentMaxLeadId();
        this.lastEventId = this.getCurrentMaxEventId();

        this.poll(); // Initial poll
        this.timer = setInterval(() => this.poll(), this.pollInterval);

        // Stats refresh (dashboard KPIs)
        if (document.querySelector('[data-dashboard-stats]')) {
            this.refreshStats();
            this.statsTimer = setInterval(() => this.refreshStats(), this.statsInterval);
        }

        this.isConnected = true;
    }

    // Stop polling
    stop() {
        clearInterval(this.timer);
        clearInterval(this.statsTimer);
        this.isConnected = false;
        console.log('[ApexRealtime] Stopped');
    }

    // Main polling function
    async poll() {
        try {
            // Poll leads
            await this.pollLeads();

            // Poll events (if on events page)
            if (document.querySelector('#eventsTable')) {
                await this.pollEvents();
            }

            // Reset retry count on success
            this.retryCount = 0;

        } catch (err) {
            console.warn('[ApexRealtime] Poll error:', err);
            this.handleError();
        }
    }

    // Poll for new leads
    async pollLeads() {
        const url = `${this.adminPath}/api/leads-poll.php?after_id=${this.lastLeadId}&limit=20`;

        const res = await fetch(url);
        if (!res.ok) throw new Error(`HTTP ${res.status}`);

        const data = await res.json();

        if (!data.ok) {
            throw new Error(data.error || 'Poll failed');
        }

        if (data.new_count > 0) {
            console.log(`[ApexRealtime] ${data.new_count} new leads`);

            // Add new leads to table
            this.addNewLeads(data.leads);

            // Update lastLeadId
            if (data.leads.length > 0) {
                const lastLead = data.leads[data.leads.length - 1];
                this.lastLeadId = Math.max(this.lastLeadId, lastLead.id);
            }

            // Show notification
            this.showToast(`${data.new_count} lead mới!`, 'success');
            this.playSound();
        }
    }

    // Poll for new events
    async pollEvents() {
        const url = `${this.adminPath}/api/events-poll.php?after_id=${this.lastEventId}&limit=20`;

        const res = await fetch(url);
        if (!res.ok) return;

        const data = await res.json();

        if (data.ok && data.new_count > 0) {
            // Add new events
            this.addNewEvents(data.events);

            if (data.events.length > 0) {
                const lastEvent = data.events[data.events.length - 1];
                this.lastEventId = Math.max(this.lastEventId, parseInt(lastEvent.id));
            }
        }
    }

    // Refresh dashboard stats
    async refreshStats() {
        const params = new URLSearchParams({
            from: document.querySelector('[data-stats-from]')?.dataset.statsFrom || this.getDefaultFrom(),
            to: document.querySelector('[data-stats-to]')?.dataset.statsTo || this.getDefaultTo()
        });

        const url = `${this.basePath}/api/stats.php?${params}`;

        try {
            const res = await fetch(url);
            if (!res.ok) return;

            const data = await res.json();

            if (data.ok) {
                this.updateStatsUI(data);
            }
        } catch (err) {
            console.warn('[ApexRealtime] Stats refresh error:', err);
        }
    }

    // Add new leads to table
    addNewLeads(leads) {
        const tbody = document.querySelector('#leadsTable tbody');
        if (!tbody) return;

        // Remove empty row if present
        const emptyRow = tbody.querySelector('tr td.empty');
        if (emptyRow) emptyRow.closest('tr').remove();

        leads.forEach((lead, index) => {
            const tr = this.createLeadRow(lead);
            tr.classList.add('new-row');

            // Insert at top
            tbody.insertBefore(tr, tbody.firstChild);

            // Highlight animation
            setTimeout(() => {
                tr.classList.add('highlight');
            }, 100);

            // Remove highlight after 3 seconds
            setTimeout(() => {
                tr.classList.remove('new-row', 'highlight');
                const small = tr.querySelector('small.text-muted');
                if (small) small.textContent = this.timeAgo(lead.created_at);
            }, 3000);
        });

        // Update counter
        this.updateLeadCounter(leads.length);
    }

    // Create lead row HTML
    createLeadRow(lead) {
        const tr = document.createElement('tr');
        tr.dataset.leadId = lead.id;

        const phoneZalo = (lead.phone || '').replace(/^0/, '+84');
        const formClass = lead.form_type === 'hero' ? 'pill--hero' : 'pill--footer';
        const statusClass = this.getStatusClass(lead.status);

        tr.innerHTML = `
            <td><input type="checkbox" name="ids[]" value="${lead.id}" class="row-check"></td>
            <td>
                <div>${this.formatDate(lead.created_at)}</div>
                <small class="text-muted">vừa xong</small>
            </td>
            <td><strong>${this.escapeHtml(lead.name)}</strong></td>
            <td>
                <a href="tel:${this.escapeHtml(lead.phone)}" style="color:var(--orange);font-weight:600">${this.escapeHtml(lead.phone)}</a>
                <button type="button" class="btn btn--xs btn--ghost" onclick="navigator.clipboard.writeText('${this.escapeHtml(lead.phone)}')">📋</button>
            </td>
            <td><span class="pill ${formClass}">${this.escapeHtml(lead.form_type || 'unknown')}</span></td>
            <td><span class="pill-status ${statusClass}">${this.escapeHtml(lead.status || 'new')}</span></td>
            <td>${this.renderDuplicate(lead)}</td>
            <td class="text-sm">${this.escapeHtml(lead.utm_source || '(direct)')}</td>
            <td class="text-sm text-muted">${this.escapeHtml(lead.device_type || '—')}</td>
            <td class="text-sm" style="font-family:monospace">${lead.ip_hash ? lead.ip_hash.substring(0, 8) : '***'}</td>
            <td class="text-right">
                <a href="https://zalo.me/${phoneZalo}" target="_blank" class="btn btn--xs btn--ghost" title="Chat Zalo">💬</a>
                <a href="sms:${this.escapeHtml(lead.phone)}" class="btn btn--xs btn--ghost" title="SMS">✉️</a>
                <button type="button" class="btn btn--xs btn--ghost" onclick="window.showLeadDetail && window.showLeadDetail(${lead.id})">👁️</button>
                <button type="button" class="btn btn--xs btn--ghost" style="color:var(--red)">🗑️</button>
            </td>
        `;

        // Rebind checkbox
        const cb = tr.querySelector('.row-check');
        if (cb) cb.addEventListener('change', window.updateBulkUI);

        return tr;
    }

    // Add new events to table
    addNewEvents(events) {
        const tbody = document.querySelector('#eventsTable tbody');
        if (!tbody) return;

        const emptyRow = tbody.querySelector('tr td.empty');
        if (emptyRow) emptyRow.closest('tr').remove();

        events.forEach(event => {
            const tr = this.createEventRow(event);
            tr.classList.add('new-row');
            tbody.insertBefore(tr, tbody.firstChild);

            setTimeout(() => tr.classList.remove('new-row'), 3000);
        });
    }

    // Create event row HTML
    createEventRow(event) {
        const tr = document.createElement('tr');
        tr.dataset.eventId = event.id;

        const typeIcon = this.getEventIcon(event.event_type);
        const typeColor = this.getEventColor(event.event_type);

        tr.innerHTML = `
            <td class="text-muted" style="font-family:monospace">#${event.id}</td>
            <td>
                <div>${this.formatDate(event.created_at)}</div>
                <small class="text-muted">vừa xong</small>
            </td>
            <td><span style="color:${typeColor}">${typeIcon} ${this.escapeHtml(event.event_type)}</span></td>
            <td>${event.event_value ? `<code>${this.escapeHtml(String(event.event_value).substring(0, 60))}</code>` : '—'}</td>
            <td><a href="${this.adminPath}/sessions.php?id=${this.escapeHtml(event.session_id)}" style="font-family:monospace;font-size:11px">${this.escapeHtml(event.session_id).substring(0, 8)}...</a></td>
            <td>${event.utm_source ? `<span class="pill">${this.escapeHtml(event.utm_source)}</span>` : '—'}</td>
            <td class="text-sm text-muted">${this.escapeHtml(event.device_type || '—')}</td>
            <td style="font-family:monospace;font-size:11px">${event.ip_hash ? event.ip_hash.substring(0, 8) : '***'}</td>
            <td><a href="${this.adminPath}/sessions.php?id=${this.escapeHtml(event.session_id)}" class="btn btn--xs btn--ghost">🧭</a></td>
        `;

        return tr;
    }

    // Update stats UI
    updateStatsUI(data) {
        const summary = data.summary || {};

        // Update visitors
        const visitorsEl = document.querySelector('[data-stat="visitors"]');
        if (visitorsEl) visitorsEl.textContent = this.formatNumber(summary.visitors_total || 0);

        // Update leads
        const leadsEl = document.querySelector('[data-stat="leads"]');
        if (leadsEl) leadsEl.textContent = this.formatNumber(summary.leads_total || 0);

        // Update conv rate
        const convEl = document.querySelector('[data-stat="conv_rate"]');
        if (convEl) convEl.textContent = (summary.conv_rate || 0) + '%';

        // Update today
        if (data.today) {
            const todayVisitorsEl = document.querySelector('[data-stat="visitors_today"]');
            if (todayVisitorsEl) todayVisitorsEl.textContent = this.formatNumber(data.today.visitors || 0);

            const todayLeadsEl = document.querySelector('[data-stat="leads_today"]');
            if (todayLeadsEl) todayLeadsEl.textContent = this.formatNumber(data.today.leads || 0);
        }
    }

    // Update lead counter
    updateLeadCounter(addedCount) {
        const counter = document.querySelector('.panel__head .text-sm strong');
        if (counter) {
            const current = parseInt(counter.textContent.replace(/\D/g, '')) || 0;
            counter.textContent = (current + addedCount).toLocaleString('vi-VN');
        }
    }

    // Handle polling errors
    handleError() {
        this.retryCount++;

        if (this.retryCount >= this.maxRetries) {
            console.warn('[ApexRealtime] Max retries reached, pausing...');
            this.stop();

            this.showToast('Mất kết nối realtime. Đang thử kết nối lại...', 'warning');

            // Retry after delay
            setTimeout(() => {
                this.retryCount = 0;
                this.start();
            }, this.retryDelay);
        }
    }

    // Show toast notification
    showToast(msg, type = 'info') {
        let toast = document.getElementById('apex-toast');
        if (!toast) {
            toast = document.createElement('div');
            toast.id = 'apex-toast';
            toast.style.cssText = `
                position:fixed;top:20px;right:20px;z-index:99999;
                padding:12px 24px;border-radius:8px;font-weight:700;
                box-shadow:0 8px 32px rgba(0,0,0,0.2);
                transition:opacity 0.3s;cursor:pointer;
                font-family:system-ui,sans-serif;font-size:14px;
            `;
            document.body.appendChild(toast);
        }

        const colors = {
            success: 'linear-gradient(135deg,#10b981,#059669)',
            error: 'linear-gradient(135deg,#dc2626,#b91c1c)',
            warning: 'linear-gradient(135deg,#f59e0b,#d97706)',
            info: 'linear-gradient(135deg,#3b82f6,#2563eb)'
        };

        toast.style.background = colors[type] || colors.info;
        toast.style.color = '#fff';
        toast.textContent = msg;
        toast.style.opacity = '1';

        toast.onclick = () => { toast.style.opacity = '0'; };

        setTimeout(() => { toast.style.opacity = '0'; }, 4000);
    }

    // Play notification sound
    playSound() {
        try {
            const ctx = new (window.AudioContext || window.webkitAudioContext)();
            const osc = ctx.createOscillator();
            const gain = ctx.createGain();

            osc.connect(gain);
            gain.connect(ctx.destination);

            osc.frequency.value = 880;
            osc.type = 'sine';
            gain.gain.setValueAtTime(0.2, ctx.currentTime);
            gain.gain.exponentialRampToValueAtTime(0.01, ctx.currentTime + 0.3);

            osc.start(ctx.currentTime);
            osc.stop(ctx.currentTime + 0.3);
        } catch (e) {
            // Silently fail if AudioContext not available
        }
    }

    // Utility functions
    escapeHtml(str) {
        if (!str) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }

    formatDate(dt) {
        if (!dt) return '—';
        try {
            const d = new Date(String(dt).replace(' ', 'T'));
            return d.toLocaleString('vi-VN', {
                day: '2-digit', month: '2-digit', year: 'numeric',
                hour: '2-digit', minute: '2-digit'
            });
        } catch (e) { return dt; }
    }

    timeAgo(dt) {
        if (!dt) return '—';
        try {
            const d = new Date(String(dt).replace(' ', 'T'));
            const diff = Math.floor((Date.now() - d) / 1000);
            if (diff < 60) return diff + 's ago';
            if (diff < 3600) return Math.floor(diff / 60) + 'm ago';
            if (diff < 86400) return Math.floor(diff / 3600) + 'h ago';
            return Math.floor(diff / 86400) + 'd ago';
        } catch (e) { return '—'; }
    }

    formatNumber(n) {
        return (n || 0).toLocaleString('vi-VN');
    }

    getStatusClass(status) {
        const classes = {
            new: 'pill-status--new',
            contacted: 'pill-status--contacted',
            qualified: 'pill-status--qualified',
            converted: 'pill-status--converted',
            invalid: 'pill-status--invalid',
            duplicate: 'pill-status--duplicate',
            review: 'pill-status--review'
        };
        return classes[status] || 'pill-status--new';
    }

    renderDuplicate(lead) {
        if (!lead.duplicate_of_id && !lead.duplicate_group_id) {
            return '<span class="text-muted">—</span>';
        }

        const icon = lead.status === 'duplicate' ? '🔁' : '🔄';
        const cls = lead.status === 'duplicate' ? 'text-muted' : 'text-orange';

        let html = `<span class="${cls}" title="Trùng với lead #${lead.duplicate_of_id || 'N/A'}">${icon}</span>`;
        if (lead.duplicate_of_id) {
            html += `<a href="#" onclick="window.showLeadDetail && window.showLeadDetail(${lead.duplicate_of_id}); return false;" class="text-xs" style="margin-left:4px">(#${lead.duplicate_of_id})</a>`;
        }
        return html;
    }

    getEventIcon(type) {
        const icons = {
            page_view: '📄',
            cta_click: '🖱️',
            form_focus: '📝',
            form_submit: '✅',
            form_submit_success: '✅',
            form_submit_error: '❌',
            scroll_depth: '⬇️',
            lead: '🎯',
            modal_open: '📋',
            modal_close: '📕',
            input_focus: '⌨️',
            input_blur: '👋',
            time_tick: '⏱️'
        };
        return icons[type] || '📌';
    }

    getEventColor(type) {
        if (type === 'lead' || type === 'form_submit_success') return 'var(--green)';
        if (type === 'cta_click') return 'var(--orange)';
        if (type === 'form_focus') return 'var(--purple)';
        if (type === 'form_submit_error') return 'var(--red)';
        return 'var(--text)';
    }

    getDefaultFrom() {
        return new URLSearchParams(window.location.search).get('from')
            || new Date(Date.now() - 29 * 86400000).toISOString().split('T')[0];
    }

    getDefaultTo() {
        return new URLSearchParams(window.location.search).get('to')
            || new Date().toISOString().split('T')[0];
    }
}

// Auto-initialize when DOM ready
document.addEventListener('DOMContentLoaded', function() {
    // Check if we're on a page that needs realtime updates
    if (document.querySelector('[data-live]')) {
        window.apexRealtime = new ApexRealtimeDashboard();
        window.apexRealtime.start();
    }
});
