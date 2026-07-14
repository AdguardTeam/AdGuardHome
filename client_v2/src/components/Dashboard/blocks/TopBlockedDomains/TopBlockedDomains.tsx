import { Show, For, createMemo } from 'solid-js';
import cn from 'clsx';

import { useIsDesktop } from 'panel/helpers/useMediaQuery';
import { MOBILE_TABLE_MAX_ROWS } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
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
    topBlockedDomains: DomainInfo[];
    numBlockedFiltering: number;
};

export const TopBlockedDomains = (props: Props) => {
    const isDesktop = useIsDesktop();
    const {
        sortedData: sortedDomains,
        sortField,
        sortDirection,
        handleSort,
    } = useSortedData(() => props.topBlockedDomains);
    const visibleDomains = createMemo(() =>
        isDesktop() ? sortedDomains() : sortedDomains().slice(0, MOBILE_TABLE_MAX_ROWS),
    );

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
                <SortableTableHeader
                    nameLabel={intl.getMessage('domain')}
                    countLabel={intl.getMessage('blocked_queries')}
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
                                                        ({percent().toFixed(2)}%)
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
