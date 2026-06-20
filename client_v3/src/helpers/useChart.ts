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

/**
 * SolidJS primitive that manages a Chart.js instance lifecycle.
 *
 * Returns a `ref` callback to attach to a `<canvas>` element.
 * Handles creation, reactive data/options sync, and cleanup.
 */
export function useChart(
    data: () => ChartData<'line'>,
    options: () => ChartOptions<'line'>,
    plugins?: Plugin<'line'>[],
): (el: HTMLCanvasElement) => void {
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

    onCleanup(() => {
        chart?.destroy();
        chart = undefined;
    });

    return setCanvasRef;
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

        el.innerHTML = renderContent(dataPoint);
        el.style.opacity = '1';

        const tooltipWidth = el.offsetWidth;
        const tooltipHeight = el.offsetHeight;

        // Position to the right of cursor; fall back to left near viewport edge
        let left = rect.left + tooltip.caretX + 12;
        if (left + tooltipWidth > window.innerWidth - 12) {
            left = rect.left + tooltip.caretX - tooltipWidth - 12;
        }

        let top = rect.top + tooltip.caretY - tooltipHeight / 2;
        // Keep tooltip vertically within viewport
        if (top < 8) top = 8;
        if (top + tooltipHeight > window.innerHeight - 8) {
            top = window.innerHeight - tooltipHeight - 8;
        }

        el.style.left = `${left}px`;
        el.style.top = `${top}px`;
    };
}
