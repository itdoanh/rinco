<?php
/**
 * APEX FINTECH - PDO Database Helper v2
 *
 * Cung cấp wrapper around PDO với:
 * - Singleton connection
 * - Prepared statement caching (tái sử dụng statement objects)
 * - Convenience methods cho common queries
 * - Transaction support
 * - Error logging
 *
 * PHP 8.4 optimized
 */

declare(strict_types=1);

require_once __DIR__ . '/config.php';

class ApexDB
{
    /** @var PDO|null Singleton PDO instance */
    private static ?PDO $pdo = null;

    /** @var array<string, PDOStatement> Cached prepared statements */
    private static array $stmtCache = [];

    /** @var bool Transaction state */
    private static bool $inTransaction = false;

    /**
     * Get PDO connection (singleton)
     */
    public static function get(): PDO
    {
        if (self::$pdo === null) {
            self::connect();
        }
        return self::$pdo;
    }

    /**
     * Connect to database
     */
    private static function connect(): void
    {
        $dsn = sprintf(
            'mysql:host=%s;dbname=%s;charset=%s',
            DB_HOST,
            DB_NAME,
            DB_CHARSET
        );

        $options = [
            PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION,
            PDO::ATTR_EMULATE_PREPARES => false,
            PDO::ATTR_PERSISTENT => false,
            PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
            PDO::MYSQL_ATTR_INIT_COMMAND => "SET NAMES utf8mb4, time_zone = '+07:00'",
            PDO::MYSQL_ATTR_FOUND_ROWS => true,
            PDO::ATTR_STRINGIFY_FETCHES => false,
        ];

        // Try Unix socket first (faster on cPanel)
        $socketPath = '/var/mysql/mysql.sock';
        if (file_exists($socketPath)) {
            $dsn = sprintf(
                'mysql:unix_socket=%s;dbname=%s;charset=%s',
                $socketPath,
                DB_NAME,
                DB_CHARSET
            );
        }

        try {
            self::$pdo = new PDO($dsn, DB_USER, DB_PASS, $options);
        } catch (PDOException $e) {
            error_log('APEX DB Connection Failed: ' . $e->getMessage());
            throw $e;
        }
    }

    /**
     * Get cached prepared statement
     *
     * @param string $sql SQL query
     * @return PDOStatement
     *
     * Usage:
     *   $stmt = ApexDB::stmt("SELECT * FROM leads WHERE id = ?");
     *   $stmt->execute([123]);
     *   $lead = $stmt->fetch();
     */
    public static function stmt(string $sql): PDOStatement
    {
        $hash = md5($sql);

        if (!isset(self::$stmtCache[$hash])) {
            self::$stmtCache[$hash] = self::get()->prepare($sql);
        }

        return self::$stmtCache[$hash];
    }

    /**
     * Execute query và return all rows
     *
     * @param string $sql SQL query
     * @param array $params Parameters
     * @return array<int, array<string, mixed>>
     */
    public static function selectAll(string $sql, array $params = []): array
    {
        $stmt = self::stmt($sql);
        $stmt->execute($params);
        return $stmt->fetchAll();
    }

    /**
     * Execute query và return first row
     *
     * @param string $sql SQL query
     * @param array $params Parameters
     * @return array<string, mixed>|null
     */
    public static function selectOne(string $sql, array $params = []): ?array
    {
        $stmt = self::stmt($sql);
        $stmt->execute($params);
        $result = $stmt->fetch();
        return $result === false ? null : $result;
    }

    /**
     * Execute query và return single value (scalar)
     *
     * @param string $sql SQL query
     * @param array $params Parameters
     * @return mixed
     */
    public static function scalar(string $sql, array $params = []): mixed
    {
        $stmt = self::stmt($sql);
        $stmt->execute($params);
        $result = $stmt->fetchColumn();
        return $result === false ? null : $result;
    }

    /**
     * Execute query và return count
     *
     * @param string $sql SQL query (nên là COUNT query)
     * @param array $params Parameters
     * @return int
     */
    public static function count(string $sql, array $params = []): int
    {
        return (int)self::scalar($sql, $params);
    }

    /**
     * Insert row và return inserted ID
     *
     * @param string $table Table name
     * @param array<string, mixed> $data Key-value pairs
     * @return int|string Inserted ID
     */
    public static function insert(string $table, array $data): int|string
    {
        $columns = implode(', ', array_keys($data));
        $placeholders = ':' . implode(', :', array_keys($data));

        $sql = "INSERT INTO {$table} ({$columns}) VALUES ({$placeholders})";
        self::stmt($sql)->execute($data);

        return self::get()->lastInsertId();
    }

