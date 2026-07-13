import { createSignal, onMount, onCleanup } from 'solid-js';

const MIN_SMALL_BREAKPOINT = '(min-width: 1024px)';

/**
 * Hook to detect if viewport is mobile or desktop
 * @returns accessor function that returns true if mobile (below min-small breakpoint), false if desktop
 */
export function useIsMobile() {
    const [isMobile, setIsMobile] = createSignal(true);

    onMount(() => {
        const mediaQuery = window.matchMedia(MIN_SMALL_BREAKPOINT);

        const handleMediaChange = (e: MediaQueryListEvent | MediaQueryList) => {
            setIsMobile(!e.matches);
        };

        setIsMobile(!mediaQuery.matches);

        mediaQuery.addEventListener('change', handleMediaChange);

        onCleanup(() => {
            mediaQuery.removeEventListener('change', handleMediaChange);
        });
    });

    return isMobile;
}
