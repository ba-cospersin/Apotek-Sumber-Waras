(function () {
  const page = document.getElementById('expiry-page');
  if (!page) return;

  const banner = document.getElementById('expiry-success');
  const bannerClose = document.getElementById('expiry-banner-close');
  const dialog = document.getElementById('disposal-dialog');
  const amount = document.getElementById('expiry-amount');
  const compensation = document.getElementById('expiry-compensation');
  const hint = document.getElementById('expiry-amount-hint');
  const validation = document.getElementById('expiry-validation');
  const lossBox = document.getElementById('expiry-loss');
  const lossTitle = document.getElementById('expiry-loss-title');
  const saveButton = document.getElementById('expiry-modal-save');
  const newRow = document.getElementById('expiry-history-new');
  const historySub = document.getElementById('expiry-history-sub');
  const segment = document.getElementById('expiry-action-type');
  const textarea = page.querySelector('.expiry-textarea');

  const MAX_LAYER_STOCK = 5;
  const PURCHASE_PRICE = 9000;

  function formatRupiah(value) {
    return 'Rp' + Math.max(0, value).toLocaleString('id-ID');
  }

  function setInvalid(isInvalid) {
    if (validation) validation.hidden = !isInvalid;
    if (hint) hint.hidden = isInvalid;
    if (amount) amount.classList.toggle('is-invalid', isInvalid);
    if (lossBox) lossBox.classList.toggle('is-muted', isInvalid);
    if (saveButton) saveButton.disabled = isInvalid;
    if (lossTitle) {
      lossTitle.textContent = isInvalid
        ? 'Nilai Kerugian: -'
        : 'Nilai Kerugian: ' + formatRupiah(PURCHASE_PRICE * parsedAmount() - parsedCompensation());
    }
  }

  function parsedAmount() {
    return parseInt(String(amount && amount.value || '').replace(/\D/g, ''), 10) || 0;
  }

  function parsedCompensation() {
    return parseInt(String(compensation && compensation.value || '').replace(/\D/g, ''), 10) || 0;
  }

  function update() {
    setInvalid(parsedAmount() > MAX_LAYER_STOCK);
  }

  function resetModal() {
    if (amount) amount.value = '5';
    if (compensation) compensation.value = '0';
    if (textarea) textarea.value = '';
    if (segment) {
      segment.querySelectorAll('.expiry-segment-btn').forEach((button, index) => {
        button.classList.toggle('is-active', index === 0);
      });
    }
    update();
  }

  function openDialog() {
    if (!dialog) return;
    if (!dialog.open) dialog.showModal();
  }

  function showSuccess() {
    if (banner) banner.hidden = false;
    if (newRow) newRow.hidden = false;
    if (historySub) historySub.textContent = '4 transaksi tindak lanjut terakhir';
    page.dataset.state = 'success';
  }

  function filterBatches() {
    const textInput = page.querySelector('[data-filter-input="expiry-batch-table"]');
    const statusSelect = page.querySelector('[data-filter-select="expiry-batch-table"]');
    const query = (textInput && textInput.value || '').toLowerCase().trim();
    const status = statusSelect && statusSelect.value || '';
    page.querySelectorAll('#expiry-batch-table .expiry-brow').forEach((row) => {
      const haystack = `${row.dataset.search || ''} ${row.textContent}`.toLowerCase();
      row.hidden = !(haystack.includes(query) && (!status || row.dataset.status === status));
    });
  }

  page.querySelectorAll('[data-open-dialog="disposal-dialog"]').forEach((button) => {
    button.addEventListener('click', resetModal);
  });

  if (bannerClose) {
    bannerClose.addEventListener('click', () => {
      if (banner) banner.hidden = true;
    });
  }

  if (amount) amount.addEventListener('input', update);
  if (compensation) compensation.addEventListener('input', update);

  if (segment) {
    segment.addEventListener('click', (event) => {
      const button = event.target.closest('.expiry-segment-btn');
      if (!button) return;
      segment.querySelectorAll('.expiry-segment-btn').forEach((item) => {
        item.classList.toggle('is-active', item === button);
      });
    });
  }

  if (saveButton) {
    saveButton.addEventListener('click', () => {
      if (saveButton.disabled) return;
      if (dialog) dialog.close();
      showSuccess();
    });
  }

  page.querySelectorAll('[data-filter-input="expiry-batch-table"]').forEach((input) => {
    input.addEventListener('input', filterBatches);
  });
  page.querySelectorAll('[data-filter-select="expiry-batch-table"]').forEach((select) => {
    select.addEventListener('change', filterBatches);
  });
  page.querySelectorAll('.expiry-apply').forEach((button) => {
    button.addEventListener('click', filterBatches);
  });

  const state = new URLSearchParams(window.location.search).get('state');
  if (state === 'success') {
    showSuccess();
  } else if (state === 'validation') {
    resetModal();
    if (amount) amount.value = '8';
    update();
    openDialog();
  } else if (state === 'modal') {
    resetModal();
    openDialog();
  } else {
    update();
  }
})();
