import '@testing-library/jest-dom/vitest';
import { afterEach } from 'vitest';
import { cleanup } from '@solidjs/testing-library';

afterEach(() => cleanup());

// The theme lives on `<html>` and would otherwise leak into the next test.
afterEach(() => document.documentElement.removeAttribute('data-theme'));

// Mock window.scrollTo for router navigation (jsdom doesn't implement it).
Object.defineProperty(window, 'scrollTo', {
    writable: true,
    value: () => {},
});

// Mock window.matchMedia for components that use the responsive hooks from
// `panel/hooks/useMediaQuery`.
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

// jsdom lacks ResizeObserver; floating-ui (@zag-js/popper) needs it when
// popovers/select menus open.
class ResizeObserverMock {
    observe() {}
    unobserve() {}
    disconnect() {}
}

Object.defineProperty(window, 'ResizeObserver', {
    writable: true,
    value: ResizeObserverMock,
});

// jsdom lacks Element.scrollTo; @zag-js/select calls contentEl.scrollTo when
// a menu item is selected.
if (!Element.prototype.scrollTo) {
    Object.defineProperty(Element.prototype, 'scrollTo', {
        writable: true,
        value: () => {},
    });
}
