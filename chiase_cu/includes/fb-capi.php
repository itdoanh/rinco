<?php
/**
 * Facebook Conversions API (CAPI) - Lead Event
 * 
 * Server-side event tracking để đảm bảo Lead events được gửi lên Facebook
 * ngay cả khi browser chặn pixel (adblock, tracking prevention, etc.)
 * 
 * Docs: https://developers.facebook.com/docs/marketing-api/conversions-api
 */

// Cấu hình CAPI
define('FB_PIXEL_ID', '1731191014777446');
define('FB_CAPI_TOKEN', 'EAANY9b4NgAwBSaOEx20f38bRPxnSVZBFXMkRG7lqxiIXctTeLMZB2OSBQqu4PkHRQdUUZAZBpxeYsNSKDYVTbMBIszYYDxPOQJnLNuUh6ZC9nGZCDgZAG4oePkRRsLPW2l2Fo2UVC2rmiRetltYoiDpozhJ5wBgKH7bLRcYHFRlIbJ5ZBcKk89ax6b6mHJ53ngZDZD');

/**
 * Gửi Lead event lên Facebook CAPI
 * 
 * @param array $eventData Dữ liệu event
 * @return array Kết quả: ['success' => bool, 'response' => mixed]
 */
function fb_send_lead_event(array $eventData): array {
    $pixelId = FB_PIXEL_ID;
    $token = FB_CAPI_TOKEN;
    
    // Validate token
    if ($token === 'YOUR_ACCESS_TOKEN_HERE' || empty($token)) {
        error_log('[FB CAPI] Token chưa được cấu hình');
        return ['success' => false, 'error' => 'token_not_configured'];
    }
    
    $endpoint = "https://graph.facebook.com/v18.0/{$pixelId}/events";
    
    // Build event payload theo Facebook spec
    $payload = [
        'data' => [
            [
                'event_name' => 'Lead',
                'event_time' => time(),
                'action_source' => 'website',
                'event_source_url' => $eventData['url'] ?? ($_SERVER['HTTP_ORIGIN'] ?? ''),
                'user_data' => [
                    // Client info (nếu có)
                    'ph' => isset($eventData['phone_hash']) ? [$eventData['phone_hash']] : [],
                    'fbc' => $eventData['fbc'] ?? null,
                    'fbp' => $eventData['fbp'] ?? null,
                ],
                'custom_data' => [
                    'content_name' => 'APEX Registration',
                    'content_category' => 'commodity_trading',
                    'form_type' => $eventData['form_type'] ?? 'unknown',
                ],
                // Deduplication
                'event_id' => $eventData['event_id'] ?? null,
            ]
        ],
        'access_token' => $token,
    ];
    
    // Remove null values
    $payload['data'][0]['user_data'] = array_filter($payload['data'][0]['user_data']);
    
    $ch = curl_init($endpoint);
    curl_setopt_array($ch, [
        CURLOPT_POST => true,
        CURLOPT_POSTFIELDS => json_encode($payload),
        CURLOPT_RETURNTRANSFER => true,
        CURLOPT_HTTPHEADER => ['Content-Type: application/json'],
        CURLOPT_TIMEOUT => 10,
        CURLOPT_CONNECTTIMEOUT => 5,
    ]);
    
    $response = curl_exec($ch);
    $httpCode = curl_getinfo($ch, CURLINFO_HTTP_CODE);
    $error = curl_error($ch);
    curl_close($ch);
    
    if ($error) {
        error_log('[FB CAPI] CURL Error: ' . $error);
        return ['success' => false, 'error' => $error];
    }
    
    $result = json_decode($response, true);
    
    if ($httpCode === 200 && isset($result['events_received'])) {
        error_log('[FB CAPI] Lead event sent successfully: ' . json_encode($result));
        return ['success' => true, 'response' => $result];
    }
    
    error_log('[FB CAPI] Lead event failed: HTTP ' . $httpCode . ' - ' . $response);
    return ['success' => false, 'http_code' => $httpCode, 'response' => $result];
}

/**
 * Tạo phone hash cho CAPI user matching
 * Facebook yêu cầu hashed phone theo format SHA256
 * 
 * @param string $phone Số điện thoại
 * @return string|null Hashed phone hoặc null nếu lỗi
 */
function fb_hash_phone(string $phone): ?string {
    // Normalize: remove spaces, leading +, country code 84 → 0
    $normalized = preg_replace('/\s+/', '', $phone);
    if (strpos($normalized, '+84') === 0) {
        $normalized = '0' . substr($normalized, 3);
    }
    
    // Hash với SHA256 + lowercase hex
    $hashed = hash('sha256', $normalized);
    return $hashed;
}

/**
 * Lấy Facebook Click ID (fbc) từ _fbc cookie
 */
function fb_get_fbc(): ?string {
    if (isset($_COOKIE['_fbc'])) {
        return $_COOKIE['_fbc'];
    }
    if (isset($_GET['fbclid'])) {
        return 'fb.1.' . time() . '.' . $_GET['fbclid'];
    }
    return null;
}

/**
 * Lấy Facebook Browser ID (fbp) từ _fbp cookie
 */
function fb_get_fbp(): ?string {
    return $_COOKIE['_fbp'] ?? null;
}

// === API Endpoint - CHỈ chạy khi được gọi TRỰC TIẾP (không phải require_once) ===
if (basename($_SERVER['SCRIPT_FILENAME'] ?? '') === 'fb-capi.php' && php_sapi_name() !== 'cli') {
    header('Content-Type: application/json');
    
    // Chỉ accept POST
    if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
        http_response_code(405);
        echo json_encode(['error' => 'method_not_allowed']);
        exit;
    }
    
    $input = json_decode(file_get_contents('php://input'), true);
    
    if (!$input || !isset($input['event_id'])) {
        http_response_code(400);
        echo json_encode(['error' => 'missing_event_id']);
        exit;
    }
    
    $result = fb_send_lead_event([
        'event_id' => $input['event_id'],
        'form_type' => $input['form_type'] ?? 'unknown',
        'url' => $input['url'] ?? '',
        'phone_hash' => isset($input['phone']) ? fb_hash_phone($input['phone']) : null,
        'fbc' => fb_get_fbc(),
        'fbp' => fb_get_fbp(),
    ]);
    
    http_response_code($result['success'] ? 200 : 500);
    echo json_encode($result);
}
