import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import { Dropdown } from 'panel/common/ui/Dropdown';
import { formatNumber, formatCompactNumber } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import { getTrackerData } from 'panel/helpers/trackers/trackers';
import { SortableTableHeader } from './SortableTableHeader';
import { TrackerTooltip } from './TrackerTooltip';
import { useSortedData } from '../hooks/useSortedData';

import s from './TableCard.module.pcss';

type DomainInfo = {
    name: string;
    count: number;
};

type Props = {
    topBlockedDomains: DomainInfo[];
    numBlockedFiltering: number;
};

export const TopBlockedDomains = ({ topBlockedDomains, numBlockedFiltering }: Props) => {
    const { sortedData: sortedDomains, sortField, sortDirection, handleSort } = useSortedData(topBlockedDomains);

    const hasStats = topBlockedDomains.length > 0;

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
                <SortableTableHeader
                    nameLabel={intl.getMessage('domain')}
                    countLabel={intl.getMessage('blocked_queries')}
                    sortField={sortField}
                    sortDirection={sortDirection}
                    onSort={handleSort}
                />
            )}

            <div className={s.tableRows}>
                {hasStats ? (
                    sortedDomains.map((domain) => {
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
