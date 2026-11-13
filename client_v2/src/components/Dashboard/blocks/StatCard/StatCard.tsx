import { createMemo, Show, untrack, onMount, onCleanup } from 'solid-js';
import cn from 'clsx';
import {
    Chart,
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    CategoryScale,
    Tooltip,
    Filler,
    type ScriptableContext,
} from 'chart.js';
import { type QueryParams, type RoutePathKey } from 'panel/components/Routes/Paths';

import { withAlpha } from 'panel/helpers/colors';
import { formatNumber } from 'panel/helpers/helpers';
import {
    useChart,
    createCursorLinePlugin,
    createExternalTooltipHandler,
} from 'panel/helpers/useChart';
import { formatHistoryLabel } from 'panel/helpers/lineUtils';
import theme from 'panel/lib/theme';

import { StatLink } from './StatLink';
import s from './StatCard.module.pcss';

Chart.register(
    LineController,
    LineElement,
    PointElement,
    LinearScale,
    CategoryScale,
    Tooltip,
    Filler,
);

export const CARDS_THEME = {
    QUERIES: 'queries',
    ADS: 'ads',
    THREATS: 'threats',
    ADULT: 'adult',
};

export type StatCardProps = {
    value: number;
    label: string;
    data: number[];
    timeUnits: string;
    color: string;
    fillColor?: string;
    percentValue?: number;
    cardTheme: (typeof CARDS_THEME)[keyof typeof CARDS_THEME];
    linkTo: RoutePathKey;
    query?: QueryParams;
};

export const StatCard = (props: StatCardProps) => {
    // Ensure the chart has at least 2 data points
    const paddedData = () => (props.data.length < 2 ? [0, ...props.data] : props.data);

    const chartData = createMemo(() => {
        const data = paddedData();
        const labels = data.map((_, i) => formatHistoryLabel(i, data.length, props.timeUnits));
        const fillColor = props.fillColor ?? props.color;
        return {
            labels,
            datasets: [
                {
                    data: data,
                    borderColor: props.color,
                    borderWidth: 1,
                    backgroundColor: (context: ScriptableContext<'line'>) => {
                        const ctx = context.chart.ctx;
                        const gradient = ctx.createLinearGradient(
                            0,
                            0,
                            0,
                            context.chart.height || 100,
                        );
                        gradient.addColorStop(0, withAlpha(fillColor, 0.3));
                        gradient.addColorStop(1, withAlpha(fillColor, 0));
                        return gradient;
                    },
                    fill: true,
                    clip: false as const,
                    pointRadius: 0,
                    pointHoverRadius: 4,
                    pointHoverBackgroundColor: props.color,
                    tension: 0.4,
                },
            ],
        };
    });

    const cursorLinePlugin = createCursorLinePlugin(() => props.color);

    const externalTooltipHandler = createExternalTooltipHandler(
        () => tooltipEl,
        (dataPoint) => {
            const raw = dataPoint.raw as number;
            const label = dataPoint.label || '';
            return `<div class="${s.chartTooltipValue}">${formatNumber(raw)}</div><div class="${s.chartTooltipDate}">${label}</div>`;
        },
    );

    let tooltipEl!: HTMLDivElement;
    const setTooltipRef = (el: HTMLDivElement) => {
        tooltipEl = el;
    };

    // Dismiss on pointerdown outside the chart (capture phase): Chart.js
    // keeps tooltips open after a tap on touch devices.
    onMount(() => {
        const onPointerDown = (e: PointerEvent) => {
            const el = tooltipEl;
            if (!el || el.style.opacity !== '1') return;

            const target = e.target as Element;
            // Taps on the chart itself are handled by Chart.js.
            if (el.parentElement?.contains(target)) return;

            hideTooltip();
        };

        document.addEventListener('pointerdown', onPointerDown, true);
        onCleanup(() => document.removeEventListener('pointerdown', onPointerDown, true));
    });

    const chartOptions = createMemo(() => ({
        responsive: true,
        maintainAspectRatio: false,
        animation: false as const,
        layout: {
            padding: { top: 6, bottom: 12, left: 8, right: 8 },
        },
        plugins: {
            tooltip: {
                enabled: false,
                external: externalTooltipHandler,
            },
            legend: { display: false },
        },
        scales: {
            x: { display: false },
            y: { display: false, min: 0 },
        },
        interaction: {
            intersect: false,
            mode: 'index' as const,
        },
        elements: {
            line: { tension: 0.4 },
        },
    }));

    const percent = () => props.percentValue ?? 0;
    const percentText = () => `${percent().toFixed(0)}%`;

    const { setCanvasRef, hideTooltip } = untrack(() =>
        useChart(chartData, chartOptions, [cursorLinePlugin]),
    );

    return (
        <div
            class={cn(s.statCard, {
                [s.statCardQueries]: props.cardTheme === CARDS_THEME.QUERIES,
                [s.statCardAds]: props.cardTheme === CARDS_THEME.ADS,
                [s.statCardThreats]: props.cardTheme === CARDS_THEME.THREATS,
                [s.statCardAdult]: props.cardTheme === CARDS_THEME.ADULT,
            })}
        >
            <div class={s.statCardInner}>
                <div class={s.statCardHeader}>
                    <div class={s.statCardHeaderLeft}>
                        <div class={s.statCardValue}>
                            <StatLink to={props.linkTo} query={props.query}>
                                {formatNumber(props.value)}
                            </StatLink>
                        </div>
                    </div>

                    <Show when={props.cardTheme !== CARDS_THEME.QUERIES}>
                        <div class={cn(theme.text.t3, theme.text.t2_tablet, s.statCardPercent)}>
                            <StatLink to={props.linkTo} query={props.query}>
                                {percentText()}
                            </StatLink>
                        </div>
                    </Show>

                    <div class={cn(theme.text.t4, s.statCardLabel)}>
                        <StatLink to={props.linkTo} query={props.query}>
                            {props.label}
                        </StatLink>
                    </div>
                </div>
                <div class={s.statCardChartWrapper}>
                    <div class={s.statCardChart}>
                        <canvas ref={setCanvasRef} />
                    </div>
                    <div ref={setTooltipRef} class={s.chartTooltip} />
                </div>
            </div>
            <div class={cn(theme.text.t3, s.statCardLabel)}>
                <StatLink to={props.linkTo} query={props.query}>
                    {props.label}
                </StatLink>
            </div>
        </div>
    );
};
