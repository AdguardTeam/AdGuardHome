import { createSignal } from 'solid-js';
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
let observerCount = 0;

/** Replaces the global no-op `ResizeObserver` from `src/__tests__/setup.ts`. */
const installResizeObserver = () => {
    observerCallback = undefined;
    observerDisconnected = false;
    observerCount = 0;

    class FakeResizeObserver {
        constructor(callback: ResizeObserverCallback) {
            observerCallback = callback;
            observerCount += 1;
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

/** The same probe, with the gating options driven by signals from the test. */
const GatedHarness = (props: { enabled: boolean; text: string }) => {
    let labelRef: HTMLSpanElement | undefined;

    const isTruncated = useIsTruncated(() => labelRef, {
        enabled: () => props.enabled,
        source: () => props.text,
    });

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

    it('neither measures nor observes while disabled', () => {
        stubOverflow(200, 100);
        installResizeObserver();

        render(() => <GatedHarness enabled={false} text="a" />);

        expect(observerCount).toBe(0);
        expect(screen.getByTestId('label')).toHaveTextContent('fits');
    });

    it('measures and observes once enabled', () => {
        stubOverflow(200, 100);
        installResizeObserver();

        const [enabled, setEnabled] = createSignal(false);
        render(() => <GatedHarness enabled={enabled()} text="a" />);
        expect(screen.getByTestId('label')).toHaveTextContent('fits');

        setEnabled(true);

        expect(observerCount).toBe(1);
        expect(screen.getByTestId('label')).toHaveTextContent('truncated');
    });

    it('stops observing and resets once disabled again', () => {
        stubOverflow(200, 100);
        installResizeObserver();

        const [enabled, setEnabled] = createSignal(true);
        render(() => <GatedHarness enabled={enabled()} text="a" />);
        expect(observerDisconnected).toBe(false);

        setEnabled(false);

        expect(observerDisconnected).toBe(true);
        expect(screen.getByTestId('label')).toHaveTextContent('fits');
    });

    it('re-measures when the source changes without a box resize', () => {
        stubOverflow(100, 100);
        installResizeObserver();

        const [text, setText] = createSignal('a');
        render(() => <GatedHarness enabled text={text()} />);
        expect(screen.getByTestId('label')).toHaveTextContent('fits');

        stubOverflow(200, 100);
        setText('a much longer name');

        expect(screen.getByTestId('label')).toHaveTextContent('truncated');
        // The observer follows the element, not the text, so it must survive.
        expect(observerCount).toBe(1);
    });
});
