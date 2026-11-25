(() => {
  const $ = id => document.getElementById(id);
  const money = value => new Intl.NumberFormat('id-ID', { style: 'currency', currency: 'IDR', maximumFractionDigits: 0 }).format(value).replace(/\u00a0/g, ' ');
  const angka = value => new Intl.NumberFormat('id-ID', { maximumFractionDigits: 3 }).format(value);
  const escapeHTML = value => String(value).replace(/[&<>"']/g, char => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[char]));
  const setText = (id, value) => { const node = $(id); if (node) node.textContent = value; };
  const metode = { TUNAI: 'Tunai', NONTUNAI: 'Nontunai' };

  function pesan(teks) {
    const node = $('receipt-status');
    if (!node) return;
    node.textContent = teks;
    node.hidden = !teks;
  }

  // Mengambil data struk. Jika sesi habis, server mengalihkan ke /login
  // (fetch mengikutinya otomatis), jadi pengalihan itu dikenali di sini.
  async function ambilStruk(id) {
    const res = await fetch(`/kasir/struk/${id}`, { headers: { Accept: 'application/json' } });

    if (res.status === 401 || (res.redirected && new URL(res.url).pathname === '/login')) {
      window.location.href = '/login';
      throw new Error('Sesi berakhir. Silakan login kembali.');
    }

    const isJSON = (res.headers.get('Content-Type') || '').includes('application/json');
    const body = isJSON ? await res.json() : null;

    if (res.status === 403) throw new Error('Akun ini tidak berhak melihat struk.');
    if (!res.ok) throw new Error(body?.error || `Permintaan gagal (${res.status}).`);
    if (!isJSON) throw new Error('Respons server tidak dikenali.');
    return body.data;
  }

  async function muat() {
    const id = new URLSearchParams(window.location.search).get('id');
    if (!/^\d+$/.test(id || '')) {
      pesan('Nomor penjualan tidak valid. Buka struk dari halaman Kasir.');
      return;
    }

    try {
      const struk = await ambilStruk(id);
      setText('receipt-number', struk.no_transaksi || `#${struk.sale_id}`);
      setText('receipt-date', new Date(struk.tanggal).toLocaleString('id-ID', { dateStyle: 'medium', timeStyle: 'short' }));
      setText('receipt-cashier', struk.kasir);
      setText('receipt-method', metode[struk.metode_bayar] || struk.metode_bayar);

      const lines = $('receipt-lines');
      if (lines) {
        lines.innerHTML = struk.items.map(item => `<div class="receipt-line"><div><strong>${escapeHTML(item.nama_obat)}</strong><small>${angka(item.qty)} ${escapeHTML(item.nama_satuan)} × ${money(item.harga_satuan)}</small></div><strong>${money(item.subtotal)}</strong></div>`).join('');
      }
      setText('receipt-total', money(struk.total));
      setText('receipt-paid', money(struk.jumlah_bayar));
      setText('receipt-change', money(struk.kembalian));

      pesan('');
      const cetak = $('receipt-print');
      if (cetak) cetak.disabled = false;
    } catch (err) {
      pesan(err.message.includes('tidak ditemukan') ? 'Struk tidak ditemukan.' : err.message);
    }
  }

  muat();
})();