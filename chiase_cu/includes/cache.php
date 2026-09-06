<?php
/**
 * APEX FINTECH - APCu Caching Wrapper
 *
 * Cung cấp caching layer với APCu (in-memory) và file fallback.
 * Đảm bảo:
 * - Cache hit < 1ms cho dashboard KPIs
 * - Automatic TTL expiration
 * - Namespaced cache invalidation
 * - Safe fallback khi APCu không available
 *
 * PHP 8.4 optimized với named arguments và readonly properties
 */

declare(strict_types=1);

// ===== Chống double-declare khi /admin/includes/cache.php cũng load =====
if (class_exists('ApexCache', false)) {
    return;
}

class ApexCache
{
    /** @var string Prefix cho tất cả keys */
    private const PREFIX = 'apex:';

    /** @var int Default TTL: 60 seconds */
    private const DEFAULT_TTL = 60;

    /** @var bool APCu available flag */
    private static bool $apcuAvailable = false;

    /** @var string File cache directory */
    private static string $fileCacheDir = '';

    /** @var array TTL overrides theo namespace */
    private static array $namespaceTtls = [
        'dashboard_stats' => 30,
        'summary_daily' => 300,
        'lead_list' => 15,
        'leads_today_count' => 30,
        'utm_campaigns' => 3600,
        'device_stats' => 1800,
        'traffic_sources' => 300,
        'active_visitors' => 5,
    ];

    /**
     * Initialize cache - Gọi trong config.php hoặc entry point
     */
    public static function init(?string $fileCacheDir = null): void
    {
        self::$apcuAvailable = extension_loaded('apcu') && apcu_enabled();

        if ($fileCacheDir) {
            self::$fileCacheDir = rtrim($fileCacheDir, '/');
            if (!is_dir(self::$fileCacheDir)) {
                mkdir(self::$fileCacheDir, 0755, true);
            }
        } else {
            self::$fileCacheDir = sys_get_temp_dir() . '/apex_cache';
            if (!is_dir(self::$fileCacheDir)) {
                mkdir(self::$fileCacheDir, 0755, true);
            }
        }
    }

    /**
     * Get cached value hoặc load bằng callback
     *
     * @param string $namespace Cache namespace (e.g., 'dashboard_stats', 'lead_list')
     * @param callable $loader Callback để load data khi cache miss
     * @param int $ttl Time-to-live in seconds
     * @param mixed ...$args Arguments cho loader callback (để tạo unique key)
     * @return mixed
     *
     * Usage:
     *   $stats = ApexCache::get('dashboard_stats',
     *       fn() => loadDashboardStats($from, $to),
     *       ttl: 30,
     *       $from, $to
     *   );
     */
    public static function get(string $namespace, callable $loader, int $ttl = 60, mixed ...$args): mixed
    {
        $key = self::buildKey($namespace, $args);

        // Thử get từ APCu trước
        $value = self::apcuGet($key);
        if ($value !== null) {
            return $value;
        }

        // Thử get từ file cache
        $value = self::fileGet($namespace, $args);
        if ($value !== null) {
            // Repopulate APCu
            self::apcuSet($key, $value, $ttl);
            return $value;
        }

        // Cache miss - load from source
        $value = $loader(...$args);

        // Store in both caches
        self::apcuSet($key, $value, $ttl);
        self::fileSet($namespace, $args, $value, $ttl);

        return $value;
    }

    /**
     * Get cached value đồng bộ - không có loader
     * Trả về null nếu không có trong cache
     */
    public static function getSync(string $namespace, mixed ...$args): mixed
    {
        $key = self::buildKey($namespace, $args);

        $value = self::apcuGet($key);
        if ($value !== null) {
            return $value;
        }

        return self::fileGet($namespace, $args);
    }

    /**
     * Set cache value manually
     */
    public static function set(string $namespace, mixed $value, ?int $ttl = null, mixed ...$args): void
    {
        $ttl = $ttl ?? self::getTtl($namespace);
        $key = self::buildKey($namespace, $args);

        self::apcuSet($key, $value, $ttl);
        self::fileSet($namespace, $args, $value, $ttl);
    }

    /**
     * Invalidate specific cache entry
     */
    public static function invalidate(string $namespace, mixed ...$args): void
    {
        $key = self::buildKey($namespace, $args);

        // APCu delete
        if (self::$apcuAvailable) {
            apcu_delete($key);
        }

        // File delete
        self::fileDelete($namespace, $args);
    }

    /**
     * Invalidate all cache entries trong 1 namespace
     * Sử dụng APCuIterator (PHP 8.1+) hoặc scan file cache
     */
    public static function invalidateNamespace(string $namespace): void
    {
        $pattern = self::PREFIX . $namespace . ':';

        // APCu - dùng regex delete
        if (self::$apcuAvailable && PHP_VERSION_ID >= 80100) {
            try {
                apcu_delete(new APCUIterator('/^' . preg_quote($pattern, '/') . '/'));
            } catch (Throwable $e) {
                // Fallback: clear all APCu if iterator fails
                apcu_clear_cache();
            }
        }

        // File - xóa tất cả files trong namespace folder
        $dir = self::$fileCacheDir . '/' . $namespace;
        if (is_dir($dir)) {
            $files = glob($dir . '/*.json');
            if ($files) {
                foreach ($files as $file) {
                    @unlink($file);
                }
            }
        }
    }

