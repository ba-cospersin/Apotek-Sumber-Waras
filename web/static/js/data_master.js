(() => {
  const tabs = [...document.querySelectorAll('[data-master-tab]')];
  const panels = [...document.querySelectorAll('[data-master-panel]')];
  tabs.forEach(tab => tab.addEventListener('click', () => {
    tabs.forEach(item => {
      const active = item === tab;
      item.classList.toggle('is-active', active);
      item.setAttribute('aria-selected', String(active));
    });
    panels.forEach(panel => {
      const active = panel.dataset.masterPanel === tab.dataset.masterTab;
      panel.hidden = !active;
      panel.classList.toggle('is-visible', active);
    });
  }));
})();