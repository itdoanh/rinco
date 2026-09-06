  </div>
</main>
</div>

<div id="toastContainer" class="toast-container"></div>

<script>
window.APEX_ADMIN = {
  csrf: '<?= h($csrf) ?>',
  refreshInterval: 30000,
  baseUrl: '<?= h(ADMIN_BASE) ?>'
};
</script>
<script src="<?= h(apex_admin_url('assets/js/admin.js')) ?>?v=2"></script>
<?php if (!empty($pageScripts)): ?>
  <?php foreach ($pageScripts as $src): ?>
    <script src="<?= h($src) ?>"></script>
  <?php endforeach; ?>
<?php endif; ?>
</body>
</html>
