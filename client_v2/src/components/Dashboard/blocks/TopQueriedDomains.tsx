import React from 'react';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { SortableTableHeader } from './SortableTableHeader';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
import theme from 'panel/lib/theme';
import cn from 'clsx';
import { TrackerTooltip } from './TrackerTooltip';
import { useSortedData } from '../hooks/useSortedData';

import s from './TableCard.module.pcss';

type DomainInfo = {
    name: string;
    count: number;
};

type Props = {
    topQueriedDomains: DomainInfo[];
    numDnsQueries: number;
};

export const TopQueriedDomains = ({ topQueriedDomains, numDnsQueries }: Props) => {
    const { sortedData: sortedDomains, sortField, sortDirection, handleSort } = useSortedData(topQueriedDomains);

    const hasStats = topQueriedDomains.length > 0;

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
                <SortableTableHeader
                    nameLabel={intl.getMessage('domain')}
                    countLabel={intl.getMessage('queries')}
                    sortField={sortField}
                    sortDirection={sortDirection}
                    onSort={handleSort}
                />
            )}

            <div className={s.tableRows}>
                {hasStats ? (
                    sortedDomains.map((domain) => {
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
