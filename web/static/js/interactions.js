(() => {
  const toastRegion = document.getElementById('toast-region');
  function toast(message, title = 'Mode demo') {
    if (!toastRegion) return;
    const node = document.createElement('div');
    node.className = 'toast-message';
    const heading = document.createElement('strong'); heading.textContent = title;
    const body = document.createElement('span'); body.textContent = message;
    node.append(heading, body); toastRegion.appendChild(node);
    window.setTimeout(() => node.remove(), 3600);
  }
  document.addEventListener('click', (event) => {
    const toastButton = event.target.closest('[data-toast]');
    if (toastButton) { event.preventDefault(); toast(toastButton.dataset.toast); }
    const openButton = event.target.closest('[data-open-dialog]');
    if (openButton) {
      const dialog = document.getElementById(openButton.dataset.openDialog);
      if (dialog?.id === 'treatment-dialog') {
        const selectedName = dialog.querySelector('#expiry-selected-name');
        const selectedBatch = dialog.querySelector('#expiry-selected-batch');
        if (selectedName && openButton.dataset.expiryItem) selectedName.textContent = openButton.dataset.expiryItem;
        if (selectedBatch && openButton.dataset.expiryBatch) selectedBatch.textContent = openButton.dataset.expiryBatch;
      }
      if (dialog?.showModal) dialog.showModal();
    }
    const closeButton = event.target.closest('[data-close-dialog]');
    if (closeButton) closeButton.closest('dialog')?.close();
    const removeButton = event.target.closest('[data-remove-row]');
    if (removeButton) { const row = removeButton.closest('tr'); const table = row?.closest('table'); row?.remove(); const count = document.getElementById('receiving-row-count'); if (count && table) { const unit = count.dataset.countUnit || 'baris'; count.textContent = `${table.querySelectorAll('tbody tr').length} ${unit}`; }
      document.dispatchEvent(new CustomEvent('apotek:rows-changed')); }
    const addRowButton = event.target.closest('[data-add-row]');
    if (addRowButton) {
      const table = document.getElementById(addRowButton.dataset.addRow);
      const tbody = table?.querySelector('tbody');
      const firstRow = tbody?.querySelector('tr');
      if (tbody && firstRow) {
        const row = firstRow.cloneNode(true);
        row.querySelectorAll('input').forEach(input => {
          if (input.type === 'number') input.value = input.classList.contains('receiving-qty') ? '1' : '0';
          else input.value = '';
        });
        row.querySelectorAll('select').forEach(select => { select.selectedIndex = 0; });
        row.querySelectorAll('.receiving-base-total').forEach(node => { node.textContent = '1'; });
        row.querySelectorAll('.receiving-base-price').forEach(node => { node.textContent = 'Rp0'; });
        tbody.appendChild(row);
        const count = document.getElementById('receiving-row-count');
        if (count) { const unit = count.dataset.countUnit || 'Item'; count.textContent = `${tbody.querySelectorAll('tr').length} ${unit}`; }
        document.dispatchEvent(new CustomEvent('apotek:rows-changed'));
      }
    }
    const exportButton = event.target.closest('[data-export-table]');
    if (exportButton) exportTable(exportButton.dataset.exportTable);
    if (event.target.closest('[data-print]')) window.print();
  });
  document.addEventListener('submit', (event) => {
    const form = event.target.closest('[data-demo-form]');
    if (!form) return;
    event.preventDefault();
    if (!form.reportValidity()) return;
    toast(form.dataset.success || 'Aksi berhasil divalidasi dalam mode demo. Data belum disimpan.');
    if (form.closest('dialog')) form.closest('dialog').close();
  });
  document.addEventListener('input', (event) => {
    const input = event.target.closest('[data-filter-input]');
    if (input) filterRows(input.dataset.filterInput);
  });
  document.addEventListener('change', (event) => {
    const select = event.target.closest('[data-filter-select]');
    if (select) filterRows(select.dataset.filterSelect);
  });
  function filterRows(id) {
    const tableOrList = document.getElementById(id);
    if (!tableOrList) return;
    const textInput = document.querySelector(`[data-filter-input="${id}"]`);
    const statusInput = document.querySelector(`[data-filter-select="${id}"]`);
    const query = (textInput?.value || '').toLowerCase().trim();
    const status = statusInput?.value || '';
    tableOrList.querySelectorAll('tbody tr, .medicine-card').forEach((row) => {
      const haystack = `${row.dataset.search || ''} ${row.textContent}`.toLowerCase();
      const rowStatus = row.dataset.status || '';
      row.hidden = !(haystack.includes(query) && (!status || rowStatus === status || row.textContent.includes(status)));
    });
  }
  function exportTable(id) {
    const table = document.getElementById(id); if (!table) return;
    const rows = [...table.querySelectorAll('tr')].filter(row => !row.hidden);
    const csv = rows.map(row => [...row.children].map(cell => `"${cell.innerText.replace(/"/g, '""').trim()}"`).join(',')).join('\r\n');
    const blob = new Blob([csv], {type:'text/csv;charset=utf-8;'}); const url = URL.createObjectURL(blob);
    const anchor = document.createElement('a'); anchor.href = url; anchor.download = `${id}.csv`; anchor.click(); URL.revokeObjectURL(url);
  }
  window.apotekToast = toast;
})();
