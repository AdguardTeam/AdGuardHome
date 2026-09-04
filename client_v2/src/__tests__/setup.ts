import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@solidjs/testing-library';

afterEach(() => cleanup());

// Mock window.scrollTo for router navigation (jsdom doesn't implement it).
Object.defineProperty(window, 'scrollTo', {
    writable: true,
    value: () => {},
});

// Mock window.matchMedia for components that use useIsMobile
Object.defineProperty(window, 'matchMedia', {
    writable: true,
    value: (query: string): MediaQueryList =>
        ({
            matches: false,
            media: query,
            onchange: null,
            addListener: () => {},
            removeListener: () => {},
            addEventListener: () => {},
            removeEventListener: () => {},
            dispatchEvent: () => false,
        }) as MediaQueryList,
});
