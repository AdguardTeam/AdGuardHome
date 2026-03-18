import { useEffect, useState } from 'react';

const MIN_SMALL_BREAKPOINT = '(min-width: 768px)';

/**
 * Hook to detect if viewport is mobile or desktop
 * @returns true if mobile (below min-small breakpoint), false if desktop
 */
export function useIsMobile() {
    const [isMobile, setIsMobile] = useState(true);

    useEffect(() => {
        const mediaQuery = window.matchMedia(MIN_SMALL_BREAKPOINT);

        const handleMediaChange = (e: MediaQueryListEvent | MediaQueryList) => {
            setIsMobile(!e.matches);
        };

        setIsMobile(!mediaQuery.matches);

        mediaQuery.addEventListener('change', handleMediaChange);

        return () => {
            mediaQuery.removeEventListener('change', handleMediaChange);
        };
    }, []);

    return isMobile;
}
