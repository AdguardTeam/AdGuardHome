import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';

import { StatCard, CARDS_THEME } from 'panel/components/Dashboard/blocks/StatCard';
import { RoutePath } from 'panel/components/Routes/Paths';

/**
 * Collects the Chart instances created by useChart so tests can drive the
 * tooltip through the captured chart options (same way Chart.js does).
 */
const { chartInstances } = vi.hoisted(() => ({
    chartInstances: [] as Array<{
        options: {
            plugins?: { tooltip?: { external?: (ctx: unknown) => void } };
        };
        setActiveElements: ReturnType<typeof vi.fn>;
        tooltip: {
            opacity: number;
            getActiveElements: ReturnType<typeof vi.fn>;
            setActiveElements: ReturnType<typeof vi.fn>;
        };
        update: ReturnType<typeof vi.fn>;
    }>,
}));

vi.mock('chart.js', () => {
    class Chart {
        static register = vi.fn();

        canvas: unknown;
        data: unknown;
        options: unknown;
        plugins: unknown;
        setActiveElements = vi.fn();
        tooltip: {
            opacity: number;
            getActiveElements: ReturnType<typeof vi.fn>;
            setActiveElements: ReturnType<typeof vi.fn>;
        };
        update = vi.fn();
        destroy = vi.fn();

        constructor(canvasEl: unknown, config: Record<string, unknown>) {
            this.canvas = canvasEl;
            this.data = config.data;
            this.options = config.options;
            this.plugins = config.plugins;
            this.tooltip = {
                opacity: 0,
                getActiveElements: vi.fn(() => []),
                setActiveElements: vi.fn(),
            };
            chartInstances.push(this as never);
        }
    }

    class ChartComponent {}
    const ChartPlugin = class {};

    return {
        Chart,
        LineController: ChartComponent,
        LineElement: ChartComponent,
        PointElement: ChartComponent,
        LinearScale: ChartComponent,
        CategoryScale: ChartComponent,
        Tooltip: ChartPlugin,
        Filler: ChartPlugin,
    };
});

const renderStatCard = () => {
    const result = render(() => (
        <HashRouter>
            <Route
                path="/"
                component={() => (
                    <StatCard
                        value={123}
                        label="DNS queries"
                        data={[1, 2, 3]}
                        timeUnits="hours"
                        color="#7F7F7F"
                        cardTheme={CARDS_THEME.QUERIES}
                        linkTo={RoutePath.QueryLog}
                    />
                )}
            />
        </HashRouter>
    ));

    const chart = chartInstances.at(-1);
    if (!chart) throw new Error('Chart instance was not created');

    const canvas = result.container.querySelector('canvas');
    if (!canvas) throw new Error('Canvas was not rendered');
    const wrapper = canvas.parentElement?.parentElement;
    if (!wrapper) throw new Error('Chart wrapper was not rendered');
    const tooltipEl = wrapper.lastElementChild as HTMLDivElement;

    // Show the tooltip the same way Chart.js does through the external
    // handler (sets the inline opacity used by the dismiss logic).
    if (!chart.options.plugins?.tooltip?.external) {
        throw new Error('External tooltip handler is not configured');
    }
    chart.options.plugins.tooltip.external({
        chart,
        tooltip: {
            opacity: 1,
            dataPoints: [{ raw: 7, label: '24 Aug 01:00' }],
            caretX: 10,
            caretY: 10,
        },
    });

    chart.update.mockClear();

    return { chart, canvas, tooltipEl };
};

describe('StatCard chart tooltip', () => {
    it('dismisses the tooltip on pointerdown outside the chart', () => {
        const { chart, tooltipEl } = renderStatCard();

        expect(tooltipEl.style.opacity).toBe('1');

        fireEvent.pointerDown(document.body);

        // Hover dots and cursor line are driven by chart-level active
        // elements, so they must be cleared together with the tooltip.
        expect(chart.setActiveElements).toHaveBeenCalledWith([]);
        expect(chart.tooltip.setActiveElements).toHaveBeenCalledWith([], {
            x: 0,
            y: 0,
        });
        expect(chart.update).toHaveBeenCalledWith('none');
        expect(chart.update).toHaveBeenCalledTimes(1);
    });

    it('keeps the tooltip when the pointerdown is inside the chart', () => {
        const { chart, canvas } = renderStatCard();
        const setActiveElements = chart.tooltip.setActiveElements;
        setActiveElements.mockClear();

        fireEvent.pointerDown(canvas);

        expect(setActiveElements).not.toHaveBeenCalled();
    });

    it('does not dismiss when no tooltip is visible', () => {
        render(() => (
            <HashRouter>
                <Route
                    path="/"
                    component={() => (
                        <StatCard
                            value={123}
                            label="DNS queries"
                            data={[1, 2, 3]}
                            timeUnits="hours"
                            color="#7F7F7F"
                            cardTheme={CARDS_THEME.QUERIES}
                            linkTo={RoutePath.QueryLog}
                        />
                    )}
                />
            </HashRouter>
        ));
        const chart = chartInstances.at(-1);
        if (!chart) throw new Error('Chart instance was not created');

        fireEvent.pointerDown(document.body);

        expect(chart.tooltip.setActiveElements).not.toHaveBeenCalled();
    });
});
