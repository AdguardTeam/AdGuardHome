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
