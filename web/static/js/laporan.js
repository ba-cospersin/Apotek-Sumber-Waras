(() => {
  const tabs = [...document.querySelectorAll('[data-report-tab]')];
  const panels = [...document.querySelectorAll('[data-report-panel]')];
  tabs.forEach(tab => tab.addEventListener('click', () => {
    tabs.forEach(item => { const active=item===tab; item.classList.toggle('is-active',active); item.setAttribute('aria-selected',String(active)); });
    panels.forEach(panel => { const active=panel.dataset.reportPanel===tab.dataset.reportTab; panel.hidden=!active; panel.classList.toggle('is-visible',active); });
  }));
  document.addEventListener('click', event => {
    const button=event.target.closest('[data-report-layer]'); if(!button)return;
    const row=button.closest('tr'); const detail=row?.nextElementSibling; if(!detail?.classList.contains('report-layer-row'))return;
    detail.hidden=!detail.hidden; button.textContent=detail.hidden?'⌄':'⌃'; button.classList.toggle('is-open',!detail.hidden);
  });
})();