(() => {
  const cart = new Map();
  const rupiah = value => new Intl.NumberFormat('id-ID',{style:'currency',currency:'IDR',maximumFractionDigits:0}).format(value);
  const itemsEl = document.getElementById('cart-items');
  const amountEl = document.getElementById('amount-paid');
  let total = 0;
  function drawCart(){
    if (!itemsEl) return;
    const entries=[...cart.values()];
    if (!entries.length) itemsEl.innerHTML='<div class="empty-state"><span>▤</span><strong>Keranjang masih kosong</strong><small>Pilih obat dari daftar untuk memulai transaksi.</small></div>';
    else itemsEl.innerHTML=entries.map(item=>`<div class="cart-item"><div><strong>${escapeHTML(item.name)}</strong><small>FEFO: ${escapeHTML(item.batch)}</small><div class="qty-control"><button type="button" data-qty="-1" data-key="${escapeHTML(item.key)}">−</button><span>${item.qty}</span><button type="button" data-qty="1" data-key="${escapeHTML(item.key)}">＋</button></div></div><div class="cart-item-side"><strong>${rupiah(item.price*item.qty)}</strong><button class="link-button" type="button" data-delete="${escapeHTML(item.key)}">Hapus</button></div></div>`).join('');
    total=entries.reduce((sum,item)=>sum+item.price*item.qty,0);
    document.getElementById('cart-count').textContent=`${entries.reduce((sum,item)=>sum+item.qty,0)} item`;
    document.getElementById('subtotal').textContent=rupiah(total); document.getElementById('grand-total').textContent=rupiah(total); updatePayment();
  }
  function escapeHTML(s){return String(s).replace(/[&<>"']/g,c=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c]));}
  function updatePayment(){const paid=Number(amountEl?.value||0);const diff=paid-total;const msg=document.getElementById('payment-message');const val=document.getElementById('change-value');if(!msg||!val)return;if(!total){msg.textContent='Masukkan pembayaran untuk menghitung kembalian.';val.textContent=rupiah(0);}else if(diff<0){msg.textContent=`Kurang ${rupiah(-diff)} dari total tagihan.`;val.textContent='Belum cukup';}else{msg.textContent='Pembayaran mencukupi.';val.textContent=rupiah(diff);}}
  document.addEventListener('click',event=>{const add=event.target.closest('[data-add-cart]');if(add){const key=add.dataset.name;const item=cart.get(key)||{key,name:add.dataset.name,price:Number(add.dataset.price),batch:add.dataset.batch,qty:0};item.qty++;cart.set(key,item);drawCart();window.apotekToast?.(`${item.name} ditambahkan ke keranjang.`);}const qty=event.target.closest('[data-qty]');if(qty){const item=cart.get(qty.dataset.key);if(item){item.qty+=Number(qty.dataset.qty);if(item.qty<=0)cart.delete(qty.dataset.key);else cart.set(qty.dataset.key,item);drawCart();}}const del=event.target.closest('[data-delete]');if(del){cart.delete(del.dataset.delete);drawCart();}});
  amountEl?.addEventListener('input',updatePayment);
  document.getElementById('checkout-button')?.addEventListener('click',()=>{if(!cart.size){window.apotekToast?.('Keranjang masih kosong. Tambahkan obat terlebih dahulu.');return;}if(Number(amountEl.value||0)<total){window.apotekToast?.('Jumlah pembayaran belum mencukupi.');return;}window.apotekToast?.('Transaksi demo selesai. Membuka pratinjau struk…');window.setTimeout(()=>{window.location.href=document.body.dataset.receiptUrl||'/__preview/struk_penjualan';},500);});
  drawCart();
})();
