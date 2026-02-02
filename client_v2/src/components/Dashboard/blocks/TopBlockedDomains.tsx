import React, { useState, useMemo } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
import { TrackerTooltip } from './TrackerTooltip';

import s from '../Dashboard.module.pcss';

type DomainInfo = {
    name: string;
    count: number;
};

type Props = {
    topBlockedDomains: DomainInfo[];
    numBlockedFiltering: number;
};

type SortField = 'name' | 'count';
type SortDirection = 'asc' | 'desc';

export const TopBlockedDomains = ({ topBlockedDomains, numBlockedFiltering }: Props) => {
    const [sortField, setSortField] = useState<SortField>('count');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

    const hasStats = topBlockedDomains.length > 0;

    const sortedDomains = useMemo(() => {
        return [...topBlockedDomains].sort((a, b) => {
            const modifier = sortDirection === 'asc' ? 1 : -1;
            if (sortField === 'name') {
                return a.name.localeCompare(b.name) * modifier;
            }
            return (a.count - b.count) * modifier;
        });
    }, [topBlockedDomains, sortField, sortDirection]);

    const handleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortField(field);
            setSortDirection(field === 'name' ? 'asc' : 'desc');
        }
    };

    return (
        <div className={cn(s.card, s.cardBlocked)}>
            <div className={s.cardHeader}>
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('top_blocked_domains')}</div>

                {hasStats && (
                    <div className={cn(theme.text.t3, s.cardSubtitle)}>
                        {intl.getMessage('blocked_total', { value: formatCompactNumber(numBlockedFiltering) })}
                    </div>
                )}
            </div>

            {hasStats && (
                <div className={cn(theme.text.t3, theme.text.semibold, s.tableHeader)}>
                    <span
                        className={s.sortableHeader}
                        onClick={() => handleSort('name')}
                    >
                        {intl.getMessage('domain')}
                        {sortField === 'name' ? (
                            <Icon icon="arrow_bottom" className={cn(s.sortIcon, sortDirection === 'asc' && s.sortIconAsc)} />
                        ) : (
                            <span className={s.sortDash}>—</span>
                        )}
                    </span>
                    <span
                        className={s.sortableHeader}
                        onClick={() => handleSort('count')}
                    >
                        {intl.getMessage('blocked_queries')}
                        {sortField === 'count' ? (
                            <Icon icon="arrow_bottom" className={cn(s.sortIcon, sortDirection === 'asc' && s.sortIconAsc)} />
                        ) : (
                            <span className={s.sortDash}>—</span>
                        )}
                    </span>
                </div>
            )}

            <div className={s.tableRows}>
                {hasStats ? (
                    sortedDomains.slice(0, 10).map((domain) => {
                        const percent = numBlockedFiltering > 0
                            ? (domain.count / numBlockedFiltering) * 100
                            : 0;
                        const trackerData = getTrackerData(domain.name);

                        return (
                            <div key={domain.name} className={cn(s.tableRow, s.statRowValue)}>
                                <div className={cn(theme.text.t3, theme.text.condenced, s.tableRowLeft)}>
                                    {trackerData ? (
                                        <Dropdown
                                            menu={<TrackerTooltip trackerData={trackerData} />}
                                            trigger="hover"
                                            position="bottomLeft"
                                            noIcon
                                        >
                                            <Icon icon="eye_open" className={s.tableRowIcon} />
                                        </Dropdown>
                                    ) : (
                                        <div className={s.tableRowDot}></div>
                                    )}
                                    <span className={s.domainName}>{domain.name}</span>
                                </div>

                                <div className={s.tableRowRight}>
                                    <Dropdown
                                        trigger="hover"
                                        position="top"
                                        noIcon
                                        disableAnimation
                                        overlayClassName={s.queryTooltipOverlay}
                                        menu={
                                            <div className={s.queryTooltip}>
                                                {formatNumber(domain.count)} {intl.getMessage('queries').toLowerCase()}
                                            </div>
                                        }
                                    >
                                        <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                                            {formatCompactNumber(domain.count)}

                                            <div className={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}>
                                                ({percent.toFixed(2)}%)
                                            </div>
                                        </div>
                                    </Dropdown>

                                    <div className={s.queryBar}>
                                        <div
                                            className={cn(s.queryBarFill)}
                                            style={{ width: `${percent}%` }}
                                        />
                                    </div>
                                </div>
                            </div>
                        );
                    })
                ) : (
                    <div className={s.emptyState}>
                        <Icon icon="not_found_search" className={s.emptyStateIcon} />

                        <div className={s.emptyStateText}>{intl.getMessage('no_stats_yet')}</div>
                    </div>
                )}
            </div>
        </div>
    );
};
