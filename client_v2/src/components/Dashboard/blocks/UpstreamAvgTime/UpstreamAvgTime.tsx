import { Show, For, createMemo } from 'solid-js';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
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
    topUpstreamsAvgTime: UpstreamInfo[];
    avgProcessingTime: number;
};

export const UpstreamAvgTime = (props: Props) => {
    const { sortedData: sortedUpstreams } = useSortedData(() => props.topUpstreamsAvgTime);

    const hasStats = createMemo(() => props.topUpstreamsAvgTime.length > 0);

    return (
        <div class={s.card}>
            <div class={s.cardHeader}>
                <div class={cn(theme.title.h5, s.cardTitle)}>
                    {intl.getMessage('average_upstream_response_time')}
                </div>

                <Show when={hasStats()}>
                    <div class={cn(theme.text.t3, s.cardSubtitle)}>
                        {(props.avgProcessingTime ?? 0).toFixed(0)}{' '}
                        {intl.getMessage('milliseconds_abbreviation')}
                    </div>
                </Show>
            </div>

            <Show when={hasStats()}>
                <TableHeader
                    nameLabel={intl.getMessage('upstream')}
                    countLabel={intl.getMessage('response_time')}
                />
            </Show>

            <div class={s.tableRows}>
                <Show when={hasStats()} fallback={<EmptyState />}>
                    <For each={sortedUpstreams()}>
                        {(upstream) => (
                            <div class={cn(s.tableRow)}>
                                <div
                                    class={cn(theme.text.t3, theme.text.condenced, s.tableRowLeft)}
                                >
                                    <span class={s.domainName}>{upstream.name}</span>
                                </div>
                                <div class={s.tableRowRight}>
                                    <div
                                        class={cn(
                                            theme.text.t3,
                                            theme.text.condenced,
                                            s.queryCount,
                                        )}
                                    >
                                        {(upstream.count ?? 0).toFixed(0)}{' '}
                                        {intl.getMessage('milliseconds_abbreviation')}
                                    </div>
                                </div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
