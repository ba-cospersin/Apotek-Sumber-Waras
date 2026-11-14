(() => {
  const table = document.getElementById('receiving-table');
  const rowCount = document.getElementById('receiving-row-count');
  const totalDisplay = document.getElementById('receiving-total-display');
  const fmt = number => `Rp${Math.round(number).toLocaleString('id-ID')}`;
  const conversion = unit => ({Box:100,Strip:10,Tablet:1,Botol:30,Kapsul:1}[unit] || 1);
  function recalculate() {
    if (!table) return;
    let total = 0;
    const rows = [...table.querySelectorAll('tbody tr')];
    rows.forEach(row => {
      const qty = Number(row.querySelector('.receiving-qty')?.value || 0);
      const price = Number(row.querySelector('.receiving-price')?.value || 0);
      const unit = row.querySelector('td:nth-child(5) select')?.value || 'Tablet';
      const multiplier = conversion(unit);
      const baseQty = qty * multiplier;
      const basePrice = multiplier ? price / multiplier : price;
      const totalNode = row.querySelector('.receiving-base-total');
      const priceNode = row.querySelector('.receiving-base-price');
      const baseUnit = unit === 'Strip' || unit === 'Box' ? 'Tablet' : (unit === 'Botol' ? 'Tablet' : unit);
      if (totalNode) totalNode.textContent = `${baseQty.toLocaleString('id-ID')} ${baseUnit}`;
      if (priceNode) priceNode.textContent = fmt(basePrice);
      total += qty * price;
    });
    if (rowCount) rowCount.textContent = `${rows.length} Item`;
    if (totalDisplay) totalDisplay.textContent = fmt(total);
  }
  document.addEventListener('input', event => {
    if (event.target.closest('#receiving-table')) recalculate();
  });
  document.addEventListener('change', event => {
    if (event.target.closest('#receiving-table')) recalculate();
  });
  document.addEventListener('apotek:rows-changed', recalculate);
  recalculate();
})();
