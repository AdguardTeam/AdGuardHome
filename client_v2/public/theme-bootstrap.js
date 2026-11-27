// Applies the stored theme before the first paint.  This is the only copy:
// `webpack.common.js` inlines it into every HTML entry point.
(function () {
    var THEME_KEY = 'account_theme';
    var theme = null;
    try {
        var raw = window.localStorage.getItem(THEME_KEY);
        if (raw) {
            try {
                theme = JSON.parse(raw);
            } catch (e) {
                theme = raw;
            }
        }
    } catch (e) {}
    if (!theme || theme === 'auto') {
        var prefersDark =
            window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
        theme = prefersDark ? 'dark' : 'light';
    }
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;
})();
