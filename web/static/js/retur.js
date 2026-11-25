(() => {
  // ---------- state ----------
  let penjualan = null; // transaksi asal yang sedang ditampilkan
  let sedangMenyimpan = false;

  // ---------- elemen ----------
  const $ = id => document.getElementById(id);
  const idInput = $('return-transaction-id');
  const dateInput = $('return-transaction-date');
  const verified = $('verified-return-transaction');
  const errorBox = $('return-search-error');
  const itemsBody = $('return-items-body');
  const itemCount = $('return-items-count');
  const totalOutput = $('return-refund-total');
  const qtyOutput = $('return-refund-qty');
  const validation = $('return-qty-validation');
  const form = $('return-form');
  const submitButton = $('return-submit');
  const historyBody = $('return-history-body');
  const historyNote = $('return-history-note');
  const teksAwalError = errorBox?.textContent || '';
  // Hanya Owner yang boleh menetapkan kelayakan retur.
  const bolehMenilaiRetur = document.body.dataset.userRole === 'OWNER';

  // ---------- util ----------
  const money = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value).replace(/\u00a0/g, ' ');
  const angka = value => new Intl.NumberFormat('id-ID', { maximumFractionDigits: 3 }).format(value);
  const escapeHTML = value => String(value).replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
  const setText = (id, value) => { const node = $(id); if (node) node.textContent = value; };
  const toast = (message, title) => window.apotekToast?.(message, title);
  const tglPendek = iso => new Date(iso).toLocaleDateString('id-ID', { day: '2-digit', month: 'short', year: 'numeric' });
  const tglWaktu = iso => new Date(iso).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' });

  const jenisLabel = {
    SALAH_MEMBELI: 'Salah membeli atau perubahan resep',
    OBAT_RUSAK: 'Obat rusak atau dugaan masalah mutu',
    KESALAHAN_PENYERAHAN: 'Kesalahan penyerahan oleh apotek',
    PENARIKAN_RESMI: 'Pengembalian akibat penarikan resmi',
  };

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

  // ---------- transaksi asal ----------
  const barisItem = () => [...(itemsBody?.querySelectorAll('[data-return-item]') || [])];

  function kosongkanTabel(pesan) {
    if (itemsBody) itemsBody.innerHTML = `<tr><td colspan="9" class="muted">${escapeHTML(pesan || 'Cari transaksi asal untuk menampilkan obat yang dapat diretur.')}</td></tr>`;
    if (itemCount) itemCount.textContent = '0 Item Terdaftar';
    if (submitButton) submitButton.disabled = true;
    updateRefund();
  }

  function tampilkanPenjualan() {
    setText('verified-transaction-id', penjualan.no_transaksi);
    setText('verified-time', tglWaktu(penjualan.tanggal));
    setText('verified-cashier', penjualan.kasir);
    setText('verified-total', money(penjualan.total));
    if (dateInput) dateInput.value = new Date(penjualan.tanggal).toLocaleDateString('id-ID');
    if (verified) verified.hidden = false;

    const baris = [];
    penjualan.items.forEach(item => item.batches.forEach(batch => baris.push({ item, batch })));
    if (!baris.length) {
      kosongkanTabel('Transaksi ini tidak memiliki rincian obat.');
      return;
    }

    itemsBody.innerHTML = baris.map(({ item, batch }) => {
      const sisa = Number(batch.bisa_diretur);
      const bisa = sisa > 0;
      const nama = escapeHTML(item.nama_obat);
      const dasar = escapeHTML(item.satuan_dasar);
      return `
        <tr data-return-item data-batch-id="${batch.sale_item_batch_id}" data-price="${item.harga_jual_per_dasar}" data-name="${nama}">
          <td><input class="return-check" type="checkbox" aria-label="Pilih ${nama}" ${bisa ? '' : 'disabled'}></td>
          <td><strong>${nama}</strong><small>Dibeli ${angka(item.qty_dibeli)} ${escapeHTML(item.nama_satuan)}</small></td>
          <td><span class="batch-chip">${escapeHTML(batch.no_batch)}</span></td>
          <td>${tglPendek(batch.tgl_kadaluarsa)}</td>
          <td>${angka(batch.qty_dasar)} ${dasar}</td>
          <td>${angka(batch.sudah_diretur)} ${dasar}</td>
          <td class="return-remaining">${angka(sisa)} ${dasar}</td>
          <td>${money(item.harga_jual_per_dasar)}</td>
          <td><div class="return-qty-control"><button type="button" data-qty-change="-1" aria-label="Kurangi">−</button><input type="number" min="0" max="${sisa}" step="any" value="0" data-return-qty aria-label="Qty retur ${nama}" disabled><button type="button" data-qty-change="1" aria-label="Tambah">+</button></div></td>
        </tr>`;
    }).join('');

    if (itemCount) itemCount.textContent = `${baris.length} Item Terdaftar`;
    if (submitButton) submitButton.disabled = !baris.some(({ batch }) => Number(batch.bisa_diretur) > 0);
    updateRefund();
  }

  async function cariTransaksi(nomor) {
    const kata = (nomor ?? idInput?.value ?? '').trim();
    if (!kata) {
      toast('Masukkan nomor transaksi terlebih dahulu.', 'Pencarian transaksi');
      return;
    }
    try {
      penjualan = await api(`/retur/penjualan/${encodeURIComponent(kata)}`);
      if (errorBox) errorBox.hidden = true;
      tampilkanPenjualan();
    } catch (err) {
      penjualan = null;
      if (verified) verified.hidden = true;
      kosongkanTabel();
      if (errorBox) {
        errorBox.textContent = err.message.includes('tidak ditemukan') ? teksAwalError : err.message;
        errorBox.hidden = false;
      }
    }
  }

  // ---------- pemilihan dan estimasi refund ----------
  function updateRefund() {
    let total = 0;
    let selectedQty = 0;
    let invalid = false;
    barisItem().forEach(row => {
      const checkbox = row.querySelector('.return-check');
      const quantity = row.querySelector('[data-return-qty]');
      const checked = checkbox.checked;
      quantity.disabled = !checked;
      const max = Number(quantity.max || 0);
      let value = Math.max(0, Number(quantity.value || 0));
      if (value > max) { value = max; quantity.value = String(max); invalid = true; }
      if (!checked) { quantity.value = '0'; value = 0; }
      total += checked ? Math.round(value * Number(row.dataset.price || 0) * 100) / 100 : 0;
      selectedQty += checked ? value : 0;
      row.classList.toggle('is-selected', checked);
    });
    if (totalOutput) totalOutput.textContent = money(total);
    if (qtyOutput) qtyOutput.textContent = angka(selectedQty);
    if (validation) validation.hidden = !invalid;
  }

  document.addEventListener('change', event => {
    if (!event.target.matches('.return-check,[data-return-qty]')) return;
    // Saat baris baru dicentang, isi jumlah awal 1 (atau sisa jika kurang dari 1).
    if (event.target.matches('.return-check') && event.target.checked) {
      const input = event.target.closest('[data-return-item]')?.querySelector('[data-return-qty]');
      if (input && Number(input.value || 0) === 0) input.value = String(Math.min(1, Number(input.max || 0)));
    }
    updateRefund();
  });

  // ---------- simpan retur ----------
  async function simpanRetur(event) {
    event.preventDefault();
    if (sedangMenyimpan) return;
    if (!penjualan) {
      toast('Cari transaksi asal terlebih dahulu.', 'Retur konsumen');
      return;
    }

    const dipilih = barisItem().filter(row => row.querySelector('.return-check').checked)
      .map(row => ({
        id: Number(row.dataset.batchId),
        nama: row.dataset.name,
        qty: Number(row.querySelector('[data-return-qty]').value || 0),
      }))
      .filter(item => item.qty > 0);
    if (!dipilih.length) {
      toast('Pilih minimal satu obat dengan jumlah retur lebih dari 0.', 'Retur konsumen');
      return;
    }

    const jenis = form.elements['return_type'].value;
    const kondisi = form.elements['condition'].value;
    const alasan = form.elements['reason'].value.trim();

    sedangMenyimpan = true;
    if (submitButton) submitButton.disabled = true;
    const berhasil = [];
    const gagal = [];
    for (const item of dipilih) {
      try {
        berhasil.push(await api('/retur', {
          method: 'POST',
          body: JSON.stringify({ sale_item_batch_id: item.id, qty_dasar: item.qty, jenis_pengembalian: jenis, alasan, kondisi_obat: kondisi }),
        }));
      } catch (err) {
        gagal.push(`${item.nama}: ${err.message}`);
      }
    }

    const pesan = [];
    if (berhasil.length) {
      const refund = berhasil.reduce((sum, hasil) => sum + Number(hasil.nilai_refund), 0);
      pesan.push(`${berhasil.length} retur tersimpan, total refund ${money(refund)}.`);
    }
    if (gagal.length) pesan.push(`Gagal: ${gagal.join('; ')}`);
    toast(pesan.join(' '), 'Retur konsumen');

    if (!gagal.length) form.elements['reason'].value = '';
    sedangMenyimpan = false;
    await cariTransaksi(penjualan.no_transaksi); // segarkan sisa yang bisa diretur
    muatRiwayat();
  }

  form?.addEventListener('submit', simpanRetur);

  // ---------- riwayat dan penetapan kelayakan ----------
  const statusPill = item => {
    if (item.status_kelayakan === 'LAYAK_JUAL') return '<span class="status-pill safe">✓ Layak Jual</span>';
    if (item.status_kelayakan === 'TIDAK_LAYAK_JUAL') return '<span class="status-pill danger">× Tidak Layak Jual</span>';
    if (!bolehMenilaiRetur) return '<span class="status-pill warning">Menunggu Pemeriksaan</span>';
    const data = `data-retur-id="${item.retur_id}" data-no="${escapeHTML(item.no_retur)}"`;
    return `<span class="status-pill warning">◷ Menunggu Pemeriksaan</span>
      <div class="kelayakan-actions">
        <button type="button" class="btn btn-outline btn-sm" data-kelayakan="LAYAK_JUAL" ${data}>Layak jual</button>
        <button type="button" class="btn btn-danger-soft btn-sm" data-kelayakan="TIDAK_LAYAK_JUAL" ${data}>Tidak layak</button>
      </div>`;
  };

  async function muatRiwayat() {
    if (!historyBody) return;
    try {
      const daftar = await api('/retur/riwayat?limit=10');
      historyBody.innerHTML = daftar.length ? daftar.map(item => `
        <tr>
          <td class="mono">${escapeHTML(item.no_retur)}</td>
          <td>${tglWaktu(item.tanggal)}</td>
          <td>${escapeHTML(item.nama_obat)}</td>
          <td><span class="batch-chip">${escapeHTML(item.no_batch)}</span></td>
          <td>${angka(item.qty_dasar)} ${escapeHTML(item.satuan_dasar)}</td>
          <td>${escapeHTML(jenisLabel[item.jenis_pengembalian] || item.jenis_pengembalian)}</td>
          <td>${statusPill(item)}</td>
          <td><strong>${money(item.nilai_refund)}</strong></td>
        </tr>`).join('') : '<tr><td colspan="8" class="muted">Belum ada retur.</td></tr>';
      if (historyNote) historyNote.textContent = daftar.length ? `Menampilkan ${daftar.length} entri terakhir` : '';
    } catch (err) {
      toast(err.message, 'Riwayat retur');
    }
  }

  async function tetapkanKelayakan(tombol) {
    const status = tombol.dataset.kelayakan;
    const label = status === 'LAYAK_JUAL' ? 'LAYAK JUAL' : 'TIDAK LAYAK JUAL';
    if (!window.confirm(`Tetapkan retur ${tombol.dataset.no} sebagai ${label}? Keputusan ini tidak dapat diubah.`)) return;
    try {
      const hasil = await api(`/retur/${tombol.dataset.returId}/kelayakan`, {
        method: 'POST',
        body: JSON.stringify({ status_kelayakan: status }),
      });
      toast(hasil.stok_ditambah ? 'Ditetapkan layak jual. Stok dikembalikan ke lapisan penerimaan asal.' : 'Ditetapkan tidak layak jual. Stok tidak berubah.', 'Retur konsumen');
    } catch (err) {
      toast(err.message, 'Retur konsumen');
    }
    muatRiwayat();
    if (penjualan) cariTransaksi(penjualan.no_transaksi);
  }

  // ---------- klik ----------
  document.addEventListener('click', event => {
    const step = event.target.closest('[data-qty-change]');
    if (step) {
      const row = step.closest('[data-return-item]');
      const input = row?.querySelector('[data-return-qty]');
      if (!input || input.disabled) return;
      const next = Math.max(0, Math.min(Number(input.max || 0), Number(input.value || 0) + Number(step.dataset.qtyChange)));
      input.value = String(Math.round(next * 1000) / 1000);
      updateRefund();
      return;
    }

    const kelayakan = event.target.closest('[data-kelayakan]');
    if (kelayakan) {
      tetapkanKelayakan(kelayakan);
      return;
    }

    if (event.target.closest('#find-return-transaction')) cariTransaksi();
  });

  idInput?.addEventListener('keydown', event => {
    if (event.key === 'Enter') {
      event.preventDefault();
      cariTransaksi();
    }
  });

  kosongkanTabel();
  muatRiwayat();
})();