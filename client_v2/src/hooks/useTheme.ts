import { type Accessor, createSignal, onCleanup, onMount } from 'solid-js';

import { THEMES } from 'panel/helpers/constants';

/**
 * Reports whether the dark theme is currently applied.
 *
 * The theme is driven by the `data-theme` attribute on `<html>` rather than by
 * a store, so the hook observes the attribute directly.  That is what makes it
 * usable from the standalone entries (`login`, `install`, `forgot_password`),
 * which never mount the store-driven theme effect in `components/App`.
 */
export function useIsDarkTheme(): Accessor<boolean> {
    const read = () => document.documentElement.dataset.theme === THEMES.dark;

    const [isDark, setIsDark] = createSignal(read());

    onMount(() => {
        const observer = new MutationObserver(() => setIsDark(read()));

        observer.observe(document.documentElement, {
            attributes: true,
            attributeFilter: ['data-theme'],
        });

        // Re-read after subscribing: the theme may have changed between the
        // initial read above and the observer registration.
        setIsDark(read());

        onCleanup(() => observer.disconnect());
    });

    return isDark;
}
