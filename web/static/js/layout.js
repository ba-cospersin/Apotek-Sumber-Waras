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

(() => {
    const menu = document.querySelector('[data-user-menu]');
    if (!menu) return;

    const toggle = menu.querySelector('#user-menu-toggle');
    const panel = menu.querySelector('#user-menu-panel');
    if (!toggle || !panel) return;

    const setOpen = (open) => {
        menu.classList.toggle('is-open', open);
        toggle.setAttribute('aria-expanded', String(open));
    };

    toggle.addEventListener('click', (event) => {
        event.stopPropagation();
        setOpen(!menu.classList.contains('is-open'));
    });

    document.addEventListener('click', (event) => {
        if (!menu.contains(event.target)) setOpen(false);
    });

    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape' && menu.classList.contains('is-open')) {
            setOpen(false);
            toggle.focus();
        }
    });

    menu.addEventListener('focusout', () => {
        window.setTimeout(() => {
            if (!menu.contains(document.activeElement)) setOpen(false);
        }, 0);
    });

    panel.addEventListener('click', (event) => {
        if (event.target.closest('a, button')) setOpen(false);
    });
})();