    /**
     * Insert row với ON DUPLICATE KEY UPDATE
     *
     * @param string $table Table name
     * @param array<string, mixed> $data Key-value pairs
     * @param array<string, mixed> $update Columns to update on duplicate (null = update all)
     * @return int Affected rows
     */
    public static function upsert(string $table, array $data, ?array $update = null): int
    {
        $columns = implode(', ', array_keys($data));
        $placeholders = ':' . implode(', :', array_keys($data));

        if ($update === null) {
            $update = $data;
        }

        $updateParts = [];
        foreach (array_keys($update) as $col) {
            $updateParts[] = "{$col} = VALUES({$col})";
        }
        $updateSql = implode(', ', $updateParts);

        $sql = "INSERT INTO {$table} ({$columns}) VALUES ({$placeholders})
                ON DUPLICATE KEY UPDATE {$updateSql}";

        self::stmt($sql)->execute($data);
        return self::stmt($sql)->rowCount();
    }

    /**
     * Update rows
     *
     * @param string $table Table name
     * @param array<string, mixed> $data Key-value pairs to update
     * @param string $where WHERE clause
     * @param array $params WHERE parameters
     * @return int Affected rows
     */
    public static function update(string $table, array $data, string $where, array $params = []): int
    {
        $setParts = [];
        foreach (array_keys($data) as $col) {
            $setParts[] = "{$col} = ?";
        }
        $setSql = implode(', ', $setParts);

        $sql = "UPDATE {$table} SET {$setSql} WHERE {$where}";
        $params = array_merge(array_values($data), $params);

        self::stmt($sql)->execute($params);
        return self::stmt($sql)->rowCount();
    }

    /**
     * Delete rows
     *
     * @param string $table Table name
     * @param string $where WHERE clause
     * @param array $params WHERE parameters
     * @return int Affected rows
     */
    public static function delete(string $table, string $where, array $params = []): int
    {
        $sql = "DELETE FROM {$table} WHERE {$where}";
        self::stmt($sql)->execute($params);
        return self::stmt($sql)->rowCount();
    }

    /**
     * Begin transaction
     */
    public static function begin(): void
    {
        if (!self::$inTransaction) {
            self::get()->beginTransaction();
            self::$inTransaction = true;
        }
    }

    /**
     * Commit transaction
     */
    public static function commit(): void
    {
        if (self::$inTransaction) {
            self::get()->commit();
            self::$inTransaction = false;
        }
    }

    /**
     * Rollback transaction
     */
    public static function rollback(): void
    {
        if (self::$inTransaction) {
            self::get()->rollBack();
            self::$inTransaction = false;
        }
    }

    /**
     * Execute callback in transaction
     *
     * @param callable $callback Function that receives ApexDB and returns value
     * @return mixed
     * @throws Throwable
     */
    public static function transaction(callable $callback): mixed
    {
        self::begin();
        try {
            $result = $callback(self::get());
            self::commit();
            return $result;
        } catch (Throwable $e) {
            self::rollback();
            throw $e;
        }
    }

    /**
     * Clear statement cache
     */
    public static function clearStmtCache(): void
    {
        foreach (self::$stmtCache as $stmt) {
            $stmt->closeCursor();
        }
        self::$stmtCache = [];
    }

    /**
     * Get last insert ID
     */
    public static function lastInsertId(): int|string
    {
        return self::get()->lastInsertId();
    }

    /**
     * Escape string for LIKE query
     *
     * @param string $value Value to escape
     * @param string $wildcard Wildcard character (default: %)
     * @return string
     */
    public static function escapeLike(string $value, string $wildcard = '%'): string
    {
        $value = str_replace(['\\', $wildcard, '_'], ['\\\\', '\\' . $wildcard, '\\_'], $value);
        return $wildcard . $value . $wildcard;
    }

    /**
     * Build IN clause với placeholders
     *
     * @param string $column Column name
     * @param int $count Number of values
     * @return string e.g., "column IN (?,?,?)"
     */
    public static function buildInClause(string $column, int $count): string
    {
        return "{$column} IN (" . implode(',', array_fill(0, $count, '?')) . ")";
    }

    /**
     * Batch insert (multi-row INSERT)
     *
     * @param string $table Table name
     * @param array<int, array<string, mixed>> $rows Array of rows
     * @return int Affected rows
     */
    public static function batchInsert(string $table, array $rows): int
    {
        if (empty($rows)) {
            return 0;
        }

        $columns = implode(', ', array_keys($rows[0]));
        $placeholders = '(' . implode(', ', array_fill(0, count($rows[0]), '?')) . ')';
        $allPlaceholders = implode(', ', array_fill(0, count($rows), $placeholders));

        $sql = "INSERT INTO {$table} ({$columns}) VALUES {$allPlaceholders}";

        $values = [];
        foreach ($rows as $row) {
            foreach ($row as $val) {
                $values[] = $val;
            }
        }

        self::stmt($sql)->execute($values);
        return self::stmt($sql)->rowCount();
    }
}

/**
 * Helper: Get PDO for backwards compatibility
 */
function apex_db(): PDO
{
    return ApexDB::get();
}
