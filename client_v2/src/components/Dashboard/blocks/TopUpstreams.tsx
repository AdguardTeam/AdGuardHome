import React from 'react';

import intl from 'panel/common/intl';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
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
    topUpstreamsResponses: UpstreamInfo[];
    numDnsQueries: number;
};

export const TopUpstreams = ({ topUpstreamsResponses, numDnsQueries }: Props) => {
    const { sortedData: sortedUpstreams, sortField, sortDirection, handleSort } = useSortedData(topUpstreamsResponses);

    const hasStats = topUpstreamsResponses.length > 0;

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
                <SortableTableHeader
                    nameLabel={intl.getMessage('upstream')}
                    countLabel={intl.getMessage('queries')}
                    sortField={sortField}
                    sortDirection={sortDirection}
                    onSort={handleSort}
                />
            )}

            <div className={s.tableRows}>
                {hasStats ? (
                    sortedUpstreams.map((upstream) => {
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
