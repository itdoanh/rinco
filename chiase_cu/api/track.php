    <?php
    /**
     * APEX FINTECH - API: TRACKING v2
     * POST JSON -> INSERT batch vào `events` + UPSERT `sessions` + `visitor_sessions`
     *
     * Logic:
     *  - events: ghi tất cả (kể cả bot) - dùng để phân tích chi tiết
     *  - sessions: ghi tất cả (tương thích cũ)
     *  - visitor_sessions: CHỈ ghi real visitors (is_bot=0) - dùng cho dashboard
     */

    require_once __DIR__ . '/../includes/config.php';
require_once __DIR__ . '/../admin/includes/db.php';

    header('Content-Type: application/json; charset=utf-8');
    header('Cache-Control: no-store, no-cache, must-revalidate');
    header('Access-Control-Allow-Origin: *');
    header('Access-Control-Allow-Methods: POST, OPTIONS');
    header('Access-Control-Allow-Headers: Content-Type');

    if ($_SERVER['REQUEST_METHOD'] === 'OPTIONS') { http_response_code(204); exit; }
    if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
        http_response_code(405);
        echo json_encode(['ok' => false, 'error' => 'method']);
        exit;
    }

    $raw = file_get_contents('php://input');
    if (!$raw || strlen($raw) > 50000) {
        http_response_code(413);
        echo json_encode(['ok' => false, 'error' => 'too_large']);
        exit;
    }
    $data = json_decode($raw, true);
    if (!is_array($data)) {
        http_response_code(400);
        echo json_encode(['ok' => false, 'error' => 'invalid']);
        exit;
    }

    $ctx = $data['c'] ?? [];
    $events = $data['e'] ?? [];

    $sessionId = $ctx['session'] ?? $ctx['sid'] ?? '';
    // Accept: UUID (36) hoặc UUID-hash (45 chars) cho device fingerprint
    if (!is_string($sessionId) || !preg_match('/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}(-[a-f0-9]{1,8})?$/i', $sessionId)) {
        http_response_code(400);
        echo json_encode(['ok' => false, 'error' => 'bad_session', 'id' => $sessionId]);
        exit;
    }

    // Bot detection từ client + double check server-side
    $isBot = (int)($ctx['is_bot'] ?? 0);
    $botReason = (string)($ctx['bot_reason'] ?? '');

    if (!$isBot) {
        // Server-side double check UA
        $ua = $_SERVER['HTTP_USER_AGENT'] ?? '';
        $botUaPatterns = ['bot', 'crawler', 'spider', 'headless', 'facebookexternalhit',
                        'googlebot', 'bingbot', 'ahrefs', 'semrush', 'mj12',
                        'wget/', 'curl/', 'python-requests', 'go-http-client',
                        'phantomjs', 'selenium', 'playwright'];
        $lowerUa = strtolower($ua);
        foreach ($botUaPatterns as $p) {
            if (strpos($lowerUa, $p) !== false) {
                $isBot = 1;
                $botReason = 'server_ua:' . $p;
                break;
            }
        }
    }

    $ip = client_ip();
    if (!rate_limit($ip, 'track', TRACK_RATE_PER_MIN, 60)) {
        http_response_code(429);
        echo json_encode(['ok' => false, 'error' => 'rl']);
        exit;
    }

    $ALLOWED_TYPES = [
        'page_view','scroll_depth','time_tick','cta_click',
        'form_focus','form_submit','form_idle_15s','lead','session_start',
        // v3: tracking chi tiết cho lead pipeline
        'input_focus','input_blur','input_change',
        'name_input','phone_input','validation_error',
        'form_submit_attempt','form_submit_success','form_submit_error','form_submit_network_error',
        'modal_open','modal_close',
        'multistep_view','multistep_step_view','multistep_channel_selected','multistep_back_clicked',
        'multistep_submit_attempt','multistep_submit_success','multistep_submit_error','multistep_submit_network_error',
        'lead_queued_for_retry','retry_flush_start','retry_success','retry_fail','retry_giveup',
        'retry_endpoint_error','retry_flush_network_error','retry_queue_loaded','retry_flush_beacon',
        'online_recovery','section_view','anchor_click',
        'pending_lead_retried_success',
        'seats_decreased','gift_increased'
    ];

    if (!is_array($events) || empty($events)) {
        echo json_encode(['ok' => true, 'count' => 0]);
        exit;
    }

    $events = array_slice($events, 0, 50);

    try {
        $pdo = apex_db();
    } catch (Throwable $e) {
        // Log error nhưng vẫn return ok để không làm phiền client
        error_log('[track.php] DB error: ' . $e->getMessage());
        echo json_encode(['ok' => true, 'count' => 0, 'debug_error' => 'db_connect_failed']);
        exit;
    }

    // === INSERT EVENTS (ghi tất cả) ===
    $insert = $pdo->prepare("
        INSERT INTO events
        (session_id, event_type, event_value, form_type, page_url, referrer,
        utm_source, utm_medium, utm_campaign, utm_content, utm_term,
        fbclid, gclid, device_type, os_name, browser_name, screen_res,
        language, timezone, ip, user_agent, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, FROM_UNIXTIME(?/1000))
    ");

    $ua = mb_substr($_SERVER['HTTP_USER_AGENT'] ?? '', 0, 500);

    // Các event_type có form_type trong event_value (prefix: "hero|..." hoặc "modal|...")
    $FORM_EVENTS = [
        'form_focus','form_submit','form_submit_attempt','form_submit_success','form_submit_error','form_submit_network_error',
        'form_idle_15s','input_focus','input_blur','input_change','name_input','phone_input','validation_error',
        'multistep_view','multistep_step_view','multistep_channel_selected','multistep_back_clicked',
        'multistep_submit_attempt','multistep_submit_success','multistep_submit_error','multistep_submit_network_error'
    ];
    $ALLOWED_FORM_TYPES = ['hero','modal','multistep','exit','unknown'];

    $pdo->beginTransaction();
    $count = 0;
    foreach ($events as $e) {
        if (!is_array($e) || count($e) < 2) continue;
        [$t, $type, $v] = array_pad($e, 3, '');
        $t = is_numeric($t) ? (int)$t : time() * 1000;
        $type = is_string($type) && in_array($type, $ALLOWED_TYPES, true) ? $type : 'unknown';
        $v = is_scalar($v) ? mb_substr((string)$v, 0, 200) : '';

        // Tách form_type từ event_value (format: "hero|id=...|dur=..." hoặc "modal|...")
        $formType = null;
        if (in_array($type, $FORM_EVENTS, true) && strpos($v, '|') !== false) {
            $candidate = strtolower(trim(explode('|', $v, 2)[0]));
            if (in_array($candidate, $ALLOWED_FORM_TYPES, true)) {
                $formType = $candidate;
            }
        }

        try {
            $insert->execute([
                $sessionId, $type, $v, $formType,
                mb_substr($ctx['url'] ?? '', 0, 255),
                mb_substr($ctx['ref'] ?? '', 0, 255),
                $ctx['utm_source'] ?? null,
                $ctx['utm_medium'] ?? null,
                $ctx['utm_campaign'] ?? null,
                $ctx['utm_content'] ?? null,
                $ctx['utm_term'] ?? null,
                $ctx['fbclid'] ?? null,
                $ctx['gclid'] ?? null,
                $ctx['device_type'] ?? null,
                $ctx['os_name'] ?? null,
                $ctx['browser_name'] ?? null,
                $ctx['screen_res'] ?? null,
                $ctx['language'] ?? null,
                $ctx['tz'] ?? null,
                $ip,
                $ua,
                $t
            ]);
            $count++;
        } catch (Throwable $_) {}
    }
    $pdo->commit();

    // === UPDATE SESSIONS (tương thích cũ) ===
    $lastEventTime = end($events)[0] ?? (time() * 1000);
    $firstEventTime = $events[0][0] ?? $lastEventTime;

    try {
        $pageViews = 0;
        foreach ($events as $e) { if (($e[1] ?? '') === 'page_view') $pageViews++; }
        if ($pageViews === 0) $pageViews = 1;

        $pdo->prepare("
            INSERT INTO sessions
                (session_id, first_visit, last_activity, page_views, device_type, utm_source, browser_name, os_name)
            VALUES (?, FROM_UNIXTIME(?/1000), FROM_UNIXTIME(?/1000), 1, ?, ?, ?, ?)
            ON DUPLICATE KEY UPDATE
                last_activity = FROM_UNIXTIME(?/1000),
                page_views = page_views + 1
        ")->execute([
            $sessionId, $firstEventTime, $lastEventTime,
            $ctx['device_type'] ?? 'unknown',
            $ctx['utm_source'] ?? null,
            $ctx['browser_name'] ?? null,
            $ctx['os_name'] ?? null,
            $lastEventTime
        ]);

        foreach ($events as $e) {
            if (($e[1] ?? '') === 'lead') {
                $pdo->prepare("UPDATE sessions SET is_lead=1 WHERE session_id=?")->execute([$sessionId]);
                break;
            }
        }
        foreach ($events as $e) {
            if (($e[1] ?? '') === 'scroll_depth') {
                $v = (int)($e[2] ?? 0);
                if ($v > 0) {
                    $pdo->prepare("UPDATE sessions SET scroll_max=GREATEST(scroll_max, ?) WHERE session_id=?")
                        ->execute([$v, $sessionId]);
                }
            }
        }
    } catch (Throwable $_) {}

    // === UPDATE visitor_sessions (CHỈ ghi nếu KHÔNG phải bot) ===
    $vsError = null;
    if (!$isBot) {
        try {
            $pdo->prepare("
                INSERT INTO visitor_sessions
                    (session_id, ip, user_agent, is_bot, bot_reason, device_type, browser_name, os_name,
                    screen_res, language, referrer, utm_source, utm_medium, utm_campaign,
                    utm_content, fbclid, gclid, first_visit, last_activity, page_views)
                VALUES (?, ?, ?, 0, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?,
                        FROM_UNIXTIME(?/1000), FROM_UNIXTIME(?/1000), 1)
                ON DUPLICATE KEY UPDATE
                    last_activity = FROM_UNIXTIME(?/1000),
                    page_views = page_views + 1
            ")->execute([
                $sessionId, $ip, $ua,
                $ctx['device_type'] ?? null,
                $ctx['browser_name'] ?? null,
                $ctx['os_name'] ?? null,
                $ctx['screen_res'] ?? null,
                $ctx['language'] ?? null,
                mb_substr($ctx['ref'] ?? '', 0, 500),
                $ctx['utm_source'] ?? null,
                $ctx['utm_medium'] ?? null,
                $ctx['utm_campaign'] ?? null,
                $ctx['utm_content'] ?? null,
                $ctx['fbclid'] ?? null,
                $ctx['gclid'] ?? null,
                $firstEventTime, $lastEventTime,
                $lastEventTime
            ]);
        } catch (Throwable $ex) {
            $vsError = $ex->getMessage();
        }

        // Cập nhật scroll_max, is_lead
        try {
            foreach ($events as $e) {
                $type = $e[1] ?? '';
                if ($type === 'lead') {
                    $pdo->prepare("UPDATE visitor_sessions SET is_lead=1 WHERE session_id=?")->execute([$sessionId]);
                }
                if ($type === 'scroll_depth') {
                    $v = (int)($e[2] ?? 0);
                    if ($v > 0) {
                        $pdo->prepare("UPDATE visitor_sessions SET scroll_max=GREATEST(scroll_max, ?) WHERE session_id=?")
                            ->execute([$v, $sessionId]);
                    }
                }
                if ($type === 'time_tick') {
                    $sec = (int)($e[2] ?? 0);
                    if ($sec > 0) {
                        $pdo->prepare("UPDATE visitor_sessions SET total_duration=GREATEST(total_duration, ?) WHERE session_id=?")
                            ->execute([$sec, $sessionId]);
                    }
                }
            }
        } catch (Throwable $_) {}
    }

    $resp = ['ok' => true, 'count' => $count, 'is_bot' => $isBot];
    if ($vsError) $resp['vs_error'] = $vsError;
    echo json_encode($resp);

    // ===== HELPERS =====
    function client_ip(){
        foreach (['HTTP_CF_CONNECTING_IP','HTTP_X_FORWARDED_FOR','REMOTE_ADDR'] as $k) {
            if (!empty($_SERVER[$k])) {
                $ip = explode(',', $_SERVER[$k])[0];
                $ip = trim($ip);
                if (filter_var($ip, FILTER_VALIDATE_IP)) return $ip;
            }
        }
        return '0.0.0.0';
    }

    function rate_limit($ip, $bucket, $max, $windowSec){
        $dir = sys_get_temp_dir() . '/apex_rl/';
        if (!is_dir($dir)) @mkdir($dir, 0700, true);
        $f = $dir . md5($ip . '|' . $bucket) . '.txt';
        $data = ['count' => 0, 'reset' => 0];
        if (file_exists($f)) {
            $data = json_decode(@file_get_contents($f), true) ?: $data;
        }
        if ($data['reset'] < time()) {
            $data = ['count' => 0, 'reset' => time() + $windowSec];
        }
        $data['count']++;
        @file_put_contents($f, json_encode($data), LOCK_EX);
        return $data['count'] <= $max;
    }

    function db(){
        static $pdo = null;
        if ($pdo) return $pdo;
        $dsn = 'mysql:host=' . DB_HOST . ';dbname=' . DB_NAME . ';charset=' . DB_CHARSET;
        $pdo = new PDO($dsn, DB_USER, DB_PASS, [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_EMULATE_PREPARES => false,
            PDO::ATTR_PERSISTENT => false
        ]);
        return $pdo;
    }
