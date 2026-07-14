import { Show, For, createMemo } from 'solid-js';

import { useIsDesktop } from 'panel/helpers/useMediaQuery';
import { MOBILE_TABLE_MAX_ROWS } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { SortableTableHeader } from '../SortableTableHeader';
import { TrackerTooltip } from '../TrackerTooltip';
import { EmptyState } from '../EmptyState';
import { useSortedData } from '../../hooks/useSortedData';

import s from '../TableCard.module.pcss';

type DomainInfo = {
    name: string;
    count: number;
};

type Props = {
    topQueriedDomains: DomainInfo[];
    numDnsQueries: number;
};

export const TopQueriedDomains = (props: Props) => {
    const isDesktop = useIsDesktop();
    const {
        sortedData: sortedDomains,
        sortField,
        sortDirection,
        handleSort,
    } = useSortedData(() => props.topQueriedDomains);
    const visibleDomains = createMemo(() =>
        isDesktop() ? sortedDomains() : sortedDomains().slice(0, MOBILE_TABLE_MAX_ROWS),
    );

    const hasStats = createMemo(() => props.topQueriedDomains.length > 0);

    return (
        <div class={s.card}>
            <div class={s.cardHeader}>
                <div class={cn(theme.title.h5, s.cardTitle)}>
                    {intl.getMessage('stats_query_domain')}
                </div>
            </div>

            <Show when={hasStats()}>
                <SortableTableHeader
                    nameLabel={intl.getMessage('domain')}
                    countLabel={intl.getMessage('queries')}
                    sortField={sortField()}
                    sortDirection={sortDirection()}
                    onSort={handleSort}
                />
            </Show>

            <div class={s.tableRows}>
                <Show when={hasStats()} fallback={<EmptyState />}>
                    <For each={visibleDomains()}>
                        {(domain) => {
                            const percent = createMemo(() =>
                                props.numDnsQueries > 0
                                    ? (domain.count / props.numDnsQueries) * 100
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
                                            <Dropdown
                                                menu={<TrackerTooltip trackerData={trackerData!} />}
                                                trigger="hover"
                                                position="bottomLeft"
                                                noIcon
                                            >
                                                <Icon icon="eye_open" class={s.tableRowIcon} />
                                            </Dropdown>
                                        </Show>
                                        <span class={s.domainName}>{domain.name}</span>
                                    </div>

                                    <div class={s.tableRowRight}>
                                        <div class={s.dropdowWrapper}>
                                            <Dropdown
                                                trigger="hover"
                                                position="top"
                                                noIcon
                                                disableAnimation
                                                overlayClass={s.queryTooltipOverlay}
                                                menu={
                                                    <div class={s.queryTooltip}>
                                                        {formatNumber(domain.count)}{' '}
                                                        {intl.getMessage('queries').toLowerCase()}
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
                                                    {formatCompactNumber(domain.count)}

                                                    <div
                                                        class={cn(
                                                            theme.text.t3,
                                                            theme.text.condenced,
                                                            s.queryPercent,
                                                        )}
                                                    >
                                                        ({percent().toFixed(1)}%)
                                                    </div>
                                                </div>
                                            </Dropdown>
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
