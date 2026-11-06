/**
 * Resolves a CSS custom property to the computed color value of the current
 * theme. Chart.js paints into a canvas, where CSS variables cannot be used
 * directly, so the value has to be read from a styled probe element.
 *
 * @param token - custom property name, including the leading `--`
 * @param fallback - color used when the property is not defined or cannot be
 *     resolved (e.g. in tests or non-browser environments)
 */
export const resolveCssColor = (token: string, fallback: string): string => {
    if (typeof document === 'undefined' || !document.body) return fallback;

    const probe = document.createElement('div');
    probe.style.display = 'none';
    // The fallback keeps the declaration valid when the token is missing.
    probe.style.color = `var(${token}, ${fallback})`;
    document.body.appendChild(probe);

    const color = getComputedStyle(probe).color;
    probe.remove();

    // jsdom does not resolve var() in computed styles; use the fallback there.
    return color && !color.startsWith('var(') ? color : fallback;
};

const HEX_COLOR_RE = /^#([0-9a-f]{3}|[0-9a-f]{6})$/i;
const RGB_COLOR_RE = /^rgba?\(\s*(\d{1,3})[,\s]+(\d{1,3})[,\s]+(\d{1,3})/i;

/**
 * Returns `color` with the given alpha channel. Accepts hex (`#rgb`, `#rrggbb`)
 * and `rgb()`/`rgba()` values; unsupported formats are returned unchanged.
 */
export const withAlpha = (color: string, alpha: number): string => {
    const rgb = color.match(RGB_COLOR_RE);
    if (rgb) {
        return `rgba(${rgb[1]}, ${rgb[2]}, ${rgb[3]}, ${alpha})`;
    }

    if (!HEX_COLOR_RE.test(color)) return color;

    let hex = color.slice(1);
    if (hex.length === 3) {
        hex = hex
            .split('')
            .map((c) => c + c)
            .join('');
    }

    const r = parseInt(hex.slice(0, 2), 16);
    const g = parseInt(hex.slice(2, 4), 16);
    const b = parseInt(hex.slice(4, 6), 16);

    return `rgba(${r}, ${g}, ${b}, ${alpha})`;
};
