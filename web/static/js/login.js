(() => {
    const passwordInput = document.querySelector("#password");
    const toggleButton = document.querySelector("[data-password-toggle]");

    if (!passwordInput || !toggleButton) {
        return;
    }

    toggleButton.addEventListener("click", () => {
        const willShowPassword = passwordInput.type === "password";
        passwordInput.type = willShowPassword ? "text" : "password";
        toggleButton.setAttribute("aria-pressed", String(willShowPassword));
        toggleButton.setAttribute(
            "aria-label",
            willShowPassword ? "Sembunyikan password" : "Tampilkan password"
        );

        toggleButton.innerHTML = willShowPassword
            ? '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M3 3l18 18M10.6 10.6a2 2 0 002.8 2.8M9.9 5.2A10.8 10.8 0 0112 5c6.1 0 9.5 7 9.5 7a16 16 0 01-3.1 3.8M6.2 6.2C3.8 7.9 2.5 12 2.5 12s3.4 7 9.5 7c1.2 0 2.3-.2 3.2-.6"/></svg>'
            : '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="M2.5 12s3.4-6 9.5-6 9.5 6 9.5 6-3.4 6-9.5 6-9.5-6-9.5-6Z"/><circle cx="12" cy="12" r="2.7"/></svg>';
    });
})();