    /**
     * Clear all cache (admin action)
     */
    public static function clear(): void
    {
        if (self::$apcuAvailable) {
            apcu_clear_cache();
        }

        // Clear file cache
        if (is_dir(self::$fileCacheDir)) {
            $dirs = glob(self::$fileCacheDir . '/*', GLOB_ONLYDIR);
            if ($dirs) {
                foreach ($dirs as $dir) {
                    $files = glob($dir . '/*.json');
                    if ($files) {
                        foreach ($files as $file) {
                            @unlink($file);
                        }
                    }
                    @rmdir($dir);
                }
            }
        }
    }

    /**
     * Get cache statistics
     */
    public static function stats(): array
    {
        $stats = [
            'apcu_available' => self::$apcuAvailable,
            'apcu_memory' => null,
            'file_cache_dir' => self::$fileCacheDir,
            'file_cache_size' => 0,
            'namespaces' => [],
        ];

        if (self::$apcuAvailable) {
            $info = apcu_cache_info(true);
            $stats['apcu_memory'] = [
                'used' => $info['mem_size'],
                'avail' => $info['memory_size'],
                'hits' => $info['num_hits'],
                'misses' => $info['num_misses'],
            ];
        }

        // File cache stats
        if (is_dir(self::$fileCacheDir)) {
            $totalSize = 0;
            $namespaces = [];
            $dirs = glob(self::$fileCacheDir . '/*', GLOB_ONLYDIR);
            if ($dirs) {
                foreach ($dirs as $dir) {
                    $ns = basename($dir);
                    $files = glob($dir . '/*.json') ?: [];
                    $count = count($files);
                    $size = 0;
                    foreach ($files as $file) {
                        $size += filesize($file);
                    }
                    $totalSize += $size;
                    $namespaces[$ns] = ['count' => $count, 'size' => $size];
                }
            }
            $stats['file_cache_size'] = $totalSize;
            $stats['namespaces'] = $namespaces;
        }

        return $stats;
    }

    /**
     * Check if APCu is available
     */
    public static function isApcuAvailable(): bool
    {
        return self::$apcuAvailable;
    }

    // ==================== PRIVATE HELPERS ====================

    private static function buildKey(string $namespace, array $args): string
    {
        if (empty($args)) {
            return self::PREFIX . $namespace;
        }
        $hash = md5(json_encode($args, JSON_INVALID_UTF8_IGNORE));
        return self::PREFIX . $namespace . ':' . $hash;
    }

    private static function getTtl(string $namespace): int
    {
        return self::$namespaceTtls[$namespace] ?? self::DEFAULT_TTL;
    }

    private static function apcuGet(string $key): mixed
    {
        if (!self::$apcuAvailable) {
            return null;
        }

        $value = apcu_fetch($key, $ok);
        return $ok ? $value : null;
    }

    private static function apcuSet(string $key, mixed $value, int $ttl): bool
    {
        if (!self::$apcuAvailable) {
            return false;
        }

        return apcu_store($key, $value, $ttl);
    }

    private static function fileGet(string $namespace, array $args): mixed
    {
        if (empty(self::$fileCacheDir)) {
            return null;
        }

        $file = self::getFilePath($namespace, $args);
        if (!file_exists($file)) {
            return null;
        }

        $stat = stat($file);
        if (!$stat) {
            return null;
        }

        $ttl = self::getTtl($namespace);
        if ($stat['mtime'] + $ttl < time()) {
            @unlink($file);
            return null;
        }

        $content = file_get_contents($file);
        if ($content === false) {
            return null;
        }

        return json_decode($content, true, 512, JSON_INVALID_UTF8_IGNORE);
    }

    private static function fileSet(string $namespace, array $args, mixed $value, int $ttl): bool
    {
        if (empty(self::$fileCacheDir)) {
            return false;
        }

        $dir = self::$fileCacheDir . '/' . $namespace;
        if (!is_dir($dir)) {
            mkdir($dir, 0755, true);
        }

        $file = self::getFilePath($namespace, $args);
        $content = json_encode($value, JSON_INVALID_UTF8_IGNORE | JSON_PARTIAL_OUTPUT_ON_ERROR);

        if ($content === false) {
            return false;
        }

        // Touch với mtime = now (TTL tính từ đây)
        $result = file_put_contents($file, $content, LOCK_EX);
        if ($result !== false) {
            touch($file);
        }

        return $result !== false;
    }

    private static function fileDelete(string $namespace, array $args): void
    {
        if (empty(self::$fileCacheDir)) {
            return;
        }

        $file = self::getFilePath($namespace, $args);
        if (file_exists($file)) {
            @unlink($file);
        }
    }

    private static function getFilePath(string $namespace, array $args): string
    {
        $hash = empty($args) ? 'default' : md5(json_encode($args, JSON_INVALID_UTF8_IGNORE));
        // Dùng 2 chars đầu của hash làm subdirectory (tối ưu I/O)
        $subdir = substr($hash, 0, 2);
        $dir = self::$fileCacheDir . '/' . $namespace . '/' . $subdir;

        if (!is_dir($dir)) {
            mkdir($dir, 0755, true);
        }

        return $dir . '/' . $hash . '.json';
    }
}

// Auto-init nếu được gọi
if (!isset($GLOBALS['__apex_cache_init'])) {
    ApexCache::init();
    $GLOBALS['__apex_cache_init'] = true;
}
