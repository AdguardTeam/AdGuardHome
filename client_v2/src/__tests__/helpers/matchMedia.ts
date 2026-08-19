/**
 * Overrides `window.matchMedia` for a single test.
 *
 * The global setup mock in `setup.ts` always reports `matches: false`
 * (mobile).  Components that branch on `useIsDesktop()` need an explicit
 * override, e.g. `setMatchMedia(true)` to emulate a desktop viewport.
 */
export const setMatchMedia = (matches: boolean): void => {
    Object.defineProperty(window, 'matchMedia', {
        writable: true,
        value: (query: string): MediaQueryList =>
            ({
                matches,
                media: query,
                onchange: null,
                addListener: () => {},
                removeListener: () => {},
                addEventListener: () => {},
                removeEventListener: () => {},
                dispatchEvent: () => false,
            }) as MediaQueryList,
    });
};
