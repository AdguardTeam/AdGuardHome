import React, { useState, useMemo } from 'react';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';

import s from '../Dashboard.module.pcss';

type UpstreamInfo = {
    name: string;
    count: number;
};

type Props = {
    topUpstreamsAvgTime: UpstreamInfo[];
    avgProcessingTime: number;
};

type SortField = 'name' | 'count';
type SortDirection = 'asc' | 'desc';

export const UpstreamAvgTime = ({ topUpstreamsAvgTime, avgProcessingTime }: Props) => {
    const [sortField, setSortField] = useState<SortField>('count');
    const [sortDirection, setSortDirection] = useState<SortDirection>('desc');

    const hasStats = topUpstreamsAvgTime.length > 0;

    const sortedUpstreams = useMemo(() => {
        return [...topUpstreamsAvgTime].sort((a, b) => {
            const modifier = sortDirection === 'asc' ? 1 : -1;
            if (sortField === 'name') {
                return a.name.localeCompare(b.name) * modifier;
            }
            return (a.count - b.count) * modifier;
        });
    }, [topUpstreamsAvgTime, sortField, sortDirection]);

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
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('average_upstream_response_time')}</div>

                {hasStats && (
                    <div className={cn(theme.text.t3, s.cardSubtitle)}>
                        {avgProcessingTime.toFixed(0)} {intl.getMessage('milliseconds_abbreviation')}
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
                        {intl.getMessage('response_time')}
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
                    sortedUpstreams.slice(0, 10).map((upstream) => (
                        <div key={upstream.name} className={cn(s.tableRow)}>
                            <div className={cn(theme.text.t3, theme.text.condenced, s.tableRowLeft)}>
                                <span className={s.domainName}>{upstream.name}</span>
                            </div>
                            <div className={s.tableRowRight}>
                                <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                                    {upstream.count.toFixed(0)} {intl.getMessage('milliseconds_abbreviation')}
                                </div>
                            </div>
                        </div>
                    ))
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
