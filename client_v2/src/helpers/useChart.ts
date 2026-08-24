import { onMount, createEffect, onCleanup } from 'solid-js';
import {
    Chart,
    type ChartData,
    type ChartOptions,
    type Plugin,
    type TooltipItem,
    type TooltipModel,
} from 'chart.js';

/**
 * Clones only the array properties that Chart.js internally mutates,
 * preserving callback functions (e.g. backgroundColor) that can't be
 * deep-cloned. This prevents "Cannot mutate a Store directly" warnings
 * when chart data originates from Solid reactive proxies.
 */
function cloneChartData(d: ChartData<'line'>): ChartData<'line'> {
    return {
        ...d,
        labels: Array.isArray(d.labels) ? [...d.labels] : d.labels,
        datasets: d.datasets.map((ds) => ({
            ...ds,
            data: [...ds.data],
        })),
    };
}

export type UseChartApi = {
    /** Ref callback to attach to a `<canvas>` element. */
    setCanvasRef: (el: HTMLCanvasElement) => void;
    /** Hides the external tooltip and removes the hover cursor line. */
    hideTooltip: () => void;
};

/**
 * SolidJS primitive that manages a Chart.js instance lifecycle.
 *
 * Returns a `ref` callback and a `hideTooltip` method to dismiss the
 * hover state on outside taps. Handles reactive sync and cleanup.
 */
export function useChart(
    data: () => ChartData<'line'>,
    options: () => ChartOptions<'line'>,
    plugins?: Plugin<'line'>[],
): UseChartApi {
    let canvasEl!: HTMLCanvasElement;
    let chart: Chart | undefined;

    const setCanvasRef = (el: HTMLCanvasElement) => {
        canvasEl = el;
    };

    onMount(() => {
        chart = new Chart(canvasEl, {
            type: 'line',
            // Clone arrays to prevent Chart.js from mutating Solid reactive proxies
            data: cloneChartData(data()),
            options: options(),
            plugins,
        });
    });

    createEffect(() => {
        if (!chart) return;
        // Clone arrays to prevent Chart.js from mutating Solid reactive proxies
        chart.data = cloneChartData(data());
        chart.options = options();
        chart.update('none');
    });

    /** Clears the hover state (dots, cursor line, tooltip) and redraws. */
    const hideTooltip = () => {
        if (!chart) return;
        chart.setActiveElements([]); // hover dots + cursor line
        chart.tooltip.setActiveElements([], { x: 0, y: 0 }); // tooltip DOM
        chart.update('none');
    };

    onCleanup(() => {
        chart?.destroy();
        chart = undefined;
    });

    return { setCanvasRef, hideTooltip };
}

/**
 * Chart.js plugin that draws a vertical cursor line in the given color
 * at the active hover position. Matches recharts' `cursor` prop behavior.
 */
export function createCursorLinePlugin(color: string): Plugin<'line'> {
    return {
        id: 'cursorLine',
        afterDraw: (chart: Chart<'line'>) => {
            const activeElements = chart.tooltip?.getActiveElements();
            if (!activeElements?.length) return;
            const active = activeElements[0];
            const x = active.element.x;
            const { top, bottom } = chart.scales.y;
            const ctx = chart.ctx;
            ctx.save();
            ctx.beginPath();
            ctx.moveTo(x, top);
            ctx.lineTo(x, bottom);
            ctx.strokeStyle = color;
            ctx.lineWidth = 1;
            ctx.stroke();
            ctx.restore();
        },
    };
}

/**
 * Creates a Chart.js `external` tooltip handler that renders custom HTML
 * into a DOM element with recharts-style positioning (right of cursor,
 * flipping left near viewport edges).
 *
 * The tooltip element must be positioned with `position: absolute` and its
 * nearest positioned ancestor should be the chart container. The handler
 * converts viewport coordinates to coordinates relative to that ancestor,
 * so the tooltip scrolls together with the chart instead of staying fixed
 * to the viewport.
 *
 * @param getTooltipEl - accessor for the tooltip DOM element
 * @param renderContent - returns the innerHTML string for a data point
 */
export function createExternalTooltipHandler(
    getTooltipEl: () => HTMLDivElement | undefined,
    renderContent: (dataPoint: TooltipItem<'line'>) => string,
): (context: { chart: Chart; tooltip: TooltipModel<'line'> }) => void {
    return (context) => {
        const el = getTooltipEl();
        if (!el) return;

        const { tooltip } = context;
        if (tooltip.opacity === 0) {
            el.style.opacity = '0';
            return;
        }

        const dataPoint = tooltip.dataPoints?.[0];
        if (!dataPoint) return;

        const { chart } = context;
        const rect = chart.canvas.getBoundingClientRect();
        // The tooltip is absolutely positioned relative to its nearest
        // positioned ancestor, so coordinates must be relative to it.
        const parentRect = el.parentElement?.getBoundingClientRect() ?? rect;

        el.innerHTML = renderContent(dataPoint);
        el.style.opacity = '1';

        const tooltipWidth = el.offsetWidth;
        const tooltipHeight = el.offsetHeight;

        // Position to the right of cursor; fall back to left near viewport edge
        let left = rect.left + tooltip.caretX + 12;
        if (left + tooltipWidth > window.innerWidth - 12) {
            left = rect.left + tooltip.caretX - tooltipWidth - 12;
        }
        left -= parentRect.left;

        let top = rect.top + tooltip.caretY - tooltipHeight / 2;
        // Keep tooltip vertically within viewport
        if (top < 8) top = 8;
        if (top + tooltipHeight > window.innerHeight - 8) {
            top = window.innerHeight - tooltipHeight - 8;
        }
        top -= parentRect.top;

        el.style.left = `${left}px`;
        el.style.top = `${top}px`;
    };
}
