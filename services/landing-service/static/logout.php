<?php
/**
 * APEX Admin - Logout
 */

define('APEX_ADMIN', true);
require_once __DIR__ . '/includes/auth.php';

apex_admin_logout();
apex_admin_redirect('login.php');
exit;
