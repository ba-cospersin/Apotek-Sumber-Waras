// Halaman yang dilindungi login tidak boleh ditampilkan lagi dari bfcache
// setelah logout. Saat browser memulihkan halaman dari bfcache, muat ulang
// supaya server memeriksa session terbaru.
window.addEventListener("pageshow", (event) => {
    if (event.persisted) {
        window.location.reload();
    }
});
