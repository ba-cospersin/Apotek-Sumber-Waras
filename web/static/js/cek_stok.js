(() => {
  const table = document.getElementById('batch-table');
  document.addEventListener('click', event => {
    const toggle = event.target.closest('[data-batch-toggle]');
    if (!toggle) return;
    const row = toggle.closest('tr');
    const detail = row?.nextElementSibling;
    if (!detail?.classList.contains('batch-layer-row')) return;
    const open = detail.hidden;
    detail.hidden = !open;
    toggle.classList.toggle('is-open', open);
    toggle.textContent = open ? '⌃' : '⌄';
  });
  const apply = document.getElementById('batch-apply-filter');
  const search = document.getElementById('batch-search-input');
  const status = document.getElementById('batch-status-filter');
  apply?.addEventListener('click', () => {
    const query = (search?.value || '').trim().toLowerCase();
    const selectedStatus = status?.value || '';
    const rows = [...(table?.querySelectorAll('.batch-data-row') || [])];
    rows.forEach(row => {
      const visible = (!query || `${row.dataset.search} ${row.textContent}`.toLowerCase().includes(query)) && (!selectedStatus || row.dataset.status === selectedStatus);
      row.hidden = !visible;
      const detail = row.nextElementSibling;
      if (detail?.classList.contains('batch-layer-row') && !visible) detail.hidden = true;
    });
  });
  search?.addEventListener('keydown', event => { if (event.key === 'Enter') apply?.click(); });
})();