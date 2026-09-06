/* ============================================
   APEX FINTECH - TRACKER.JS v3 (Refactored)
   Features:
     - Stronger bot detection (client + server side whitelist for FB bots)
     - Detailed funnel tracking:
        * section_view (IntersectionObserver)
        * cta_click (with label + location)
        * modal_open / modal_close
        * input_focus / input_blur / input_change
        * form_validation_error
        * multistep_step_view / multistep_channel_selected
        * scroll_depth (25/50/75/100)
        * time_tick (heartbeat)
        * seats_decreased / gift_increased
        * page_view
     - Send via sendBeacon (preferred) hoặc fetch keepalive
     - Batched (10s interval) + flush on unload
   ============================================ */

(function(){
  'use strict';

  const BASE_PATH = window.APEX_BASE || '/chiase';
  const ENDPOINT = BASE_PATH + '/api/track.php';
  const FLUSH_INTERVAL_MS = 10000;
  const MAX_BUFFER = 30;

  let buffer = [];
  let sessionId = null;
  let flushTimer = null;
  let heartbeatTimer = null;
  let pageStart = Date.now();
  let lastTickSent = 0;
  let maxScroll = 0;
  let scrollReached = { 25:false, 50:false, 75:false, 100:false };
  let isUnloading = false;
  let isVisible = true;

  /* ============================================
     BOT DETECTION
     - Phân biệt FB/Google bots (cần đếm) vs malicious bots (loại bỏ)
     ============================================ */
  const ALLOWED_BOTS = [
    'facebookexternalhit', 'facebot',  // Facebook crawler - cho phép đếm link preview
    'twitterbot', 'linkedinbot',         // Social preview
    'whatsapp', 'telegrambot', 'slackbot',
    'googlebot', 'bingbot',              // Search engines
    'applebot'
  ];

  function detectBot(){
    if (navigator.webdriver) return { isBot: 1, reason: 'webdriver' };

    const ua = (navigator.userAgent || '').toLowerCase();
    const botPatterns = [
      'bot', 'crawler', 'spider', 'crawling', 'headless',
      'phantom', 'phantomjs', 'wget', 'curl', 'httpclient',
      'python-requests', 'scrapy', 'mediapartners',
      'yandex', 'baidu', 'duckduckgo', 'ahrefs', 'semrush',
      'mj12', 'dotbot', 'petalbot', 'petal', 'gtmetrix',
      'pingdom', 'uptimerobot', 'curl/', 'wget/',
      'go-http-client', 'okhttp', 'selenium', 'playwright', 'puppeteer'
    ];
    for (const p of botPatterns) {
      if (ua.indexOf(p) >= 0) return { isBot: 1, reason: 'ua_pattern:' + p };
    }

    // Whitelist các bot "hữu ích" → vẫn ghi event nhưng không tính visitor
    for (const w of ALLOWED_BOTS) {
      if (ua.indexOf(w) >= 0) return { isBot: 0, reason: 'whitelisted:' + w, isWhitelistedBot: 1 };
    }

    // Headless Chrome detection
    if (navigator.plugins && navigator.plugins.length === 0 &&
        /Chrome|Chromium/.test(navigator.userAgent) &&
        typeof window.chrome === 'undefined') {
      return { isBot: 1, reason: 'headless_chrome' };
    }

    if (!navigator.cookieEnabled) return { isBot: 1, reason: 'no_cookies' };
    if (!screen.width || !screen.height) return { isBot: 1, reason: 'no_screen' };

    return { isBot: 0, reason: '' };
  }

  const botInfo = detectBot();
  const IS_BOT = botInfo.isBot === 1;
  const IS_WHITELISTED_BOT = botInfo.isWhitelistedBot === 1;

  /* ============================================
     HELPERS
     ============================================ */
  function uuid(){
    if (window.crypto && crypto.randomUUID) return crypto.randomUUID();
    return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, c => {
      const r = (Math.random() * 16) | 0;
      return (c === 'x' ? r : (r & 0x3) | 0x8).toString(16);
    });
  }

  function hashCode(str){
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const chr = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + chr;
      hash |= 0; // Convert to 32bit int
    }
    // Trả về hex 8 chars (positive)
    return (hash >>> 0).toString(16).padStart(8, '0').slice(-8);
  }

  /* ============================================
     DEVICE FINGERPRINT
     - Tạo fingerprint DUY NHẤT cho mỗi thiết bị/trình duyệt
     - Session = cookie_id + fingerprint → tách được nhiều thiết bị
     ============================================ */
  function getFingerprint(){
    // Components tạo fingerprint duy nhất
    const components = [
      navigator.userAgent || '',
      navigator.language || '',
      screen.width + 'x' + screen.height + 'x' + (screen.colorDepth || 0),
      new Date().getTimezoneOffset(),
      navigator.hardwareConcurrency || '',
      // Canvas fingerprint
      (function(){
        try {
          const c = document.createElement('canvas');
          const ctx = c.getContext('2d');
          c.width = 200; c.height = 40;
          ctx.textBaseline = 'top';
          ctx.font = '14px Arial';
          ctx.fillStyle = '#f60';
          ctx.fillRect(125, 1, 62, 20);
          ctx.fillStyle = '#069';
          ctx.fillText('APEX Tracker', 2, 15);
          return c.toDataURL().slice(-50);
        } catch(_) { return ''; }
      })(),
      // Audio fingerprint (removed - caused unwanted sound + blocked by browser)
      ''
    ];
    
    // Hash fingerprint
    let hash = 0;
    const str = components.join('|');
    for (let i = 0; i < str.length; i++) {
      const chr = str.charCodeAt(i);
      hash = ((hash << 5) - hash) + chr;
      hash |= 0;
    }
    return Math.abs(hash).toString(16).padStart(8, '0');
  }

  function getSession(){
    if (sessionId) return sessionId;
    
    const cookieName = 'apex_sid';
    
    // Lấy cookie cũ
    const m = document.cookie.match(new RegExp('(^|; )' + cookieName + '=([^;]*)'));
    const fp = getFingerprint();
    
    // Cookie tồn tại → tạo composite session (cookie + fingerprint)
    if (m) {
      const storedSession = decodeURIComponent(m[2]);
      
      // Nếu là UUID cũ (36 ký tự) → tạo session mới với fingerprint
      // Nếu đã có fingerprint rồi → giữ nguyên
      if (storedSession.length === 36 && storedSession.includes('-')) {
        // UUID cũ → tạo session mới với fingerprint
        sessionId = uuid() + '-' + hashCode(fp).slice(0, 8);
      } else {
        // Session đã có fingerprint → giữ nguyên
        sessionId = storedSession;
      }
    } else {
      // Chưa có cookie → tạo mới với fingerprint
      sessionId = uuid() + '-' + hashCode(fp).slice(0, 8);
    }
    
    // Lưu cookie
    const exp = new Date(Date.now() + 365 * 24 * 3600 * 1000).toUTCString();
    document.cookie = cookieName + '=' + sessionId + '; expires=' + exp + '; path=/; SameSite=Lax';
    
    return sessionId;
  }

  function getDevice(){
    const ua = navigator.userAgent;
    if (/iPad|Tablet/i.test(ua)) return 'tablet';
    if (/Mobi|Andr|iP(hone|ad|od)/i.test(ua)) return 'mobile';
    return 'desktop';
  }

  function getOS(){
    const ua = navigator.userAgent;
    if (/Windows/i.test(ua)) return 'Windows';
    if (/Android/i.test(ua)) return 'Android';
    if (/iP(hone|ad|od)/i.test(ua)) return 'iOS';
    if (/Mac/i.test(ua)) return 'macOS';
    if (/Linux/i.test(ua)) return 'Linux';
    return 'Unknown';
  }

  function getBrowser(){
    const ua = navigator.userAgent;
    if (/Edg\//i.test(ua)) return 'Edge';
    if (/OPR/i.test(ua)) return 'Opera';
    if (/Chrome\//i.test(ua) && !/Chromium/i.test(ua)) return 'Chrome';
    if (/Firefox\//i.test(ua)) return 'Firefox';
    if (/Safari\//i.test(ua) && !/Chrome/i.test(ua)) return 'Safari';
    return 'Unknown';
  }

  function qParam(name){
    const m = location.search.match(new RegExp('[?&]' + name + '=([^&#]*)'));
    return m ? decodeURIComponent(m[1]) : '';
  }

  function commonContext(){
    return {
      session: getSession(),
      sid: getSession(),
      fp: getFingerprint(),  // Device fingerprint
      url: location.pathname + location.search,
      ref: document.referrer || '',
      utm_source: qParam('utm_source'),
      utm_medium: qParam('utm_medium'),
      utm_campaign: qParam('utm_campaign'),
      utm_content: qParam('utm_content'),
      utm_term: qParam('utm_term'),
      fbclid: qParam('fbclid'),
      gclid: qParam('gclid'),
      device_type: getDevice(),
      os_name: getOS(),
      browser_name: getBrowser(),
      screen_res: screen.width + 'x' + screen.height,
      language: (navigator.language || 'vi').slice(0, 10),
      tz: (Intl.DateTimeFormat().resolvedOptions().timeZone || '').slice(0, 50),
      is_bot: IS_BOT ? 1 : 0,
      bot_reason: botInfo.reason || '',
      is_whitelisted_bot: IS_WHITELISTED_BOT ? 1 : 0
    };
  }

  /* ============================================
     PUSH EVENT
     ============================================ */
  function push(type, value){
    if (!type) return;
    const ev = { t: Date.now(), type: String(type).slice(0, 40) };
    if (value !== undefined && value !== null) {
      ev.v = String(value).slice(0, 200);
    }
    buffer.push(ev);
    if (buffer.length >= MAX_BUFFER) flush();
  }

  /* ============================================
     FLUSH (send to server)
     ============================================ */
  function flush(){
    if (buffer.length === 0) return;
    const batch = buffer.slice();
    buffer = [];
    send(batch);
  }

  function flushFetch(){
    if (buffer.length === 0) return;
    const batch = buffer.slice();
    buffer = [];
    fetch(ENDPOINT, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ c: commonContext(), e: batch.map(x => [x.t, x.type, x.v || '']) }),
      keepalive: true,
      credentials: 'omit'
    }).catch(() => {});
  }

  function send(batch){
    if (!batch.length) return;
    const body = JSON.stringify({
      c: commonContext(),
      e: batch.map(x => [x.t, x.type, x.v || ''])
    });
    const blob = new Blob([body], { type: 'application/json' });

    // Prefer sendBeacon khi KHÔNG trong quá trình unload (gửi batch lớn)
    if (!isUnloading && navigator.sendBeacon) {
      try {
        const ok = navigator.sendBeacon(ENDPOINT, blob);
        if (ok) return;
      } catch(_){}
    }
    // Fallback: fetch keepalive (chạy được cả khi đã navigate)
    try {
      fetch(ENDPOINT, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: body,
        keepalive: true,
        credentials: 'omit'
      }).catch(() => {});
    } catch(_){}
  }

  /* ============================================
     INIT
     ============================================ */
  function init(){
    if (window.ApexTracker) return;
    window.ApexTracker = {
      send: push,
      flush: flush,
      isBot: IS_BOT,
      isWhitelistedBot: IS_WHITELISTED_BOT,
      sessionId: getSession()
    };

    // --- Page view ---
    push('page_view');

    // --- Heartbeat (mỗi 10s) ---
    heartbeatTimer = setInterval(() => {
      const seconds = Math.round((Date.now() - pageStart) / 1000);
      if (seconds - lastTickSent >= 9) {
        push('time_tick', seconds);
        lastTickSent = seconds;
      }
    }, FLUSH_INTERVAL_MS);

    // --- Auto-flush ---
    flushTimer = setInterval(flush, FLUSH_INTERVAL_MS);

    // --- Visibility / unload ---
    document.addEventListener('visibilitychange', () => {
      isVisible = document.visibilityState === 'visible';
      if (!isVisible) flush();
    });
    window.addEventListener('pagehide', () => { isUnloading = true; flush(); });
    window.addEventListener('beforeunload', flush);

    // --- Scroll depth ---
    setupScrollTracking();

    // --- Section visibility (IntersectionObserver) ---
    setupSectionViewTracking();

    // --- CTA / button clicks (delegation) ---
    setupClickTracking();

    // --- Form interactions ---
    setupFormTracking();

    console.log('[ApexTracker] Init done. is_bot=' + IS_BOT + ', bot_reason=' + botInfo.reason);
  }

  /* ============================================
     SCROLL DEPTH
     ============================================ */
  function setupScrollTracking(){
    let scrollTimer = null;
    window.addEventListener('scroll', () => {
      const docH = document.documentElement.scrollHeight - window.innerHeight;
      const pct = docH > 0 ? Math.round((window.scrollY / docH) * 100) : 0;
      if (pct > maxScroll) maxScroll = pct;
      clearTimeout(scrollTimer);
      scrollTimer = setTimeout(() => {
        [25, 50, 75, 100].forEach(threshold => {
          if (!scrollReached[threshold] && maxScroll >= threshold) {
            scrollReached[threshold] = true;
            push('scroll_depth', threshold);
          }
        });
      }, 200);
    }, { passive: true });
  }

  /* ============================================
     SECTION VIEW (IntersectionObserver)
     ============================================ */
  function setupSectionViewTracking(){
    if (!('IntersectionObserver' in window)) return;
    const sections = document.querySelectorAll('section[id], main > div[id], [data-track-section]');
    if (!sections.length) return;
    const viewedSet = new Set();

    const obs = new IntersectionObserver(entries => {
      entries.forEach(entry => {
        if (entry.isIntersecting && !viewedSet.has(entry.target.id || entry.target.dataset.trackSection)) {
          const id = entry.target.id || entry.target.dataset.trackSection || 'unknown';
          viewedSet.add(id);
          push('section_view', id);
        }
      });
    }, { threshold: 0.4 });

    sections.forEach(s => obs.observe(s));
  }

  /* ============================================
     CLICK TRACKING (delegation)
     Track:
       - data-modal-trigger (modal open)
       - data-cta-label (custom CTA)
       - .btn-cta (CTA buttons)
       - anchor # link
     ============================================ */
  function setupClickTracking(){
    document.addEventListener('click', e => {
      const target = e.target.closest('[data-modal-trigger], [data-cta-label], .btn-cta, a[href^="#"]');
      if (!target) return;

      if (target.dataset.modalTrigger !== undefined) {
        push('cta_click', 'modal:' + (target.dataset.ctaLabel || target.dataset.modalTrigger));
      } else if (target.dataset.ctaLabel) {
        push('cta_click', target.dataset.ctaLabel);
      } else if (target.classList && target.classList.contains('btn-cta')) {
        const txt = (target.textContent || '').trim().slice(0, 40);
        push('cta_click', 'btn:' + txt);
      } else if (target.tagName === 'A') {
        push('cta_click', 'anchor:' + (target.getAttribute('href') || '').slice(0, 30));
      }
    }, { capture: true });
  }

  /* ============================================
     FORM INTERACTION TRACKING
     Track: focus / blur / input / submit / validation_error
     ============================================ */
  function setupFormTracking(){
    // Focus tracking (per form + field)
    document.addEventListener('focusin', e => {
      const form = e.target.closest('form');
      if (!form) return;
      const formId = form.id || 'unknown';
      const field = e.target.name || e.target.id || 'unknown';
      push('input_focus', formId + '|' + field);
    });

    // Blur tracking
    document.addEventListener('focusout', e => {
      const form = e.target.closest('form');
      if (!form) return;
      const formId = form.id || 'unknown';
      const field = e.target.name || e.target.id || 'unknown';
      // Track blur kèm giá trị (đã nhập gì) - để biết funnel drop-off
      const v = e.target.value ? `len=${e.target.value.length}` : 'empty';
      push('input_blur', formId + '|' + field + '|' + v);
    });

    // Input tracking (typing)
    document.addEventListener('input', e => {
      const form = e.target.closest('form');
      if (!form) return;
      const formId = form.id || 'unknown';
      const field = e.target.name || e.target.id || 'unknown';
      // Chỉ track số lần keystroke (mỗi 3 lần mới push để giảm tải)
      if (!e.target.__apexInputCount) e.target.__apexInputCount = 0;
      e.target.__apexInputCount = (e.target.__apexInputCount || 0) + 1;
      if (e.target.__apexInputCount % 3 === 0) {
        push('input_change', formId + '|' + field + '|len=' + (e.target.value || '').length);
      }
    });

    // Submit tracking (kèm trạng thái)
    document.addEventListener('submit', e => {
      const formId = e.target.id || 'unknown';
      push('form_submit', formId);
    });

    // Modal open/close
    document.addEventListener('click', e => {
      const closer = e.target.closest('[data-close-modal]');
      if (closer) {
        const modal = closer.closest('.modal-overlay');
        if (modal) {
          push('modal_close', modal.id || 'modal');
        }
      }
    });
  }

  /* ============================================
     BOOT
     ============================================ */
  if (document.readyState === 'loading') {
    document.addEventListener('DOMContentLoaded', init);
  } else {
    init();
  }

})();
