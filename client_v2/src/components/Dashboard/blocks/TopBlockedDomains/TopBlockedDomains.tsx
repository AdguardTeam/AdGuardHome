import { Show, For, createMemo } from 'solid-js';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Tooltip } from 'panel/common/ui/Tooltip';
import { QueriesTooltip } from 'panel/common/ui/QueriesTooltip';
import { Link } from 'panel/common/ui/Link';
import { RoutePath } from 'panel/components/Routes/Paths';
import { formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
import { TableHeader } from '../TableHeader';
import { TrackerTooltip } from '../TrackerTooltip';
import { EmptyState } from '../EmptyState';
import { useSortedData } from '../../hooks/useSortedData';

import s from '../TableCard.module.pcss';

type DomainInfo = {
    name: string;
    count: number;
};

type Props = {
    topBlockedDomains: DomainInfo[];
    numBlockedFiltering: number;
};

export const TopBlockedDomains = (props: Props) => {
    const { sortedData: sortedDomains } = useSortedData(() => props.topBlockedDomains);

    const hasStats = createMemo(() => props.topBlockedDomains.length > 0);

    return (
        <div class={cn(s.card, s.cardBlocked)}>
            <div class={s.cardHeader}>
                <div class={cn(theme.title.h5, s.cardTitle)}>
                    {intl.getMessage('top_blocked_domains')}
                </div>

                <Show when={hasStats()}>
                    <div class={cn(theme.text.t3, s.cardSubtitle)}>
                        {intl.getMessage('blocked_total', {
                            value: formatCompactNumber(props.numBlockedFiltering),
                        })}
                    </div>
                </Show>
            </div>

            <Show when={hasStats()}>
                <TableHeader
                    nameLabel={intl.getMessage('domain')}
                    countLabel={intl.getMessage('blocked_queries')}
                />
            </Show>

            <div class={s.tableRows}>
                <Show when={hasStats()} fallback={<EmptyState />}>
                    <For each={sortedDomains()}>
                        {(domain) => {
                            const percent = createMemo(() =>
                                props.numBlockedFiltering > 0
                                    ? (domain.count / props.numBlockedFiltering) * 100
                                    : 0,
                            );
                            const trackerData = getTrackerData(domain.name);

                            return (
                                <div class={cn(s.tableRow, s.statRowValue)}>
                                    <div
                                        class={cn(
                                            theme.text.t3,
                                            theme.text.condenced,
                                            s.tableRowLeft,
                                        )}
                                    >
                                        <Show
                                            when={trackerData}
                                            fallback={<div class={s.tableRowDot} />}
                                        >
                                            <Tooltip
                                                content={
                                                    <TrackerTooltip trackerData={trackerData!} />
                                                }
                                                position="bottomLeft"
                                                class={theme.common.noShrink}
                                            >
                                                <Icon icon="eye_open" class={s.tableRowIcon} />
                                            </Tooltip>
                                        </Show>
                                        <span class={s.domainName}>{domain.name}</span>
                                    </div>

                                    <div class={s.tableRowRight}>
                                        <div class={s.dropdowWrapper}>
                                            <QueriesTooltip count={domain.count}>
                                                <div
                                                    class={cn(
                                                        theme.text.t3,
                                                        theme.text.condenced,
                                                        s.queryCount,
                                                    )}
                                                >
                                                    <Link
                                                        to={RoutePath.QueryLog}
                                                        query={{ search: `"${domain.name}"` }}
                                                        class={cn(
                                                            theme.text.t3,
                                                            theme.text.condenced,
                                                            s.queryCountLink,
                                                        )}
                                                    >
                                                        {formatCompactNumber(domain.count)}
                                                    </Link>

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
                                            </QueriesTooltip>
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
