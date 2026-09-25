import { describe, it, expect, afterEach } from 'vitest';
import { render, screen, cleanup, fireEvent } from '@solidjs/testing-library';

import { useScrollEdges } from 'panel/hooks/useScrollEdges';

/**
 * jsdom reports 0 for every scroll metric, so the scroll range has to be
 * injected.  The properties live on `Element.prototype`, so shadowing them on
 * `HTMLElement.prototype` and deleting the shadow restores the originals.
 */
type Metrics = { scrollTop: number; scrollHeight: number; clientHeight: number };

let metrics: Metrics = { scrollTop: 0, scrollHeight: 0, clientHeight: 0 };

const installScrollMetrics = (next: Partial<Metrics> = {}) => {
    metrics = { scrollTop: 0, scrollHeight: 0, clientHeight: 0, ...next };

    Object.defineProperty(HTMLElement.prototype, 'scrollTop', {
        configurable: true,
        get: () => metrics.scrollTop,
        set: (value: number) => {
            metrics.scrollTop = value;
        },
    });
    Object.defineProperty(HTMLElement.prototype, 'scrollHeight', {
        configurable: true,
        get: () => metrics.scrollHeight,
    });
    Object.defineProperty(HTMLElement.prototype, 'clientHeight', {
        configurable: true,
        get: () => metrics.clientHeight,
    });
};

const restoreScrollMetrics = () => {
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollTop');
    Reflect.deleteProperty(HTMLElement.prototype, 'scrollHeight');
    Reflect.deleteProperty(HTMLElement.prototype, 'clientHeight');
};

let observerCallback: ResizeObserverCallback | undefined;
let observedTargets: Element[] = [];
let disconnectCount = 0;

/** Replaces the global no-op `ResizeObserver` from `src/__tests__/setup.ts`. */
const installResizeObserver = () => {
    observerCallback = undefined;
    observedTargets = [];
    disconnectCount = 0;

    class FakeResizeObserver {
        constructor(callback: ResizeObserverCallback) {
            observerCallback = callback;
        }

        observe(target: Element) {
            observedTargets.push(target);
        }

        unobserve() {}

        disconnect() {
            disconnectCount += 1;
        }
    }

    window.ResizeObserver = FakeResizeObserver as unknown as typeof ResizeObserver;
};

const Harness = () => {
    let scrollRef: HTMLDivElement | undefined;

    const { canScrollUp, canScrollDown } = useScrollEdges(() => scrollRef);

    return (
        <div
            ref={scrollRef}
            data-testid="scroll-area"
            data-can-scroll-up={String(canScrollUp())}
            data-can-scroll-down={String(canScrollDown())}
        >
            <div data-testid="content" />
        </div>
    );
};

const WithoutElement = () => {
    const { canScrollUp, canScrollDown } = useScrollEdges(() => undefined);

    return (
        <div
            data-testid="scroll-area"
            data-can-scroll-up={String(canScrollUp())}
            data-can-scroll-down={String(canScrollDown())}
        />
    );
};

const scrollArea = () => screen.getByTestId('scroll-area');

describe('useScrollEdges', () => {
    afterEach(() => {
        restoreScrollMetrics();
    });

    it('reports no scrollable edges while the element is missing', () => {
        installScrollMetrics({ scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <WithoutElement />);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'false');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'false');
    });

    it('reports no scrollable edges when the content fits', () => {
        installScrollMetrics({ scrollHeight: 400, clientHeight: 400 });
        installResizeObserver();

        render(() => <Harness />);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'false');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'false');
    });

    it('reports only the bottom edge at the very top', () => {
        installScrollMetrics({ scrollTop: 0, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'false');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'true');
    });

    it('reports both edges in the middle', () => {
        installScrollMetrics({ scrollTop: 150, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'true');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'true');
    });

    it('reports only the top edge at the very bottom', () => {
        installScrollMetrics({ scrollTop: 300, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'true');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'false');
    });

    it('ignores a sub-pixel remainder at the edges', () => {
        installScrollMetrics({ scrollTop: 0.5, scrollHeight: 400, clientHeight: 399.5 });
        installResizeObserver();

        render(() => <Harness />);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'false');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'false');
    });

    it('re-measures on scroll', () => {
        installScrollMetrics({ scrollTop: 0, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);
        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'false');

        metrics.scrollTop = 300;
        fireEvent.scroll(scrollArea());

        expect(scrollArea()).toHaveAttribute('data-can-scroll-up', 'true');
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'false');
    });

    it('re-measures when the observed element resizes', () => {
        installScrollMetrics({ scrollTop: 0, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);
        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'true');

        // The content grew taller while the scroll position stayed put.
        metrics.scrollHeight = 100;
        observerCallback?.([] as ResizeObserverEntry[], {} as ResizeObserver);

        expect(scrollArea()).toHaveAttribute('data-can-scroll-down', 'false');
    });

    it('observes the container together with its content', () => {
        installScrollMetrics({ scrollTop: 0, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);

        expect(observedTargets).toContain(scrollArea());
        expect(observedTargets).toContain(screen.getByTestId('content'));
    });

    it('disconnects the observer on cleanup', () => {
        installScrollMetrics({ scrollTop: 0, scrollHeight: 400, clientHeight: 100 });
        installResizeObserver();

        render(() => <Harness />);
        const before = disconnectCount;

        cleanup();

        expect(disconnectCount).toBeGreaterThan(before);
    });
});
