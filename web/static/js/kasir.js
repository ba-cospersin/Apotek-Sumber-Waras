(() => {
  const cart = new Map();
  const cartBody = document.getElementById('cart-items');
  const amountInput = document.getElementById('amount-paid');
  const totalOutput = document.getElementById('grand-total');
  const countOutput = document.getElementById('cart-count');
  const emptyState = document.getElementById('cart-empty');
  const changeBanner = document.querySelector('.change-banner');
  const paymentMessage = document.getElementById('payment-message');
  const receiptDialog = document.getElementById('receipt-dialog');
  const receiptLines = document.getElementById('receipt-lines');
  const receiptTotal = document.getElementById('receipt-total');
  const selectedPayment = { method: 'Tunai' };
  let total = 0;

  const initialItems = [
    { key: 'Paracetamol 500mg', name: 'Paracetamol 500mg', unit: 'Strip', qty: 2, price: 5000, batch: 'B24-098 (2 strip)' },
    { key: 'Vitamin C 500mg', name: 'Vitamin C 500mg', unit: 'Tablet', qty: 10, price: 1000, batch: 'B24-041 (6 tablet) · B24-055 (4 tablet)' },
    { key: 'Amoxicillin 500mg', name: 'Amoxicillin 500mg', unit: 'Strip', qty: 1, price: 12000, batch: 'B24-098 (1 strip)' }
  ];
  initialItems.forEach(item => cart.set(item.key, item));

  const money = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value).replace(/\u00a0/g, ' ');
  const escapeHTML = value => String(value).replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));

  function drawCart() {
    if (!cartBody) return;
    const items = [...cart.values()];
    total = items.reduce((sum, item) => sum + item.price * item.qty, 0);
    cartBody.innerHTML = items.map(item => `
      <tr>
        <td><strong>${escapeHTML(item.name)}</strong><small>Alokasi FEFO: Batch ${escapeHTML(item.batch)}</small></td>
        <td>${escapeHTML(item.unit)}</td>
        <td><span class="cart-qty"><button type="button" data-cart-step="-1" data-key="${escapeHTML(item.key)}" aria-label="Kurangi">−</button>${item.qty}<button type="button" data-cart-step="1" data-key="${escapeHTML(item.key)}" aria-label="Tambah">+</button></span></td>
        <td>${money(item.price).replace('Rp', 'Rp')}</td>
        <td><strong>${money(item.price * item.qty)}</strong></td>
        <td><button type="button" class="cart-remove" data-cart-remove="${escapeHTML(item.key)}" aria-label="Hapus ${escapeHTML(item.name)}">×</button></td>
      </tr>`).join('');
    if (emptyState) emptyState.hidden = items.length > 0;
    if (cartBody.closest('.cart-table-wrap')) cartBody.closest('.cart-table-wrap').classList.toggle('is-empty', items.length === 0);
    if (countOutput) countOutput.textContent = `${items.length} Item`;
    if (totalOutput) totalOutput.textContent = money(total);
    if (receiptTotal) receiptTotal.textContent = money(total);
    updatePayment();
  }

  function updatePayment() {
    const paid = Number(amountInput?.value || 0);
    const difference = paid - total;
    const changeValue = document.getElementById('change-value');
    if (!changeValue || !changeBanner || !paymentMessage) return;
    changeBanner.classList.toggle('is-short', selectedPayment.method === 'Tunai' && difference < 0);
    if (selectedPayment.method === 'Nontunai') {
      changeValue.textContent = money(0);
      paymentMessage.innerHTML = '<span aria-hidden="true">ⓘ</span>Pastikan pembayaran QRIS atau debit sudah terkonfirmasi sebelum menyelesaikan transaksi.';
      changeBanner.querySelector('span').innerHTML = '<b aria-hidden="true">◉</b> Status pembayaran';
      changeValue.textContent = total > 0 ? 'Menunggu' : money(0);
      return;
    }
    changeBanner.querySelector('span').innerHTML = '<b aria-hidden="true">◉</b> Kembalian';
    if (!total) {
      changeValue.textContent = money(0);
      paymentMessage.innerHTML = '<span aria-hidden="true">ⓘ</span>Keranjang masih kosong. Pilih obat untuk memulai transaksi.';
    } else if (difference < 0) {
      changeValue.textContent = `Kurang ${money(Math.abs(difference))}`;
      paymentMessage.innerHTML = `<span aria-hidden="true">ⓘ</span>Nominal pembayaran masih kurang ${money(Math.abs(difference))}.`;
    } else {
      changeValue.textContent = money(difference);
      paymentMessage.innerHTML = '<span aria-hidden="true">ⓘ</span>Pastikan nominal uang tunai yang diterima pas atau melebihi tagihan sebelum mencetak.';
    }
  }

  document.addEventListener('click', event => {
    const productStep = event.target.closest('[data-product-step]');
    if (productStep) {
      const stepper = productStep.closest('.qty-stepper');
      const input = stepper?.querySelector('[data-product-qty]');
      if (input) input.value = String(Math.max(0, Number(input.value || 0) + Number(productStep.dataset.productStep)));
      return;
    }

    const addButton = event.target.closest('[data-add-cart]');
    if (addButton && !addButton.disabled) {
      const row = addButton.closest('tr');
      const source = addButton.dataset.qtySource;
      const quantityInput = source ? document.querySelector(`[data-product-qty="${CSS.escape(source)}"]`) : null;
      const quantity = Math.max(1, Number(quantityInput?.value || 1));
      const name = addButton.dataset.name;
      const item = cart.get(name) || {
        key: name,
        name,
        price: Number(addButton.dataset.price || 0),
        batch: addButton.dataset.batch || '-',
        unit: row?.querySelector('.unit-select')?.value || addButton.dataset.unit || 'Unit',
        qty: 0
      };
      item.unit = row?.querySelector('.unit-select')?.value || item.unit;
      item.qty += quantity;
      cart.set(name, item);
      if (quantityInput) quantityInput.value = '0';
      drawCart();
      window.apotekToast?.(`${name} ditambahkan ke keranjang.`);
      return;
    }

    const cartStep = event.target.closest('[data-cart-step]');
    if (cartStep) {
      const item = cart.get(cartStep.dataset.key);
      if (item) {
        item.qty += Number(cartStep.dataset.cartStep || 0);
        if (item.qty <= 0) cart.delete(cartStep.dataset.key);
        drawCart();
      }
      return;
    }

    const removeButton = event.target.closest('[data-cart-remove]');
    if (removeButton) {
      cart.delete(removeButton.dataset.cartRemove);
      drawCart();
      return;
    }

    const paymentButton = event.target.closest('[data-payment-method]');
    if (paymentButton) {
      selectedPayment.method = paymentButton.dataset.paymentMethod;
      document.querySelectorAll('[data-payment-method]').forEach(button => button.classList.toggle('is-active', button === paymentButton));
      updatePayment();
      return;
    }

    if (event.target.closest('#clear-cart')) {
      cart.clear();
      drawCart();
      window.apotekToast?.('Keranjang dikosongkan.');
      return;
    }

    if (event.target.closest('#checkout-button')) {
      if (!cart.size) {
        window.apotekToast?.('Keranjang masih kosong. Tambahkan obat terlebih dahulu.');
        return;
      }
      if (selectedPayment.method === 'Tunai' && Number(amountInput?.value || 0) < total) {
        window.apotekToast?.('Jumlah pembayaran belum mencukupi.');
        return;
      }
      if (receiptLines) {
        receiptLines.innerHTML = [...cart.values()].map(item => `<div class="receipt-modal-line"><span>${escapeHTML(item.name)}<br>${item.qty} ${escapeHTML(item.unit)} × ${money(item.price)}</span><strong>${money(item.qty * item.price)}</strong></div>`).join('');
      }
      receiptDialog?.showModal();
    }
  });

  amountInput?.addEventListener('input', updatePayment);
  drawCart();
})();
