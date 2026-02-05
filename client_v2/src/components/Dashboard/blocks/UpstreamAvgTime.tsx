import React from 'react';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';
import { SortableTableHeader } from './SortableTableHeader';
import { useSortedData } from '../hooks/useSortedData';

import s from './TableCard.module.pcss';

type UpstreamInfo = {
    name: string;
    count: number;
};

type Props = {
    topUpstreamsAvgTime: UpstreamInfo[];
    avgProcessingTime: number;
};

export const UpstreamAvgTime = ({ topUpstreamsAvgTime, avgProcessingTime }: Props) => {
    const { sortedData: sortedUpstreams, sortField, sortDirection, handleSort } = useSortedData(topUpstreamsAvgTime);

    const hasStats = topUpstreamsAvgTime.length > 0;

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
                <SortableTableHeader
                    nameLabel={intl.getMessage('upstream')}
                    countLabel={intl.getMessage('response_time')}
                    sortField={sortField}
                    sortDirection={sortDirection}
                    onSort={handleSort}
                />
            )}

            <div className={s.tableRows}>
                {hasStats ? (
                    sortedUpstreams.map((upstream) => (
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
