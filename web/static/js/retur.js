(() => {
  const rows = [...document.querySelectorAll('[data-return-item]')];
  const totalOutput = document.getElementById('return-refund-total');
  const qtyOutput = document.getElementById('return-refund-qty');
  const validation = document.getElementById('return-qty-validation');
  const formatRupiah = value => `Rp${Math.round(value).toLocaleString('id-ID')}`;

  function updateRefund() {
    let total = 0;
    let selectedQty = 0;
    let invalid = false;
    rows.forEach(row => {
      const checkbox = row.querySelector('.return-check');
      const quantity = row.querySelector('[data-return-qty]');
      const checked = checkbox.checked;
      quantity.disabled = !checked;
      const max = Number(quantity.max || row.dataset.max || 0);
      let value = Math.max(0, Number(quantity.value || 0));
      if (value > max) { value = max; quantity.value = String(max); invalid = true; }
      if (!checked) { quantity.value = '0'; value = 0; }
      total += checked ? value * Number(row.dataset.price || 0) : 0;
      selectedQty += checked ? value : 0;
      row.classList.toggle('is-selected', checked);
    });
    if (totalOutput) totalOutput.textContent = formatRupiah(total);
    if (qtyOutput) qtyOutput.textContent = String(selectedQty);
    if (validation) validation.hidden = !invalid;
  }

  document.addEventListener('change', event => {
    if (event.target.matches('.return-check,[data-return-qty]')) updateRefund();
  });
  document.addEventListener('click', event => {
    const step = event.target.closest('[data-qty-change]');
    if (step) {
      const row = step.closest('[data-return-item]');
      const input = row?.querySelector('[data-return-qty]');
      if (!input || input.disabled) return;
      const next = Math.max(0, Math.min(Number(input.max || 0), Number(input.value || 0) + Number(step.dataset.qtyChange)));
      input.value = String(next);
      updateRefund();
    }
  });

  const searchButton = document.getElementById('find-return-transaction');
  searchButton?.addEventListener('click', () => {
    const idInput = document.getElementById('return-transaction-id');
    const error = document.getElementById('return-search-error');
    const verified = document.getElementById('verified-return-transaction');
    const id = (idInput?.value || '').trim();
    const found = /^TRX-20261107-0042$/i.test(id);
    if (error) error.hidden = found;
    if (verified) verified.hidden = !found;
    const verifiedId = document.getElementById('verified-transaction-id');
    if (found && verifiedId) verifiedId.textContent = id.toUpperCase();
    if (!found) window.apotekToast?.('Transaksi tidak ditemukan. Gunakan TRX-20261107-0042 untuk data contoh.', 'Pencarian transaksi');
  });

  updateRefund();
})();
