import { createSignal, onMount, onCleanup } from 'solid-js';

const MOBILE_BREAKPOINT = 768;

export const useMediaQuery = (query: string): (() => boolean) => {
    const [matches, setMatches] = createSignal(window.matchMedia(query).matches);

    onMount(() => {
        const mediaQuery = window.matchMedia(query);
        const handler = (event: MediaQueryListEvent) => setMatches(event.matches);

        mediaQuery.addEventListener('change', handler);
        setMatches(mediaQuery.matches);

        onCleanup(() => mediaQuery.removeEventListener('change', handler));
    });

    return matches;
};

export const useIsDesktop = () => useMediaQuery(`(min-width: ${MOBILE_BREAKPOINT}px)`);
