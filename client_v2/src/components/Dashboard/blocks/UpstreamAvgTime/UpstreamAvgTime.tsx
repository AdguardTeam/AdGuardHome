import React from 'react';

import { useIsDesktop } from 'panel/helpers/useMediaQuery';
import { MOBILE_TABLE_MAX_ROWS } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { SortableTableHeader } from '../SortableTableHeader';
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

export const UpstreamAvgTime = ({ topUpstreamsAvgTime, avgProcessingTime }: Props) => {
    const isDesktop = useIsDesktop();
    const {
        sortedData: sortedUpstreams,
        sortField,
        sortDirection,
        handleSort,
    } = useSortedData(topUpstreamsAvgTime);
    const visibleUpstreams = isDesktop
        ? sortedUpstreams
        : sortedUpstreams.slice(0, MOBILE_TABLE_MAX_ROWS);

    const hasStats = topUpstreamsAvgTime.length > 0;

    return (
        <div className={s.card}>
            <div className={s.cardHeader}>
                <div className={cn(theme.title.h5, s.cardTitle)}>
                    {intl.getMessage('average_upstream_response_time')}
                </div>

                {hasStats && (
                    <div className={cn(theme.text.t3, s.cardSubtitle)}>
                        {avgProcessingTime.toFixed(0)}{' '}
                        {intl.getMessage('milliseconds_abbreviation')}
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
                    visibleUpstreams.map((upstream) => (
                        <div key={upstream.name} className={cn(s.tableRow)}>
                            <div
                                className={cn(theme.text.t3, theme.text.condenced, s.tableRowLeft)}
                            >
                                <span className={s.domainName}>{upstream.name}</span>
                            </div>
                            <div className={s.tableRowRight}>
                                <div
                                    className={cn(
                                        theme.text.t3,
                                        theme.text.condenced,
                                        s.queryCount,
                                    )}
                                >
                                    {upstream.count.toFixed(0)}{' '}
                                    {intl.getMessage('milliseconds_abbreviation')}
                                </div>
                            </div>
                        </div>
                    ))
                ) : (
                    <EmptyState />
                )}
            </div>
        </div>
    );
};
