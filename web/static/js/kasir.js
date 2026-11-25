(() => {
  // ---------- state ----------
  const cart = new Map(); // kunci: unit_id (angka)
  const selectedPayment = { method: 'Tunai' };
  let hasilCari = []; // hasil pencarian terakhir dari server
  let total = 0;
  let sedangMemproses = false;
  let timerCari = null;
  let nomorCari = 0;

  // ---------- elemen ----------
  const searchInput = document.getElementById('medicine-search');
  const rowsBody = document.getElementById('medicine-rows');
  const hint = document.getElementById('medicine-hint');
  const cartBody = document.getElementById('cart-items');
  const amountInput = document.getElementById('amount-paid');
  const totalOutput = document.getElementById('grand-total');
  const countOutput = document.getElementById('cart-count');
  const emptyState = document.getElementById('cart-empty');
  const changeBanner = document.querySelector('.change-banner');
  const paymentMessage = document.getElementById('payment-message');
  const receiptDialog = document.getElementById('receipt-dialog');
  const receiptLines = document.getElementById('receipt-lines');
  const checkoutButton = document.getElementById('checkout-button');

  // ---------- util ----------
  const money = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value).replace(/\u00a0/g, ' ');
  const angka = value => new Intl.NumberFormat('id-ID', { maximumFractionDigits: 3 }).format(value);
  const escapeHTML = value => String(value).replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
  const bulat2 = value => Math.round(value * 100) / 100;
  const setText = (id, value) => { const node = document.getElementById(id); if (node) node.textContent = value; };
  const toast = (message, title) => window.apotekToast?.(message, title);

  // api memanggil backend dan mengembalikan isi "data".
  // Jika sesi habis, server mengalihkan ke /login (fetch mengikutinya otomatis),
  // jadi pengalihan itu dikenali dan browser dikirim ke halaman login.
  async function api(url, options = {}) {
    const res = await fetch(url, {
      ...options,
      headers: { Accept: 'application/json', ...(options.body ? { 'Content-Type': 'application/json' } : {}) },
    });

    if (res.status === 401 || (res.redirected && new URL(res.url).pathname === '/login')) {
      window.location.href = '/login';
      throw new Error('Sesi berakhir. Silakan login kembali.');
    }

    const isJSON = (res.headers.get('Content-Type') || '').includes('application/json');
    const body = isJSON ? await res.json() : null;

    if (res.status === 403) throw new Error('Akun ini tidak berhak melakukan aksi tersebut.');
    if (!res.ok) throw new Error(body?.error || `Permintaan gagal (${res.status}).`);
    if (!isJSON) throw new Error('Respons server tidak dikenali.');
    return body.data;
  }

  // ---------- pencarian obat ----------
  function setHint(message) {
    if (!hint) return;
    hint.textContent = message;
    hint.hidden = !message;
  }

  const cariObat = id => hasilCari.find(obat => String(obat.id) === String(id));
  const cariSatuan = (obat, unitId) => obat?.satuan.find(satuan => String(satuan.id) === String(unitId));
  const bisaDijual = (obat, satuan) => Boolean(satuan) && Number(obat.stok_tersedia) >= Number(satuan.isi_per_satuan_dasar);

  function stokTeks(obat, satuan) {
    const stok = Number(obat.stok_tersedia);
    const isi = Number(satuan?.isi_per_satuan_dasar || 1);
    if (isi <= 1) return `${angka(stok)} ${obat.satuan_dasar}`;
    return `${angka(Math.floor(stok / isi))} ${satuan.nama} (${angka(stok)} ${obat.satuan_dasar})`;
  }

  // Total satuan dasar obat ini yang sudah ada di keranjang (semua satuan dijumlahkan).
  const dasarDiKeranjang = medicineId => [...cart.values()]
    .filter(item => String(item.medicineId) === String(medicineId))
    .reduce((sum, item) => sum + item.qty * item.isi, 0);

  const melebihiStok = (medicineId, tambahDasar, stokDasar) =>
    bulat2(dasarDiKeranjang(medicineId) + tambahDasar) > Number(stokDasar);

  function gambarHasil() {
    if (!rowsBody) return;
    rowsBody.innerHTML = hasilCari.map(obat => {
      const satuan = obat.satuan[0];
      if (!satuan) {
        return `<tr><td class="product-code">${escapeHTML(obat.kode)}</td><td><strong>${escapeHTML(obat.nama)}</strong></td><td colspan="5"><span class="stock-chip unavailable">Belum ada satuan jual</span></td></tr>`;
      }
      const tersedia = bisaDijual(obat, satuan);
      const opsi = obat.satuan.map(item => `<option value="${item.id}">${escapeHTML(item.nama)}</option>`).join('');
      return `
        <tr>
          <td class="product-code">${escapeHTML(obat.kode)}</td>
          <td><strong>${escapeHTML(obat.nama)}</strong></td>
          <td><select class="unit-select" data-unit-select="${obat.id}" aria-label="Satuan ${escapeHTML(obat.nama)}">${opsi}</select></td>
          <td class="price-cell" data-price-cell>${money(satuan.harga_jual)}</td>
          <td><span class="stock-chip ${tersedia ? 'available' : 'unavailable'}" data-stock-chip>${tersedia ? escapeHTML(stokTeks(obat, satuan)) : 'Stok tidak cukup'}</span></td>
          <td><div class="qty-stepper"><button type="button" data-product-step="-1" aria-label="Kurangi jumlah">−</button><input type="number" min="0" value="1" aria-label="Jumlah ${escapeHTML(obat.nama)}" data-product-qty="${obat.id}"><button type="button" data-product-step="1" aria-label="Tambah jumlah">+</button></div></td>
          <td><button type="button" class="btn btn-primary btn-sm add-product" data-add-cart="${obat.id}" ${tersedia ? '' : 'disabled'}>Tambah</button></td>
        </tr>`;
    }).join('');
  }

  async function cari() {
    const kata = (searchInput?.value || '').trim();
    if (kata.length === 1) { // 1 huruf terlalu pendek: biarkan daftar yang sedang tampil
      setHint('Ketik minimal 2 huruf untuk mencari obat.');
      return;
    }
    const nomor = ++nomorCari;
    try {
      // Kata kunci kosong = daftar obat awal dari database.
      const data = await api(kata ? `/kasir/obat?q=${encodeURIComponent(kata)}` : '/kasir/obat');
      if (nomor !== nomorCari) return; // sudah ada pencarian yang lebih baru
      hasilCari = data || [];
      gambarHasil();
      setHint(hasilCari.length ? '' : (kata ? 'Obat tidak ditemukan.' : 'Belum ada obat aktif di database.'));
    } catch (err) {
      if (nomor !== nomorCari) return;
      hasilCari = [];
      gambarHasil();
      setHint(`Gagal memuat obat: ${err.message}`); // tetap terlihat, tidak hilang seperti toast
      toast(err.message, 'Pencarian obat');
    }
  }

  searchInput?.addEventListener('input', () => {
    clearTimeout(timerCari);
    timerCari = setTimeout(cari, 300);
  });

  // Ganti satuan: harga dan stok pada baris ikut berubah.
  document.addEventListener('change', event => {
    const select = event.target.closest('[data-unit-select]');
    if (!select) return;
    const obat = cariObat(select.dataset.unitSelect);
    const satuan = cariSatuan(obat, select.value);
    const row = select.closest('tr');
    if (!obat || !satuan || !row) return;

    const tersedia = bisaDijual(obat, satuan);
    row.querySelector('[data-price-cell]').textContent = money(satuan.harga_jual);
    const chip = row.querySelector('[data-stock-chip]');
    chip.textContent = tersedia ? stokTeks(obat, satuan) : 'Stok tidak cukup';
    chip.classList.toggle('available', tersedia);
    chip.classList.toggle('unavailable', !tersedia);
    row.querySelector('[data-add-cart]').disabled = !tersedia;
  });

  // ---------- keranjang ----------
  function drawCart() {
    if (!cartBody) return;
    const items = [...cart.values()];
    total = bulat2(items.reduce((sum, item) => sum + item.price * item.qty, 0));
    cartBody.innerHTML = items.map(item => `
      <tr>
        <td><strong>${escapeHTML(item.name)}</strong></td>
        <td>${escapeHTML(item.unitName)}</td>
        <td><span class="cart-qty"><button type="button" data-cart-step="-1" data-key="${item.unitId}" aria-label="Kurangi">−</button>${angka(item.qty)}<button type="button" data-cart-step="1" data-key="${item.unitId}" aria-label="Tambah">+</button></span></td>
        <td>${money(item.price)}</td>
        <td><strong>${money(bulat2(item.price * item.qty))}</strong></td>
        <td><button type="button" class="cart-remove" data-cart-remove="${item.unitId}" aria-label="Hapus ${escapeHTML(item.name)}">×</button></td>
      </tr>`).join('');
    if (emptyState) emptyState.hidden = items.length > 0;
    if (cartBody.closest('.cart-table-wrap')) cartBody.closest('.cart-table-wrap').classList.toggle('is-empty', items.length === 0);
    if (countOutput) countOutput.textContent = `${items.length} Item`;
    if (totalOutput) totalOutput.textContent = money(total);
    updatePayment();
  }

  function updatePayment() {
    const paid = Number(amountInput?.value || 0);
    const difference = paid - total;
    const changeValue = document.getElementById('change-value');
    if (!changeValue || !changeBanner || !paymentMessage) return;
    changeBanner.classList.toggle('is-short', selectedPayment.method === 'Tunai' && difference < 0);
    if (selectedPayment.method === 'Nontunai') {
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

  // ---------- transaksi ----------
  function tampilkanStruk(struk) {
    setText('receipt-no', struk.no_transaksi || `#${struk.sale_id}`);
    setText('receipt-time', new Date(struk.tanggal).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }));
    setText('receipt-cashier', struk.kasir);
    if (receiptLines) {
      receiptLines.innerHTML = struk.items.map(item => `<div class="receipt-modal-line"><span>${escapeHTML(item.nama_obat)}<br>${angka(item.qty)} ${escapeHTML(item.nama_satuan)} × ${money(item.harga_satuan)}</span><strong>${money(item.subtotal)}</strong></div>`).join('');
    }
    setText('receipt-total', money(struk.total));
    setText('receipt-paid', money(struk.jumlah_bayar));
    setText('receipt-change', money(struk.kembalian));
    const link = document.getElementById('receipt-open-page');
    if (link) link.href = `${link.dataset.base}?id=${encodeURIComponent(struk.sale_id)}`;
    receiptDialog?.showModal();
  }

  async function prosesTransaksi() {
    if (sedangMemproses) return; // cegah klik ganda yang membuat dua transaksi
    if (!cart.size) {
      toast('Keranjang masih kosong. Tambahkan obat terlebih dahulu.', 'Kasir');
      return;
    }
    const tunai = selectedPayment.method === 'Tunai';
    const dibayar = tunai ? Number(amountInput?.value || 0) : total;
    if (tunai && dibayar < total) {
      toast('Jumlah pembayaran belum mencukupi.', 'Kasir');
      return;
    }

    sedangMemproses = true;
    if (checkoutButton) checkoutButton.disabled = true;
    let hasil;
    try {
      hasil = await api('/kasir/transaksi', {
        method: 'POST',
        body: JSON.stringify({
          metode_bayar: tunai ? 'TUNAI' : 'NONTUNAI',
          jumlah_bayar: dibayar,
          items: [...cart.values()].map(item => ({ unit_id: item.unitId, qty: item.qty })),
        }),
      });
    } catch (err) {
      toast(err.message, 'Transaksi gagal');
      sedangMemproses = false;
      if (checkoutButton) checkoutButton.disabled = false;
      return;
    }

    // Transaksi sudah tersimpan: keranjang dikosongkan lebih dulu supaya tidak terkirim dua kali.
    cart.clear();
    if (amountInput) amountInput.value = '0';
    drawCart();
    cari(); // segarkan angka stok

    try {
      tampilkanStruk(await api(`/kasir/struk/${hasil.sale_id}`));
    } catch (err) {
      toast(`Transaksi tersimpan (nomor ${hasil.sale_id}), tetapi struk gagal dimuat: ${err.message}`, 'Kasir');
    } finally {
      sedangMemproses = false;
      if (checkoutButton) checkoutButton.disabled = false;
    }
  }

  // ---------- klik ----------
  document.addEventListener('click', event => {
    const productStep = event.target.closest('[data-product-step]');
    if (productStep) {
      const input = productStep.closest('.qty-stepper')?.querySelector('[data-product-qty]');
      if (input) input.value = String(Math.max(0, Number(input.value || 0) + Number(productStep.dataset.productStep)));
      return;
    }

    const addButton = event.target.closest('[data-add-cart]');
    if (addButton && !addButton.disabled) {
      const obatId = addButton.dataset.addCart;
      const obat = cariObat(obatId);
      const select = rowsBody?.querySelector(`[data-unit-select="${CSS.escape(obatId)}"]`);
      const satuan = cariSatuan(obat, select?.value);
      const input = rowsBody?.querySelector(`[data-product-qty="${CSS.escape(obatId)}"]`);
      if (!obat || !satuan) return;

      const qty = Math.max(1, Number(input?.value || 1));
      const isi = Number(satuan.isi_per_satuan_dasar);
      if (melebihiStok(obat.id, qty * isi, obat.stok_tersedia)) {
        toast(`Stok tidak mencukupi untuk ${obat.nama}.`, 'Kasir');
        return; // tidak masuk keranjang
      }
      const item = cart.get(satuan.id) || { unitId: satuan.id, medicineId: obat.id, name: obat.nama, unitName: satuan.nama, price: Number(satuan.harga_jual), isi, stokDasar: 0, qty: 0 };
      item.stokDasar = Number(obat.stok_tersedia);
      item.qty += qty;
      cart.set(satuan.id, item);
      if (input) input.value = '1';
      drawCart();
      toast(`${obat.nama} ditambahkan ke keranjang.`, 'Kasir');
      return;
    }

    const cartStep = event.target.closest('[data-cart-step]');
    if (cartStep) {
      const key = Number(cartStep.dataset.key);
      const item = cart.get(key);
      if (item) {
        const delta = Number(cartStep.dataset.cartStep || 0);
        if (delta > 0 && melebihiStok(item.medicineId, delta * item.isi, item.stokDasar)) {
          toast(`Stok tidak mencukupi untuk ${item.name}.`, 'Kasir');
          return;
        }
        item.qty += delta;
        if (item.qty <= 0) cart.delete(key);
        drawCart();
      }
      return;
    }

    const removeButton = event.target.closest('[data-cart-remove]');
    if (removeButton) {
      cart.delete(Number(removeButton.dataset.cartRemove));
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
      toast('Keranjang dikosongkan.', 'Kasir');
      return;
    }

    if (event.target.closest('#checkout-button')) prosesTransaksi();
  });

  amountInput?.addEventListener('input', updatePayment);
  drawCart();
  cari(); // muat daftar obat dari database saat halaman dibuka
})();