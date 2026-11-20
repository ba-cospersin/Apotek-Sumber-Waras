(() => {
  const physicalInput = document.getElementById('physical-qty');
  const systemInput = document.getElementById('system-qty');
  const differenceOutput = document.getElementById('stock-difference');
  const differenceBox = document.getElementById('difference-box');
  const differenceLabel = document.getElementById('difference-label');
  const summary = document.getElementById('allocation-summary');
  const batchSelect = document.getElementById('opname-batch');
  const layerAdjustment1 = document.getElementById('layer-adjustment-1');
  const layerAdjustment2 = document.getElementById('layer-adjustment-2');
  const layerAfter1 = document.getElementById('layer-after-1');
  const layerAfter2 = document.getElementById('layer-after-2');

  const batches = {
    paracetamol: { system: 150, unit: 'Strip', first: 50, second: 100, invoice1: 'FAK-KF-20261015-012', invoice2: 'FAK-KF-20261107-089' },
    amoxicillin: { system: 50, unit: 'Strip', first: 20, second: 30, invoice1: 'FAK-KF-20261015-012', invoice2: 'FAK-KF-20261107-089' },
    antasida: { system: 20, unit: 'Botol', first: 8, second: 12, invoice1: 'FAK-CB-20260920-038', invoice2: 'FAK-CB-20261005-014' }
  };
  const format = value => `${value > 0 ? '+' : value < 0 ? '−' : ''}${Math.abs(value)} ${current().unit}`;
  const current = () => batches[batchSelect?.value] || batches.paracetamol;

  function update() {
    if (!physicalInput || !systemInput || !differenceOutput) return;
    const data = current();
    const physical = Math.max(0, Number(physicalInput.value || 0));
    const difference = physical - data.system;
    systemInput.value = `${data.system} ${data.unit}`;
    differenceOutput.textContent = format(difference);
    differenceBox?.classList.toggle('difference-negative', difference < 0);
    differenceBox?.classList.toggle('difference-positive', difference > 0);
    differenceBox?.classList.toggle('difference-neutral', difference === 0);
    if (differenceLabel) differenceLabel.textContent = difference < 0 ? 'Kurang dari sistem:' : difference > 0 ? 'Lebih dari sistem:' : 'Sesuai sistem:';

    let adjustFirst = 0;
    let adjustSecond = 0;
    if (difference < 0) {
      const reduction = Math.abs(difference);
      adjustFirst = -Math.min(data.first, reduction);
      adjustSecond = -Math.max(0, reduction - data.first);
    } else if (difference > 0) {
      adjustSecond = difference;
    }
    if (layerAdjustment1) layerAdjustment1.textContent = `${adjustFirst > 0 ? '+' : adjustFirst < 0 ? '−' : ''}${Math.abs(adjustFirst)} ${data.unit}`;
    if (layerAdjustment2) layerAdjustment2.textContent = `${adjustSecond > 0 ? '+' : adjustSecond < 0 ? '−' : ''}${Math.abs(adjustSecond)} ${data.unit}`;
    if (layerAfter1) layerAfter1.textContent = `${data.first + adjustFirst} ${data.unit}`;
    if (layerAfter2) layerAfter2.textContent = `${data.second + adjustSecond} ${data.unit}`;
    if (summary) {
      const action = difference < 0 ? 'pengurangan' : difference > 0 ? 'penambahan' : 'tanpa penyesuaian';
      const selectedLabel = batchSelect?.selectedOptions?.[0]?.textContent || 'batch terpilih';
      const selectedBatch = selectedLabel.match(/-\s*([^ (]+)\s*\(/)?.[1] || 'batch terpilih';
      summary.innerHTML = `<i></i> Simulasi alokasi ${action} lapisan FEFO untuk Batch ${selectedBatch} (Selisih: <strong>${format(difference)}</strong>)`;
    }
  }

  physicalInput?.addEventListener('input', update);
  batchSelect?.addEventListener('change', () => {
    physicalInput.value = String(current().system);
    update();
  });
  update();
})();
