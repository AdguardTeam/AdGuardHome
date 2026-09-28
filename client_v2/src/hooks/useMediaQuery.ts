import { type Accessor, createSignal, onCleanup, onMount } from 'solid-js';

import { DESKTOP_MIN_WIDTH, MOBILE_MAX_WIDTH } from 'panel/helpers/breakpoints';

/**
 * Reactive wrapper around `window.matchMedia`.  The returned accessor tracks
 * whether `query` currently matches and re-reads on every change.
 */
export function useMediaQuery(query: string): Accessor<boolean> {
    const [matches, setMatches] = createSignal(window.matchMedia(query).matches);

    onMount(() => {
        const mediaQuery = window.matchMedia(query);
        const handler = (event: MediaQueryListEvent) => setMatches(event.matches);

        mediaQuery.addEventListener('change', handler);

        // Re-read after subscribing: the viewport may have changed between the
        // initial read above and the listener registration.
        setMatches(mediaQuery.matches);

        onCleanup(() => mediaQuery.removeEventListener('change', handler));
    });

    return matches;
}

/**
 * Reports `true` when the viewport is at least {@link DESKTOP_MIN_WIDTH} wide.
 */
export function useIsDesktop(): Accessor<boolean> {
    return useMediaQuery(`(min-width: ${DESKTOP_MIN_WIDTH}px)`);
}

/**
 * Reports `true` when the viewport is narrower than {@link MOBILE_MAX_WIDTH}.
 *
 * Note that this is a wider range than {@link useIsDesktop} covers, so
 * viewports between 768px and 1023px report `true` for both hooks.  The
 * behaviour is pinned by `__tests__/hooks/use-media-query.test.tsx`.
 */
export function useIsMobile(): Accessor<boolean> {
    const isWideEnough = useMediaQuery(`(min-width: ${MOBILE_MAX_WIDTH}px)`);

    return () => !isWideEnough();
}

/**
 * Reports `true` on devices that cannot hover (touch screens), where tooltips
 * have to be tap-driven instead of hover-driven.
 */
export function useIsTouchDevice(): Accessor<boolean> {
    return useMediaQuery('(hover: none)');
}
