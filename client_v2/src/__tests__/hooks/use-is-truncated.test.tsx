import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup } from '@solidjs/testing-library';

import { useIsTruncated } from 'panel/hooks/useIsTruncated';

/**
 * jsdom reports 0 for both `scrollWidth` and `clientWidth`, so the overflow
 * state has to be injected.  The properties live on `Element.prototype`, so
 * shadowing them on `HTMLElement.prototype` and deleting the shadow restores
 * the originals.
 */
const stubOverflow = (scrollWidth: number, clientWidth: number) => {
    Object.defineProperty(HTMLElement.prototype, 'scrollWidth', {
        configurable: true,
        get: () => scrollWidth,
    });
    Object.defineProperty(HTMLElement.prototype, 'clientWidth', {
        configurable: true,
        get: () => clientWidth,
    });
};

const restoreOverflow = () => {
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollWidth');
    Reflect.deleteProperty(HTMLElement.prototype, 'clientWidth');
};

let observerCallback: ResizeObserverCallback | undefined;
let observerDisconnected = false;

/** Replaces the global no-op `ResizeObserver` from `src/__tests__/setup.ts`. */
const installResizeObserver = () => {
    observerCallback = undefined;
    observerDisconnected = false;

    class FakeResizeObserver {
        constructor(callback: ResizeObserverCallback) {
            observerCallback = callback;
        }

        observe() {}

        unobserve() {}

        disconnect() {
            observerDisconnected = true;
        }
    }

    window.ResizeObserver = FakeResizeObserver as unknown as typeof ResizeObserver;
};

const Harness = () => {
    let labelRef: HTMLSpanElement | undefined;

    const isTruncated = useIsTruncated(() => labelRef);

    return (
        <span ref={labelRef} data-testid="label">
            {isTruncated() ? 'truncated' : 'fits'}
        </span>
    );
};

describe('useIsTruncated', () => {
    afterEach(() => {
        restoreOverflow();
    });

    it('reports false when the content fits', () => {
        stubOverflow(100, 100);
        installResizeObserver();

        render(() => <Harness />);

        expect(screen.getByTestId('label')).toHaveTextContent('fits');
    });

    it('reports true when the content overflows', () => {
        stubOverflow(200, 100);
        installResizeObserver();

        render(() => <Harness />);

        expect(screen.getByTestId('label')).toHaveTextContent('truncated');
    });

    it('measures synchronously, without waiting for the observer', () => {
        stubOverflow(200, 100);
        installResizeObserver();

        render(() => <Harness />);

        // The observer was registered but never invoked, so the value can only
        // come from the measurement taken during setup.
        expect(observerCallback).toBeTypeOf('function');
        expect(screen.getByTestId('label')).toHaveTextContent('truncated');
    });

    it('re-measures when the observed element resizes', () => {
        stubOverflow(100, 100);
        installResizeObserver();

        render(() => <Harness />);
        expect(screen.getByTestId('label')).toHaveTextContent('fits');

        stubOverflow(200, 100);
        // Solid flushes the signal update synchronously outside a batch.
        observerCallback?.([] as ResizeObserverEntry[], {} as ResizeObserver);

        expect(screen.getByTestId('label')).toHaveTextContent('truncated');
    });

    it('disconnects the observer on cleanup', () => {
        stubOverflow(200, 100);
        installResizeObserver();

        render(() => <Harness />);
        expect(observerDisconnected).toBe(false);

        cleanup();

        expect(observerDisconnected).toBe(true);
    });
});
