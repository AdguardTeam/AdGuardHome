import { useState, useEffect } from 'react';

const MOBILE_BREAKPOINT = 768;

export const useMediaQuery = (query: string): boolean => {
    const [matches, setMatches] = useState(() => window.matchMedia(query).matches);

    useEffect(() => {
        const mediaQuery = window.matchMedia(query);
        const handler = (event: MediaQueryListEvent) => setMatches(event.matches);

        mediaQuery.addEventListener('change', handler);
        setMatches(mediaQuery.matches);

        return () => mediaQuery.removeEventListener('change', handler);
    }, [query]);

    return matches;
};

export const useIsDesktop = () => useMediaQuery(`(min-width: ${MOBILE_BREAKPOINT}px)`);
