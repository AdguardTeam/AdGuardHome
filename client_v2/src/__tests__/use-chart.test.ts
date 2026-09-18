import { describe, it, expect, beforeEach, afterEach } from 'vitest';

import { createExternalTooltipHandler } from 'panel/helpers/useChart';

/** Viewport size the stubs below report. */
const VIEWPORT_WIDTH = 1024;
const VIEWPORT_HEIGHT = 768;

const stubProperty = (target: object, property: string, value: number) => {
    Object.defineProperty(target, property, { configurable: true, value });
};

const resetProperty = (target: object, property: string) => {
    delete (target as Record<string, unknown>)[property];
};

/** Stubs an element rect for the handler's position calculations. */
const stubRect = (target: object, { left, top }: { left: number; top: number }) => {
    Object.defineProperty(target, 'getBoundingClientRect', {
        configurable: true,
        value: () => ({
            left,
            top,
            right: left,
            bottom: top,
            width: 0,
            height: 0,
            x: left,
            y: top,
        }),
    });
};

/**
 * Builds a fixture that mimics the StatCard markup: a canvas inside a
 * wrapper with the absolutely-positioned tooltip as its sibling.
 *
 * jsdom has no layout: getBoundingClientRect / offsetWidth report zeroes
 * and the document element's clientWidth / clientHeight have to be stubbed,
 * so all positions are computed relative to the canvas and wrapper at 0,0.
 */
const setup = () => {
    const wrapper = document.createElement('div');
    const tooltip = document.createElement('div');
    const canvas = document.createElement('canvas');
    wrapper.appendChild(canvas);
    wrapper.appendChild(tooltip);
    document.body.appendChild(wrapper);

    const chart = { canvas };
    return { chart, tooltip, wrapper };
};

const createTooltipState = (overrides: Record<string, unknown> = {}) => ({
    opacity: 1,
    dataPoints: [{ raw: 12345, label: '24 Aug 01:00' }],
    caretX: 100,
    caretY: 50,
    ...overrides,
});

describe('createExternalTooltipHandler', () => {
    beforeEach(() => {
        stubProperty(document.documentElement, 'clientWidth', VIEWPORT_WIDTH);
        stubProperty(document.documentElement, 'clientHeight', VIEWPORT_HEIGHT);
    });

    afterEach(() => {
        resetProperty(document.documentElement, 'clientWidth');
        resetProperty(document.documentElement, 'clientHeight');
    });

    it('hides the tooltip when tooltip opacity is 0', () => {
        const { chart, tooltip } = setup();
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );
        handler({ chart, tooltip: { opacity: 0 } } as never);

        expect(tooltip.style.opacity).toBe('0');
    });

    it('shows the tooltip with rendered content and position', () => {
        const { chart, tooltip } = setup();
        const handler = createExternalTooltipHandler(
            () => tooltip,
            (dataPoint) => `<div>${dataPoint.raw}</div><div>${dataPoint.label}</div>`,
        );
        handler({ chart, tooltip: createTooltipState() } as never);

        expect(tooltip.style.opacity).toBe('1');
        expect(tooltip.innerHTML).toContain('12345');
        expect(tooltip.innerHTML).toContain('24 Aug 01:00');
        // caretX + 12 relative to the wrapper (at 0,0)
        expect(tooltip.style.left).toBe('112px');
        expect(tooltip.style.top).toBe('50px');
    });

    it('flips the tooltip to the left near the right viewport edge', () => {
        const { chart, tooltip } = setup();
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );
        handler({
            chart,
            tooltip: createTooltipState({ caretX: 1010 }),
        } as never);

        // The tooltip is 0 wide in jsdom and the viewport is 1024 wide, so
        // there is no room on the right and it flips left: caretX - 12.
        expect(tooltip.style.left).toBe('998px');
    });

    it('pins the tooltip to the right viewport edge when flipping is not enough', () => {
        const { chart, tooltip } = setup();
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );
        handler({
            chart,
            tooltip: createTooltipState({ caretX: 5000 }),
        } as never);

        // Even the flipped position is off-screen, so the tooltip is clamped
        // to the visible area: viewport width - width (0) - margin (8).
        expect(tooltip.style.left).toBe('1016px');
    });

    it('pins a too wide tooltip to the left viewport edge', () => {
        const { chart, tooltip } = setup();
        stubProperty(tooltip, 'offsetWidth', 1000);
        stubProperty(tooltip, 'offsetHeight', 40);
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );
        handler({
            chart,
            tooltip: createTooltipState({ caretX: 500 }),
        } as never);

        // Neither side of the cursor has room for a 1000px tooltip, so it is
        // pinned to the left margin instead of sticking out of the viewport.
        expect(tooltip.style.left).toBe('8px');
        expect(tooltip.style.top).toBe('30px');
    });

    it('keeps the tooltip within the viewport vertically', () => {
        const { chart, tooltip } = setup();
        stubProperty(tooltip, 'offsetHeight', 40);
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );

        handler({ chart, tooltip: createTooltipState({ caretY: -100 }) } as never);
        expect(tooltip.style.top).toBe('8px');

        handler({ chart, tooltip: createTooltipState({ caretY: 5000 }) } as never);
        // viewport height (768) - height (40) - margin (8)
        expect(tooltip.style.top).toBe('720px');
    });

    it('positions the tooltip relative to its offset wrapper', () => {
        const { chart, tooltip, wrapper } = setup();
        stubProperty(tooltip, 'offsetWidth', 90);
        stubRect(chart.canvas, { left: 800, top: 40 });
        stubRect(wrapper, { left: 780, top: 20 });
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );
        handler({
            chart,
            tooltip: createTooltipState({ caretX: 180, caretY: 30 }),
        } as never);

        // The caret sits at 980px: 980 + 90 + 12 does not fit in the
        // viewport, so the tooltip flips to 980 - 90 - 12 = 878px and is
        // stored relative to the wrapper: 878 - 780.
        expect(tooltip.style.left).toBe('98px');
        // 40 + 30 - 0/2 = 70 relative to the viewport, minus the wrapper top.
        expect(tooltip.style.top).toBe('50px');
    });

    it('does not show the tooltip without data points', () => {
        const { chart, tooltip } = setup();
        const handler = createExternalTooltipHandler(
            () => tooltip,
            () => '',
        );
        handler({ chart, tooltip: { opacity: 1, dataPoints: [] } } as never);

        expect(tooltip.style.opacity).not.toBe('1');
    });
});
