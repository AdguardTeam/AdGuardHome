import { Show, For, createMemo } from 'solid-js';
import intl from 'panel/common/intl';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { Tooltip } from 'panel/common/ui/Tooltip';
import cn from 'clsx';
import { TableHeader } from '../TableHeader';
import { EmptyState } from '../EmptyState';
import { useSortedData } from '../../hooks/useSortedData';

import s from '../TableCard.module.pcss';

type UpstreamInfo = {
    name: string;
    count: number;
};

type Props = {
    topUpstreamsResponses: UpstreamInfo[];
    numDnsQueries: number;
};

export const TopUpstreams = (props: Props) => {
    const { sortedData: sortedUpstreams } = useSortedData(() => props.topUpstreamsResponses);

    const hasStats = createMemo(() => props.topUpstreamsResponses.length > 0);

    return (
        <div class={s.card}>
            <div class={s.cardHeader}>
                <div class={cn(theme.title.h5, s.cardTitle)}>
                    {intl.getMessage('top_upstreams')}
                </div>
            </div>

            <Show when={hasStats()}>
                <TableHeader
                    nameLabel={intl.getMessage('upstream')}
                    countLabel={intl.getMessage('queries')}
                />
            </Show>

            <div class={s.tableRows}>
                <Show when={hasStats()} fallback={<EmptyState />}>
                    <For each={sortedUpstreams()}>
                        {(upstream) => {
                            const percent = createMemo(() =>
                                props.numDnsQueries > 0
                                    ? (upstream.count / props.numDnsQueries) * 100
                                    : 0,
                            );

                            return (
                                <div class={cn(s.tableRow, s.statRowValue)}>
                                    <div
                                        class={cn(
                                            theme.text.t3,
                                            theme.text.condenced,
                                            s.tableRowLeft,
                                        )}
                                    >
                                        <span class={s.domainName}>{upstream.name}</span>
                                    </div>

                                    <div class={s.tableRowRight}>
                                        <div class={s.dropdowWrapper}>
                                            <Tooltip
                                                position="top"
                                                overlayClass={s.queryTooltipOverlay}
                                                content={
                                                    <div class={s.queryTooltip}>
                                                        {intl.getMessage('queries_tooltip', {
                                                            value: formatNumber(upstream.count),
                                                        })}
                                                    </div>
                                                }
                                            >
                                                <div
                                                    class={cn(
                                                        theme.text.t3,
                                                        theme.text.condenced,
                                                        s.queryCount,
                                                    )}
                                                >
                                                    {formatCompactNumber(upstream.count)}

                                                    <div
                                                        class={cn(
                                                            theme.text.t3,
                                                            theme.text.condenced,
                                                            s.queryPercent,
                                                        )}
                                                    >
                                                        ({percent().toFixed(2)}%)
                                                    </div>
                                                </div>
                                            </Tooltip>
                                        </div>

                                        <div class={s.queryBar}>
                                            <div
                                                class={cn(s.queryBarFill)}
                                                style={{ width: `${percent()}%` }}
                                            />
                                        </div>
                                    </div>

                                    <div class={s.queryBar}>
                                        <div
                                            class={s.queryBarFill}
                                            style={{ width: `${percent()}%` }}
                                        />
                                    </div>
                                </div>
                            );
                        }}
                    </For>
                </Show>
            </div>
        </div>
    );
};
