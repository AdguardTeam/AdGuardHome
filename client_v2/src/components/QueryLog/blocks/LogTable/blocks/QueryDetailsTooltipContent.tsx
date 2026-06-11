import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { captitalizeWords } from 'panel/helpers/helpers';
import theme from 'panel/lib/theme';
import {
    formatLogDate,
    formatLogTimeDetailed,
    getProtocolName,
} from 'panel/components/QueryLog/helpers';
import { LogEntry } from 'panel/components/QueryLog/types';

import s from '../LogTable.module.pcss';

type Props = {
    row: LogEntry;
};

const renderValue = (value: React.ReactNode) => (
    <span className={cn(s.queryDetailsTooltipValue, theme.text.t3)}>{value}</span>
);

export const QueryDetailsTooltipContent = ({ row }: Props) => {
    const trackerSource = row.tracker?.sourceData;
    const displayDomain = row.unicodeName || row.domain;

    return (
        <div className={s.queryDetailsTooltipContent} onClick={(e) => e.stopPropagation()}>
            <div className={cn(s.queryDetailsTooltipTitle, theme.text.t2, theme.text.semibold)}>
                {intl.getMessage('query_details')}
            </div>

            <div className={s.queryDetailsTooltipSection}>
                <div className={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_time', {
                        value: formatLogTimeDetailed(row.time),
                        span: renderValue,
                    })}
                </div>
                <div className={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_date', {
                        value: formatLogDate(row.time),
                        span: renderValue,
                    })}
                </div>
                <div className={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_domain', {
                        value: displayDomain,
                        span: renderValue,
                    })}
                </div>
                <div className={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_type', {
                        value: row.type,
                        span: renderValue,
                    })}
                </div>
                <div className={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_protocol', {
                        value: getProtocolName(row.client_proto),
                        span: renderValue,
                    })}
                </div>
            </div>

            {row.tracker && (
                <>
                    <div
                        className={cn(
                            s.queryDetailsTooltipTitle,
                            s.queryDetailsTooltipTitleSeparated,
                            theme.text.t2,
                            theme.text.semibold,
                        )}
                    >
                        {intl.getMessage('known_tracker')}
                    </div>

                    <div className={s.queryDetailsTooltipSection}>
                        <div className={s.queryDetailsTooltipItem}>
                            {intl.getMessage('query_log_detail_name', {
                                value: row.tracker.name,
                                span: renderValue,
                            })}
                        </div>
                        <div className={s.queryDetailsTooltipItem}>
                            {intl.getMessage('query_log_detail_category', {
                                value: captitalizeWords(row.tracker.category),
                                span: renderValue,
                            })}
                        </div>
                        {trackerSource?.name && (
                            <div className={s.queryDetailsTooltipItem}>
                                {intl.getMessage('query_log_detail_source', {
                                    value: trackerSource.name,
                                    span: (content: React.ReactNode) =>
                                        trackerSource.url ? (
                                            <a
                                                href={trackerSource.url}
                                                target="_blank"
                                                rel="noopener noreferrer"
                                                className={cn(
                                                    s.queryDetailsTooltipLink,
                                                    theme.status.statusGreen,
                                                )}
                                            >
                                                {content}
                                            </a>
                                        ) : (
                                            <span
                                                className={cn(
                                                    s.queryDetailsTooltipValue,
                                                    theme.text.t3,
                                                )}
                                            >
                                                {content}
                                            </span>
                                        ),
                                })}
                            </div>
                        )}
                    </div>
                </>
            )}
        </div>
    );
};
