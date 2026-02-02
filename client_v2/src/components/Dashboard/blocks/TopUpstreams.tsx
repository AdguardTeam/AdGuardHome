import React, { useState, useMemo } from 'react';

import intl from 'panel/common/intl';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';

import s from '../Dashboard.module.pcss';

type UpstreamInfo = {
    name: string;
    count: number;
};

type Props = {
    topUpstreamsResponses: UpstreamInfo[];
    numDnsQueries: number;
};

type SortField = 'name' | 'count';
type SortDirection = 'asc' | 'desc';

export const TopUpstreams = ({ topUpstreamsResponses, numDnsQueries }: Props) => {
    const [sortField, setSortField] = useState<SortField>('count');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

    const hasStats = topUpstreamsResponses.length > 0;

    const sortedUpstreams = useMemo(() => {
        return [...topUpstreamsResponses].sort((a, b) => {
            const modifier = sortDirection === 'asc' ? 1 : -1;
            if (sortField === 'name') {
                return a.name.localeCompare(b.name) * modifier;
            }
            return (a.count - b.count) * modifier;
        });
    }, [topUpstreamsResponses, sortField, sortDirection]);

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
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('top_upstreams')}</div>

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
                        {intl.getMessage('upstream')}
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
                    sortedUpstreams.slice(0, 10).map((upstream) => {
                        const percent = numDnsQueries > 0
                            ? (upstream.count / numDnsQueries) * 100
                            : 0;

                        return (
                            <div key={upstream.name} className={cn(s.tableRow, s.statRowValue)}>
                                <div className={cn(theme.text.t3, theme.text.condenced, s.tableRowLeft)}>
                                    <span className={s.domainName}>{upstream.name}</span>
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
                                                {formatNumber(upstream.count)} {intl.getMessage('queries').toLowerCase()}
                                            </div>
                                        }
                                    >
                                        <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                                            {formatCompactNumber(upstream.count)}

                                            <div className={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}>
                                                ({percent.toFixed(2)}%)
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
