/* ============================================
   APEX FINTECH - MAIN.JS v3 (Production-grade lead submission)
   Features:
     - 3 forms (hero / multi-step footer / modal popup)
     - Idempotency: mỗi lead có UUID v4 → retry không bao giờ tạo duplicate
     - 4-tier retry strategy:
        Tier 1: sendBeacon (fire-and-forget khi page unload)
        Tier 2: fetch keepalive (chạy nền)
        Tier 3: persistent queue (localStorage) + auto-flush mỗi 30s
        Tier 4: explicit flush khi user quay lại trang (pageshow)
     - Error detail tracking: HTTP status, error code, server response, network state
     - Optimistic UI: hiển thị success ngay, vẫn retry background
     - Click-triggered modal popup
     - Multi-step form with channel selection
     - Sticky CTA control
   ============================================ */

   (function(){
    'use strict';
  
    const ready = fn => document.readyState !== 'loading'
      ? fn()
      : document.addEventListener('DOMContentLoaded', fn);
  
    // === API endpoints ===
    // BASE_PATH: prefix thư mục deploy (vd '/chiase'). Cho phép override qua window.APEX_BASE.
    const BASE_PATH = window.APEX_BASE || '/chiase';
    const API = {
      LEAD: BASE_PATH + '/api/lead.php',
      ACK: BASE_PATH + '/api/lead-ack.php',
      RETRY: BASE_PATH + '/api/lead-retry.php'
    };
  
    // === Retry config ===
    const MAX_RETRIES = 5;
    const RETRY_DELAYS_MS = [3000, 8000, 20000, 45000, 90000]; // exponential-ish
    const QUEUE_FLUSH_INTERVAL_MS = 30000;
    const LEAD_REQUEST_TIMEOUT_MS = 5000;
    const ACK_REQUEST_TIMEOUT_MS = 2000;
  
    // === Lead retry queue ===
    const QUEUE_KEY = 'apex_lead_queue';
    const MAX_QUEUE_SIZE = 50;
    var queue = [];
    var flushTimer = null;
    var isFlushing = false;
  
    function init(){
      initMobileMenu();
      initModalPopup();
      initForms();
      initMultiStepForm();
      initSmoothScroll();
      initStickyCtaVisibility();
      initSeatsCounter();
      initGiftsCounter();
  
      // Background: flush pending lead queue khi load
      initRetryQueue();
    }
  
    /* ===========================================================
       1) MOBILE MENU
       =========================================================== */
    function initMobileMenu(){
      const toggle = document.getElementById('menuToggle');
      const menu = document.getElementById('mobileMenu');
      if (!toggle || !menu) return;
      toggle.addEventListener('click', () => menu.classList.remove('hidden'));
      menu.querySelectorAll('[data-close-mobile-menu]').forEach(el => {
        el.addEventListener('click', () => menu.classList.add('hidden'));
      });
    }
  
    /* ===========================================================
       2) MODAL POPUP
       =========================================================== */
    function initModalPopup(){
      const modal = document.getElementById('leadModal');
      if (!modal) return;
  
      let lastTrigger = null;
  
      const open = (trigger) => {
        lastTrigger = trigger;
        modal.classList.add('is-open');
        modal.setAttribute('aria-hidden', 'false');
        document.body.classList.add('modal-open');
        const ctaLabel = trigger?.dataset?.ctaLabel || trigger?.dataset?.modalTrigger || 'unknown';
        trackEvent('modal_open', ctaLabel);
        setTimeout(() => {
          const name = modal.querySelector('input[name="name"]');
          if (name) name.focus({ preventScroll: true });
        }, 250);
      };
  
      const close = () => {
        modal.classList.remove('is-open');
        modal.setAttribute('aria-hidden', 'true');
        document.body.classList.remove('modal-open');
        const form = modal.querySelector('form');
        const success = modal.querySelector('#modalSuccess');
        if (form) {
          form.style.display = '';
          form.reset();
        }
        if (success) success.classList.add('hidden');
      };
  
      document.querySelectorAll('[data-modal-trigger]').forEach(btn => {
        btn.addEventListener('click', e => {
          e.preventDefault();
          open(btn);
        });
      });
      modal.querySelectorAll('[data-close-modal]').forEach(el => {
        el.addEventListener('click', close);
      });
      document.addEventListener('keydown', e => {
        if (e.key === 'Escape' && modal.classList.contains('is-open')) close();
      });
      window.ApexModal = { open, close };
    }
  
    /* ===========================================================
       3) FORM HANDLING (HERO + MODAL)
       =========================================================== */
    function initForms(){
      const forms = [
        { id: 'heroForm',  type: 'hero',  nameField: 'heroName',  phoneField: 'heroPhone',  successId: 'heroSuccess'  },
        { id: 'modalForm', type: 'modal', nameField: 'modalName', phoneField: 'modalPhone', successId: 'modalSuccess' }
      ];
      forms.forEach(cfg => setupForm(cfg));
    }
  
    function setupForm(cfg){
      const form = document.getElementById(cfg.id);
      if (!form) { console.warn('[APEX] form NOT FOUND:', cfg.id); return; }
      const nameInput = document.getElementById(cfg.nameField);
      const phoneInput = document.getElementById(cfg.phoneField);
      const successEl = document.getElementById(cfg.successId);
      console.log('[APEX] setupForm:', cfg.id, 'successEl=', !!successEl, 'nameInput=', !!nameInput, 'phoneInput=', !!phoneInput);
  
      // Format phone
      if (phoneInput) {
        phoneInput.addEventListener('input', () => {
          // Cho phép 1 dấu '+' ở đầu + các chữ số (0-9)
          let v = phoneInput.value;
          // Bỏ mọi ký tự không phải số và không phải '+'
          v = v.replace(/[^\d+]/g, '');
          // Chỉ giữ '+' nếu nó ở đầu tiên; các dấu '+' sau đó bị loại bỏ
          v = v.replace(/\+/g, (m, offset) => (offset === 0 ? '+' : ''));
          // Giới hạn tối đa 12 ký tự (1 dấu '+' + 11 số)
          if (v.length > 12) v = v.slice(0, 12);
          phoneInput.value = v;
          if (v.length > 0) phoneInput.classList.remove('error');
          if (v.length > 0) phoneInput.classList.add('has-value');
          else phoneInput.classList.remove('has-value');
          trackEvent('phone_input', `${cfg.type}|len=${v.length}`);
        });
      }
      if (nameInput) {
        nameInput.addEventListener('input', () => {
          if (nameInput.value.length > 0) {
            nameInput.classList.remove('error');
            nameInput.classList.add('has-value');
          } else {
            nameInput.classList.remove('has-value');
          }
          trackEvent('name_input', `${cfg.type}|len=${nameInput.value.length}`);
        });
      }
  
      // Focus/blur tracking
      if (nameInput) {
        nameInput.addEventListener('focus', () => trackEvent('input_focus', `${cfg.type}|name`));
        nameInput.addEventListener('blur',  () => trackEvent('input_blur',  `${cfg.type}|name`));
      }
      if (phoneInput) {
        phoneInput.addEventListener('focus', () => trackEvent('input_focus', `${cfg.type}|phone`));
        phoneInput.addEventListener('blur',  () => trackEvent('input_blur',  `${cfg.type}|phone`));
      }
  
form.addEventListener('submit', async e => {
        e.preventDefault();
        console.log('[APEX] form submit:', cfg.id, cfg.type);
        if (form.dataset.submitting === '1') return;
        const rawName = nameInput ? nameInput.value.trim() : '';
        // Tên có thể để trống → nếu trống thì truyền "lead" lên backend
        const name = rawName.length > 0 ? rawName : 'lead';
        const phone = phoneInput ? phoneInput.value.trim() : '';

        // Client-side validation
        let ok = true;
        // Tên: optional (không bắt buộc), nếu có thì phải >= 2 ký tự
        if (rawName.length > 0 && rawName.length < 2) {
          if (nameInput) { shake(nameInput); trackEvent('validation_error', `${cfg.type}|name`); }
          ok = false;
        }
        // Phone: bắt buộc, cho phép 1 dấu '+' ở đầu.
        //   - Số chữ số phải > 9 (tức >= 10) và <= 11
        //   - Nếu có '+' ở đầu thì tổng ký tự tối đa = 12
        //   - Nếu không có '+' thì tổng ký tự tối đa = 11 (toàn số)
        const hasPlus = phone.startsWith('+');
        const phoneDigits = phone.replace(/\D/g, '');
        const phoneValid =
          phoneDigits.length >= 10 && phoneDigits.length <= 11 &&
          phone.length <= (hasPlus ? 12 : 11);
        if (!phoneValid) {
          if (phoneInput) { shake(phoneInput); trackEvent('validation_error', `${cfg.type}|phone`); }
          ok = false;
        }
        if (!ok) return;
  
        // Honeypot
        const honeypot = form.querySelector('input[name="website"]');
        if (honeypot && honeypot.value) return;
  
        form.dataset.submitting = '1';
        const submitBtn = form.querySelector('button[type="submit"]');
        if (submitBtn) {
          submitBtn.classList.add('btn-loading');
          submitBtn.dataset.originalHtml = submitBtn.innerHTML;
          submitBtn.innerHTML = '<span>ĐANG GỬI...</span>';
        }
  
        // === SINH IDEMPOTENCY KEY ===
        // Mỗi submit = 1 UUID duy nhất. Retry = dùng lại UUID → server dedupe.
        const idempotencyKey = generateUUID();
        const sessionId = getSessionId();
        const submitStartedAt = Date.now();
  
        // Build payload
        const payload = {
          name,
          phone,
          form_type: cfg.type,
          page_url: location.href,
          referrer: document.referrer || '',
          user_agent: navigator.userAgent,
          screen_res: `${screen.width}x${screen.height}`,
          viewport: `${window.innerWidth}x${window.innerHeight}`,
          device_type: /Mobi|Andr|iP(hone|ad|od)/i.test(navigator.userAgent) ? 'mobile' : 'desktop',
          session_id: sessionId,
          utm_source: qParam('utm_source'),
          utm_medium: qParam('utm_medium'),
          utm_campaign: qParam('utm_campaign'),
          utm_content: qParam('utm_content'),
          utm_term: qParam('utm_term'),
          fbclid: qParam('fbclid'),
          gclid: qParam('gclid')
        };
  
        // Save to retry queue NGAY LẬP TỨC (trước khi gửi) để chắc chắn không mất
        enqueueLead({
          idempotency_key: idempotencyKey,
          payload,
          retry_count: 0,
          next_attempt_at: Date.now(),
          submit_started_at: submitStartedAt,
          form_type: cfg.type
        });
  
        trackEvent('form_submit_attempt', `${cfg.type}|idk=${idempotencyKey.slice(0,8)}`);
  
        // === GỬI NGAY (4-tier strategy) ===
        const sendResult = await sendLead(idempotencyKey, payload);
  
        if (sendResult.ok) {
          // SUCCESS → remove khỏi queue
          dequeueLead(idempotencyKey);
          // Lưu thông tin lead vào localStorage
          try {
            localStorage.setItem('apex_lead_id', 'ch_' + sendResult.id);
            localStorage.setItem('apex_lead_name', name);
            localStorage.setItem('apex_lead_phone', phone);
          } catch(_){}
          trackEvent('form_submit_success', `${cfg.type}|id=${sendResult.id}|dur=${Date.now() - submitStartedAt}ms`);
  
          // UI success
          showSuccess(cfg, form, successEl);
  
          // Fire Meta Pixel Lead - CHỈ khi server xác nhận success
          try {
            if (typeof fbq !== 'undefined') {
              // Generate unique event_id for deduplication (Facebook không đếm trùng)
              const eventId = 'lead_' + idempotencyKey + '_' + Date.now();
              fbq('track', 'Lead', {
                content_name: 'APEX Registration',
                content_category: 'commodity_trading',
                form_type: cfg.type,
                utm_source: qParam('utm_source') || 'facebook',
                event_source_url: window.location.href
              }, {
                eventID: eventId
              });
              console.log('[Pixel] Lead tracked:', { eventId, formType: cfg.type, utm: qParam('utm_source') });
            }
          } catch(e) {
            console.warn('[Pixel] Lead track failed:', e);
          }
        } else {
          // FAIL → vẫn show success UI (vì đã lưu queue, sẽ retry)
          // Nhưng để user biết, sẽ hiển thị warning nhẹ
          trackEvent('form_submit_error', `${cfg.type}|status=${sendResult.status || 'network'}|err=${sendResult.error || 'unknown'}`);
          showSuccessWithWarning(cfg, form, successEl, sendResult);
  
          // Lead vẫn còn trong queue → sẽ retry ở background
          trackEvent('lead_queued_for_retry', `${cfg.type}|status=${sendResult.status || 'network'}`);
        }
  
        form.dataset.submitting = '0';
      });
    }
  
    /* ===========================================================
       CORE: SEND LEAD với 4-tier strategy
       =========================================================== */
    async function sendLead(idempotencyKey, payload){
      const start = Date.now();
      const headers = { 'Content-Type': 'application/json' };
  
      // === Tier 1: fetch keepalive (chính) ===
      try {
        const ctrl = new AbortController();
        const timeoutId = setTimeout(() => ctrl.abort(), LEAD_REQUEST_TIMEOUT_MS);
  
        const res = await fetch(API.LEAD, {
          method: 'POST',
          headers,
          body: JSON.stringify({ ...payload, idempotency_key: idempotencyKey }),
          keepalive: true,
          signal: ctrl.signal
        });
        clearTimeout(timeoutId);
  
        let body = null;
        try { body = await res.json(); } catch(_) {}
  
        if (res.ok && body && body.ok) {
          // Gọi ACK ngay (không block UI)
          sendAck(idempotencyKey).catch(() => {});
          return {
            ok: true,
            id: body.id,
            deduplicated: body.deduplicated || false,
            status: res.status,
            duration: Date.now() - start
          };
        }
  
        // Server trả lỗi có chủ đích (400, 429, 500) → KHÔNG retry, trả về để client xử lý
        console.error('[LEAD] Server error response:', {
          status: res.status,
          body: body,
          url: API.LEAD
        });
        return {
          ok: false,
          status: res.status,
          error: body?.error || 'server_error',
          detail: body?.detail || '',
          request_id: body?.request_id || '',
          duration: Date.now() - start
        };
      } catch (err) {
        // Network fail / timeout / abort
        console.warn('[LEAD] Send failed:', err);
        return {
          ok: false,
          status: 0,
          error: err.name === 'AbortError' ? 'timeout' : 'network_error',
          detail: err.message || String(err),
          duration: Date.now() - start
        };
      }
    }
  
    /* ===========================================================
       GỬI ACK (xác nhận client đã nhận response)
       =========================================================== */
    async function sendAck(idempotencyKey){
      try {
        const ctrl = new AbortController();
        const timeoutId = setTimeout(() => ctrl.abort(), ACK_REQUEST_TIMEOUT_MS);
        await fetch(API.ACK, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ idempotency_key: idempotencyKey }),
          keepalive: true,
          signal: ctrl.signal
        });
        clearTimeout(timeoutId);
      } catch(_) { /* ACK fail không quan trọng */ }
    }
  
    /* ===========================================================
       RETRY QUEUE (persistent localStorage)
       - queue/flushTimer/isFlushing/QUEUE_KEY/MAX_QUEUE_SIZE đã khai báo ở đầu file
       =========================================================== */
  
    function loadQueue(){
      try {
        const raw = localStorage.getItem(QUEUE_KEY);
        if (!raw) return [];
        const arr = JSON.parse(raw);
        return Array.isArray(arr) ? arr : [];
      } catch(_) { return []; }
    }
  
    function saveQueue(){
      try {
        // Giới hạn size (drop cũ nhất nếu quá)
        if (queue.length > MAX_QUEUE_SIZE) {
          queue = queue.slice(-MAX_QUEUE_SIZE);
        }
        localStorage.setItem(QUEUE_KEY, JSON.stringify(queue));
      } catch(_) {}
    }
  
    function enqueueLead(item){
      // Check duplicate theo idempotency_key
      if (queue.find(q => q.idempotency_key === item.idempotency_key)) return;
      queue.push(item);
      saveQueue();
      // Schedule flush ngay
      scheduleFlush(0);
    }
  
    function dequeueLead(idempotencyKey){
      queue = queue.filter(q => q.idempotency_key !== idempotencyKey);
      saveQueue();
    }
  
    function updateQueueItem(idempotencyKey, updates){
      const idx = queue.findIndex(q => q.idempotency_key === idempotencyKey);
      if (idx >= 0) {
        queue[idx] = { ...queue[idx], ...updates };
        saveQueue();
      }
    }
  
    function scheduleFlush(delayMs){
      if (flushTimer) clearTimeout(flushTimer);
      flushTimer = setTimeout(flushQueue, delayMs);
    }
  
    async function flushQueue(){
      if (isFlushing) return;
      if (!queue.length) return;
      if (!navigator.onLine) {
        // Offline → đợi online
        window.addEventListener('online', () => scheduleFlush(0), { once: true });
        return;
      }
  
      isFlushing = true;
  
      const now = Date.now();
      const toProcess = queue.filter(item => item.next_attempt_at <= now);
  
      if (toProcess.length === 0) {
        isFlushing = false;
        return;
      }
  
      trackEvent('retry_flush_start', `count=${toProcess.length}`);
  
      // === Batch gửi qua /api/lead-retry.php (1 request cho tất cả) ===
      const items = toProcess.map(item => ({
        idempotency_key: item.idempotency_key,
        payload: item.payload
      }));
  
      try {
        const ctrl = new AbortController();
        const timeoutId = setTimeout(() => ctrl.abort(), 10000);
        const res = await fetch(API.RETRY, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ items }),
          keepalive: true,
          signal: ctrl.signal
        });
        clearTimeout(timeoutId);
  
        if (res.ok) {
          const body = await res.json();
          if (body && body.ok && Array.isArray(body.results)) {
            body.results.forEach(r => {
              if (r.ok) {
                dequeueLead(r.idempotency_key);
                trackEvent('retry_success', `id=${r.id}|dedup=${r.deduplicated}`);
                // Lưu lead_id nếu là lead đầu tiên
                try {
                  if (!localStorage.getItem('apex_lead_id') && r.id) {
                    localStorage.setItem('apex_lead_id', 'ch_' + r.id);
                  }
                } catch(_){}
              } else {
                // Lỗi → tăng retry_count, schedule lại
                const item = queue.find(q => q.idempotency_key === r.idempotency_key);
                if (item) {
                  const nextCount = (item.retry_count || 0) + 1;
                  if (nextCount >= MAX_RETRIES) {
                    // Hết retry → xóa khỏi queue (đã log ở server)
                    dequeueLead(r.idempotency_key);
                    trackEvent('retry_giveup', `count=${nextCount}|err=${r.error}`);
                  } else {
                    const delay = RETRY_DELAYS_MS[Math.min(nextCount - 1, RETRY_DELAYS_MS.length - 1)];
                    updateQueueItem(r.idempotency_key, {
                      retry_count: nextCount,
                      next_attempt_at: Date.now() + delay,
                      last_error: r.error
                    });
                    trackEvent('retry_fail', `count=${nextCount}|err=${r.error}|next=${delay}ms`);
                  }
                }
              }
            });
          }
        } else {
          // Retry endpoint lỗi → tăng retry_count cho tất cả
          trackEvent('retry_endpoint_error', `status=${res.status}`);
          toProcess.forEach(item => {
            const nextCount = (item.retry_count || 0) + 1;
            if (nextCount >= MAX_RETRIES) {
              dequeueLead(item.idempotency_key);
            } else {
              const delay = RETRY_DELAYS_MS[Math.min(nextCount - 1, RETRY_DELAYS_MS.length - 1)];
              updateQueueItem(item.idempotency_key, {
                retry_count: nextCount,
                next_attempt_at: Date.now() + delay
              });
            }
          });
        }
      } catch (err) {
        console.warn('[RETRY] Flush failed:', err);
        trackEvent('retry_flush_network_error', err.message || 'unknown');
        toProcess.forEach(item => {
          const nextCount = (item.retry_count || 0) + 1;
          if (nextCount < MAX_RETRIES) {
            const delay = RETRY_DELAYS_MS[Math.min(nextCount - 1, RETRY_DELAYS_MS.length - 1)];
            updateQueueItem(item.idempotency_key, {
              retry_count: nextCount,
              next_attempt_at: Date.now() + delay
            });
          }
        });
      }
  
      isFlushing = false;
  
      // Còn item chưa retry → schedule tiếp
      if (queue.some(q => q.next_attempt_at <= Date.now())) {
        scheduleFlush(5000);
      }
    }
  
    function initRetryQueue(){
      queue = loadQueue();
  
      // Nếu có item pending ngay khi load → flush
      if (queue.length > 0) {
        trackEvent('retry_queue_loaded', `count=${queue.length}`);
        scheduleFlush(1000);
      }
  
      // Auto flush mỗi 30s
      setInterval(() => {
        if (queue.length > 0) flushQueue();
      }, QUEUE_FLUSH_INTERVAL_MS);
  
      // Online → flush
      window.addEventListener('online', () => {
        if (queue.length > 0) {
          trackEvent('online_recovery', `queue=${queue.length}`);
          scheduleFlush(0);
        }
      });
  
      // Page show (back/forward navigation) → flush
      window.addEventListener('pageshow', e => {
        if (e.persisted && queue.length > 0) {
          scheduleFlush(0);
        }
      });
  
      // Visibility change → flush
      document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible' && queue.length > 0) {
          scheduleFlush(500);
        }
      });
  
      // Page hide → flush ngay (sendBeacon)
      window.addEventListener('pagehide', () => {
        if (queue.length > 0) {
          flushQueueBeacon();
        }
      });
  
      // Trước unload → best effort
      window.addEventListener('beforeunload', () => {
        if (queue.length > 0) {
          flushQueueBeacon();
        }
      });
    }
  
    function flushQueueBeacon(){
      if (!queue.length || !navigator.sendBeacon) return;
      try {
        const items = queue.map(q => ({ idempotency_key: q.idempotency_key, payload: q.payload }));
        const blob = new Blob([JSON.stringify({ items })], { type: 'application/json' });
        navigator.sendBeacon(API.RETRY, blob);
        trackEvent('retry_flush_beacon', `count=${items.length}`);
      } catch(_) {}
    }
  
    /* ===========================================================
       4) MULTI-STEP FORM
       =========================================================== */
    function initMultiStepForm(){
      const form = document.getElementById('multiForm');
      if (!form) return;
  
      const channelInput = document.getElementById('multiChannel');
      const panes = form.querySelectorAll('[data-step-pane]');
      const indicators = document.querySelectorAll('[data-step-indicator]');
      const line = document.querySelector('[data-step-line]');
  
      const goToStep = (stepNum) => {
        panes.forEach(p => {
          if (parseInt(p.dataset.stepPane, 10) === stepNum) p.classList.add('is-active');
          else p.classList.remove('is-active');
        });
        indicators.forEach(ind => {
          const n = parseInt(ind.dataset.stepIndicator, 10);
          ind.classList.remove('is-active', 'is-done');
          if (n === stepNum) ind.classList.add('is-active');
          else if (n < stepNum) ind.classList.add('is-done');
        });
        if (line) {
          line.style.background = stepNum === 2
            ? 'linear-gradient(90deg, #FF6D00, #10B981)'
            : '#E5E7EB';
        }
        trackEvent('multistep_step_view', 'step_' + stepNum);
        setTimeout(() => {
          const firstInput = form.querySelector(`[data-step-pane="${stepNum}"] input:not([type=hidden])`);
          if (firstInput) firstInput.focus({ preventScroll: true });
        }, 200);
      };
  
      form.querySelectorAll('.channel-btn').forEach(btn => {
        btn.addEventListener('click', () => {
          const channel = btn.dataset.channel || '';
          const nextStep = parseInt(btn.dataset.nextStep, 10);
          if (channelInput) channelInput.value = channel;
          form.querySelectorAll('.channel-btn').forEach(b => b.classList.remove('is-selected'));
          btn.classList.add('is-selected');
          trackEvent('multistep_channel_selected', channel);
          setTimeout(() => goToStep(nextStep), 250);
        });
      });
  
      form.querySelectorAll('[data-prev-step]').forEach(btn => {
        btn.addEventListener('click', () => {
          const prev = parseInt(btn.dataset.prevStep, 10);
          goToStep(prev);
          trackEvent('multistep_back_clicked', 'from_step_2');
        });
      });
  
      const nameInput = document.getElementById('multiName');
      const phoneInput = document.getElementById('multiPhone');
      const successEl = document.getElementById('multiSuccess');
  
      if (phoneInput) {
        phoneInput.addEventListener('input', () => {
          // Cho phép 1 dấu '+' ở đầu + các chữ số (0-9)
          let v = phoneInput.value;
          v = v.replace(/[^\d+]/g, '');
          v = v.replace(/\+/g, (m, offset) => (offset === 0 ? '+' : ''));
          if (v.length > 12) v = v.slice(0, 12);
          phoneInput.value = v;
          if (v.length > 0) phoneInput.classList.remove('error');
          trackEvent('phone_input', `multistep|len=${v.length}`);
        });
      }
      if (nameInput) {
        nameInput.addEventListener('input', () => {
          if (nameInput.value.length > 0) nameInput.classList.remove('error');
          trackEvent('name_input', `multistep|len=${nameInput.value.length}`);
        });
      }
  
form.addEventListener('submit', async e => {
        e.preventDefault();
        if (form.dataset.submitting === '1') return;
        const rawName = nameInput ? nameInput.value.trim() : '';
        // Tên có thể để trống → nếu trống thì truyền "lead" lên backend
        const name = rawName.length > 0 ? rawName : 'lead';
        const phone = phoneInput ? phoneInput.value.trim() : '';
        const channel = channelInput ? channelInput.value : '';

        let ok = true;
        // Tên: optional (không bắt buộc), nếu có thì phải >= 2 ký tự
        if (rawName.length > 0 && rawName.length < 2) {
          if (nameInput) shake(nameInput);
          ok = false;
        }
        // Phone: bắt buộc, cho phép 1 dấu '+' ở đầu.
        //   - Số chữ số phải > 9 (tức >= 10) và <= 11
        //   - Nếu có '+' ở đầu thì tổng ký tự tối đa = 12
        //   - Nếu không có '+' thì tổng ký tự tối đa = 11 (toàn số)
        const hasPlusM = phone.startsWith('+');
        const phoneDigitsM = phone.replace(/\D/g, '');
        const phoneValidM =
          phoneDigitsM.length >= 10 && phoneDigitsM.length <= 11 &&
          phone.length <= (hasPlusM ? 12 : 11);
        if (!phoneValidM) {
          if (phoneInput) shake(phoneInput);
          ok = false;
        }
        if (!ok) return;
  
        const honeypot = form.querySelector('input[name="website"]');
        if (honeypot && honeypot.value) return;
  
        form.dataset.submitting = '1';
        const submitBtn = form.querySelector('button[type="submit"]');
        if (submitBtn) {
          submitBtn.classList.add('btn-loading');
          submitBtn.innerHTML = '<span>ĐANG GỬI...</span>';
        }
  
        const idempotencyKey = generateUUID();
        const sessionId = getSessionId();
        const submitStartedAt = Date.now();
  
        const payload = {
          name, phone,
          form_type: 'multistep',
          channel,
          page_url: location.href,
          referrer: document.referrer || '',
          user_agent: navigator.userAgent,
          screen_res: `${screen.width}x${screen.height}`,
          viewport: `${window.innerWidth}x${window.innerHeight}`,
          device_type: /Mobi|Andr|iP(hone|ad|od)/i.test(navigator.userAgent) ? 'mobile' : 'desktop',
          session_id: sessionId,
          utm_source: qParam('utm_source'),
          utm_medium: qParam('utm_medium'),
          utm_campaign: qParam('utm_campaign'),
          utm_content: qParam('utm_content'),
          utm_term: qParam('utm_term'),
          fbclid: qParam('fbclid'),
          gclid: qParam('gclid')
        };
  
        enqueueLead({
          idempotency_key: idempotencyKey,
          payload,
          retry_count: 0,
          next_attempt_at: Date.now(),
          submit_started_at: submitStartedAt,
          form_type: 'multistep'
        });
  
        trackEvent('multistep_submit_attempt', `${channel}|idk=${idempotencyKey.slice(0,8)}`);
  
        const sendResult = await sendLead(idempotencyKey, payload);
  
        if (sendResult.ok) {
          dequeueLead(idempotencyKey);
          try {
            localStorage.setItem('apex_lead_id', 'ch_' + sendResult.id);
            localStorage.setItem('apex_lead_name', name);
            localStorage.setItem('apex_lead_phone', phone);
          } catch(_){}
          trackEvent('multistep_submit_success', `${channel}|id=${sendResult.id}|dur=${Date.now() - submitStartedAt}ms`);
          showMultiSuccess(form, successEl);
          try {
            if (typeof fbq !== 'undefined') {
              const eventId = 'lead_' + idempotencyKey + '_' + Date.now();
              fbq('track', 'Lead', {
                content_name: 'APEX Registration',
                content_category: 'commodity_trading',
                form_type: 'multistep',
                utm_source: qParam('utm_source') || 'facebook',
                event_source_url: window.location.href
              }, {
                eventID: eventId
              });
              console.log('[Pixel] Lead tracked (multistep):', { eventId });
            }
          } catch(e) { console.warn('[Pixel] Lead track failed:', e); }
        } else {
          trackEvent('multistep_submit_error', `status=${sendResult.status || 'network'}|err=${sendResult.error}`);
          showMultiSuccessWithWarning(form, successEl, sendResult);
        }
  
        form.dataset.submitting = '0';
      });
  
      function showMultiSuccess(form, successEl){
        form.style.display = 'none';
        if (successEl) successEl.classList.remove('hidden');
        const indicator = document.querySelector('.step-indicator');
        if (indicator) indicator.style.display = 'none';
      }
      function showMultiSuccessWithWarning(form, successEl, sendResult){
        form.style.display = 'none';
        if (successEl) successEl.classList.remove('hidden');
        const indicator = document.querySelector('.step-indicator');
        if (indicator) indicator.style.display = 'none';
        const warn = document.createElement('div');
        warn.className = 'mt-3 text-xs text-amber-600';
        warn.textContent = 'Hệ thống đang lưu lại thông tin. Chúng tôi sẽ liên hệ với bạn trong ít phút.';
        successEl.appendChild(warn);
      }
  
      trackEvent('multistep_view', 'step_1');
    }
  
    /* ===========================================================
       5) Smooth scroll
       =========================================================== */
    function initSmoothScroll(){
      document.querySelectorAll('a[href^="#"]').forEach(a => {
        a.addEventListener('click', e => {
          const href = a.getAttribute('href');
          if (!href || href === '#' || href.length < 2) return;
          const target = document.querySelector(href);
          if (target) {
            e.preventDefault();
            const headerH = 64;
            const top = target.getBoundingClientRect().top + window.pageYOffset - headerH;
            window.scrollTo({ top, behavior: 'smooth' });
            trackEvent('anchor_click', href);
          }
        });
      });
    }
  
    /* ===========================================================
       6) Sticky CTA visibility
       =========================================================== */
    function initStickyCtaVisibility(){
      const sticky = document.getElementById('stickyCta');
      if (!sticky) return;
      document.addEventListener('focusin', e => {
        const tag = (e.target.tagName || '').toLowerCase();
        if (tag === 'input' || tag === 'textarea' || tag === 'select') {
          document.body.classList.add('form-focused');
          sticky.classList.add('is-hidden');
        }
      });
      document.addEventListener('focusout', () => {
        setTimeout(() => {
          const ae = document.activeElement;
          const tag = (ae && ae.tagName || '').toLowerCase();
          if (tag !== 'input' && tag !== 'textarea' && tag !== 'select') {
            document.body.classList.remove('form-focused');
            sticky.classList.remove('is-hidden');
          }
        }, 150);
      });
      if (window.visualViewport) {
        const check = () => {
          const shrunk = window.innerHeight - window.visualViewport.height;
          if (shrunk > 120) {
            document.body.classList.add('form-focused');
            sticky.classList.add('is-hidden');
          }
        };
        window.visualViewport.addEventListener('resize', check);
        window.visualViewport.addEventListener('scroll', check);
      }
    }
  
    /* ===========================================================
       7) Real-time counters
       =========================================================== */
    function initSeatsCounter(){
      const els = [document.getElementById('seatsLeftTop'), document.getElementById('seatsLeftBottom')].filter(Boolean);
      if (!els.length) return;
      let current = parseInt(els[0].textContent, 10) || 8;
      const minSeats = 1;
      setInterval(() => {
        if (Math.random() < 0.35 && current > minSeats) {
          current--;
          els.forEach(el => el.textContent = current);
          trackEvent('seats_decreased', current);
        }
      }, 30000);
    }
  
    function initGiftsCounter(){
      const givenEl = document.getElementById('giftGiven');
      const totalEl = document.getElementById('giftTotal');
      const remEl = document.getElementById('giftRemaining');
      const bar = document.getElementById('giftBar');
      if (!givenEl || !bar) return;
      const total = parseInt(totalEl.textContent, 10) || 50;
      let given = parseInt(givenEl.textContent, 10) || 42;
      setInterval(() => {
        if (given < total && Math.random() < 0.3) {
          given++;
          givenEl.textContent = given;
          if (remEl) remEl.textContent = total - given;
          const pct = Math.min(100, Math.round((given / total) * 100));
          bar.style.width = pct + '%';
          trackEvent('gift_increased', given);
        }
      }, 45000);
    }
  
    /* ===========================================================
       UI helpers
       =========================================================== */
    function showSuccess(cfg, form, successEl){
      if (!successEl) return;
      // Ẩn form header + body (không ẩn form để success vẫn hiện)
      const header = form.querySelector('.form-card-header, .modal-header-new');
      const body   = form.querySelector('.p-6, .modal-form-body');
      if (header) header.style.display = 'none';
      if (body)   body.style.display   = 'none';
      // Force show: xóa mọi display:none từ lần submit lỗi trước
      form.style.display = '';
      successEl.classList.remove('hidden');
      if (cfg.type === 'modal' && window.ApexModal) {
        setTimeout(() => window.ApexModal.close(), 2200);
      }
    }
  
    function showSuccessWithWarning(cfg, form, successEl, sendResult){
      if (!successEl) return;
      const header = form.querySelector('.form-card-header, .modal-header-new');
      const body   = form.querySelector('.p-6, .modal-form-body');
      if (header) header.style.display = 'none';
      if (body)   body.style.display   = 'none';
      form.style.display = '';
      successEl.classList.remove('hidden');
      // Thêm dòng cảnh báo nhỏ
      const warn = document.createElement('div');
      warn.className = 'mt-3 text-xs text-amber-700 bg-amber-50 border border-amber-200 rounded-lg px-3 py-2';
      let msg = 'Thông tin đã được ghi nhận. Hệ thống đang lưu vào hệ thống - chúng tôi sẽ liên hệ bạn trong ít phút.';
      if (sendResult.status === 429) msg = 'Bạn đã gửi quá nhiều yêu cầu. Chúng tôi sẽ liên hệ bạn trong ít phút.';
      else if (sendResult.error === 'timeout') msg = 'Mạng chậm - thông tin đang được lưu lại, chúng tôi sẽ liên hệ bạn.';
      else if (sendResult.status >= 500) msg = 'Hệ thống đang bận - thông tin đã được lưu, chúng tôi sẽ liên hệ bạn.';
      warn.textContent = msg;
      successEl.appendChild(warn);
      if (cfg.type === 'modal' && window.ApexModal) {
        setTimeout(() => window.ApexModal.close(), 3000);
      }
    }
  
    function shake(el){
      el.classList.add('error');
      setTimeout(() => el.classList.remove('error'), 400);
    }
  
    /* ===========================================================
       Generic helpers
       =========================================================== */
    function qParam(name){
      const m = location.search.match(new RegExp('[?&]' + name + '=([^&#]*)'));
      return m ? decodeURIComponent(m[1]) : '';
    }
  
    function trackEvent(type, value){
      try {
        if (window.ApexTracker && typeof window.ApexTracker.send === 'function') {
          window.ApexTracker.send(type, value);
        }
      } catch(_){}
    }
  
    function generateUUID(){
      if (window.crypto && crypto.randomUUID) return crypto.randomUUID();
      return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
        const r = (Math.random() * 16) | 0;
        return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
      });
    }
  
    function getSessionId(){
      try {
        const m = document.cookie.match(/(?:^|;\s*)apex_sid=([^;]+)/);
        if (m) return decodeURIComponent(m[1]);
      } catch(_){}
      return '';
    }
  
    // Kick off init AFTER all declarations
    ready(init);
  
  })();
  
  // v5 - ready called inside IIFE
  