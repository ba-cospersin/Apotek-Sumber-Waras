(() => {
    const sidebar = document.getElementById('app-sidebar');
    const toggle = document.querySelector('[data-sidebar-toggle]');

    if (!sidebar || !toggle) return;

    toggle.addEventListener('click', () => {
        const isOpen = sidebar.classList.toggle('is-open');
        toggle.setAttribute('aria-expanded', String(isOpen));
    });

    document.addEventListener('click', (event) => {
        if (window.innerWidth > 760 || !sidebar.classList.contains('is-open')) return;
        if (sidebar.contains(event.target) || toggle.contains(event.target)) return;
        sidebar.classList.remove('is-open');
        toggle.setAttribute('aria-expanded', 'false');
    });
})();
