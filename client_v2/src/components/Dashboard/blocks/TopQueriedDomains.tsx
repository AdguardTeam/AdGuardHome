import React, { useState, useMemo } from 'react';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
import { TrackerTooltip } from './TrackerTooltip';
import theme from 'panel/lib/theme';
import cn from 'clsx';

import s from '../Dashboard.module.pcss';

type DomainInfo = {
    name: string;
    count: number;
};

type Props = {
    topQueriedDomains: DomainInfo[];
    numDnsQueries: number;
};

type SortField = 'name' | 'count';
type SortDirection = 'asc' | 'desc';

export const TopQueriedDomains = ({ topQueriedDomains, numDnsQueries }: Props) => {
    const [sortField, setSortField] = useState<SortField>('count');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

    const hasStats = topQueriedDomains.length > 0;

    const sortedDomains = useMemo(() => {
        return [...topQueriedDomains].sort((a, b) => {
            const modifier = sortDirection === 'asc' ? 1 : -1;
            if (sortField === 'name') {
                return a.name.localeCompare(b.name) * modifier;
            }
            return (a.count - b.count) * modifier;
        });
    }, [topQueriedDomains, sortField, sortDirection]);

    const handleSort = (field: SortField) => {
        if (sortField === field) {
            setSortDirection(sortDirection === 'asc' ? 'desc' : 'asc');
        } else {
            setSortField(field);
            setSortDirection(field === 'name' ? 'asc' : 'desc');
        }
    };

    return (
        <div className={s.card}>
            <div className={s.cardHeader}>
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('stats_query_domain')}</div>

                {hasStats && (
                    <div className={cn(theme.text.t3, s.cardSubtitle)}>
                        {intl.getPlural('queries_total', formatCompactNumber(numDnsQueries))}
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
                        {intl.getMessage('queries')}
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
                        const percent = numDnsQueries > 0 ? (domain.count / numDnsQueries) * 100 : 0;
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
                                            <Icon icon="eye_opened" className={s.tableRowIcon} />
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
                                                ({percent.toFixed(1)}%)
                                            </div>
                                        </div>
                                    </Dropdown>

                                    <div className={s.queryBar}>
                                        <div
                                            className={s.queryBarFill}
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
