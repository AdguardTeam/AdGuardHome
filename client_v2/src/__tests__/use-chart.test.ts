import { describe, it, expect } from 'vitest';

import { createExternalTooltipHandler } from 'panel/helpers/useChart';

/**
 * Builds a fixture that mimics the StatCard markup: a canvas inside a
 * wrapper with the absolutely-positioned tooltip as its sibling.
 *
 * jsdom returns zeroes for getBoundingClientRect / offsetWidth, so all
 * positions are computed relative to them (canvas & wrapper at 0,0).
 */
const setup = () => {
    const wrapper = document.createElement('div');
    const tooltip = document.createElement('div');
    const canvas = document.createElement('canvas');
    wrapper.appendChild(canvas);
    wrapper.appendChild(tooltip);
    document.body.appendChild(wrapper);

    const chart = { canvas };
    return { chart, tooltip };
};

const createTooltipState = (overrides: Record<string, unknown> = {}) => ({
    opacity: 1,
    dataPoints: [{ raw: 12345, label: '24 Aug 01:00' }],
    caretX: 100,
    caretY: 50,
    ...overrides,
});

describe('createExternalTooltipHandler', () => {
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
            tooltip: createTooltipState({ caretX: 2000 }),
        } as never);

        // window.innerWidth is 1024 in jsdom, so the tooltip flips left:
        // caretX - width - 12, all relative to the wrapper.
        expect(tooltip.style.left).toBe('1988px');
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
