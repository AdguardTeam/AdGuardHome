import React from 'react';

import { useIsDesktop } from 'panel/helpers/useMediaQuery';
import { MOBILE_TABLE_MAX_ROWS } from 'panel/helpers/constants';
import intl from 'panel/common/intl';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
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
    topUpstreamsResponses: UpstreamInfo[];
    numDnsQueries: number;
};

export const TopUpstreams = ({ topUpstreamsResponses, numDnsQueries }: Props) => {
    const isDesktop = useIsDesktop();
    const { sortedData: sortedUpstreams, sortField, sortDirection, handleSort } = useSortedData(topUpstreamsResponses);
    const visibleUpstreams = isDesktop ? sortedUpstreams : sortedUpstreams.slice(0, MOBILE_TABLE_MAX_ROWS);

    const hasStats = topUpstreamsResponses.length > 0;

    return (
        <div className={s.card}>
            <div className={s.cardHeader}>
                <div className={cn(theme.title.h5, s.cardTitle)}>{intl.getMessage('top_upstreams')}</div>
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
                    visibleUpstreams.map((upstream) => {
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

                                <div className={s.queryBar}>
                                    <div
                                        className={s.queryBarFill}
                                        style={{ width: `${percent}%` }}
                                    />
                                </div>
                            </div>
                        );
                    })
                ) : (
                    <EmptyState />
                )}
            </div>
        </div>
    );
};
