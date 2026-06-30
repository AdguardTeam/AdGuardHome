import { Show } from 'solid-js';
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

const renderValue = (value: any) => (
    <span class={cn(s.queryDetailsTooltipValue, theme.text.t3)}>{value}</span>
);

export const QueryDetailsTooltipContent = (props: Props) => {
    const trackerSource = () => props.row.tracker?.sourceData;
    const displayDomain = () => props.row.unicodeName || props.row.domain;

    return (
        <div class={s.queryDetailsTooltipContent} onClick={(e) => e.stopPropagation()}>
            <div class={cn(s.queryDetailsTooltipTitle, theme.text.t2, theme.text.semibold)}>
                {intl.getMessage('query_details')}
            </div>

            <div class={s.queryDetailsTooltipSection}>
                <div class={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_time', {
                        value: formatLogTimeDetailed(props.row.time),
                        span: renderValue,
                    })}
                </div>
                <div class={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_date', {
                        value: formatLogDate(props.row.time),
                        span: renderValue,
                    })}
                </div>
                <div class={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_domain', {
                        value: displayDomain(),
                        span: renderValue,
                    })}
                </div>
                <div class={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_type', {
                        value: props.row.type,
                        span: renderValue,
                    })}
                </div>
                <div class={s.queryDetailsTooltipItem}>
                    {intl.getMessage('query_log_detail_protocol', {
                        value: getProtocolName(props.row.client_proto),
                        span: renderValue,
                    })}
                </div>
            </div>

            <Show when={props.row.tracker}>
                <div
                    class={cn(
                        s.queryDetailsTooltipTitle,
                        s.queryDetailsTooltipTitleSeparated,
                        theme.text.t2,
                        theme.text.semibold,
                    )}
                >
                    {intl.getMessage('known_tracker')}
                </div>

                <div class={s.queryDetailsTooltipSection}>
                    <div class={s.queryDetailsTooltipItem}>
                        {intl.getMessage('query_log_detail_name', {
                            value: props.row.tracker!.name,
                            span: renderValue,
                        })}
                    </div>
                    <div class={s.queryDetailsTooltipItem}>
                        {intl.getMessage('query_log_detail_category', {
                            value: captitalizeWords(props.row.tracker!.category),
                            span: renderValue,
                        })}
                    </div>
                    <Show when={trackerSource()?.name}>
                        <div class={s.queryDetailsTooltipItem}>
                            {intl.getMessage('query_log_detail_source', {
                                value: trackerSource()!.name,
                                span: (content: any) =>
                                    trackerSource()!.url ? (
                                        <a
                                            href={trackerSource()!.url}
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            class={cn(
                                                s.queryDetailsTooltipLink,
                                                theme.status.statusGreen,
                                            )}
                                        >
                                            {content}
                                        </a>
                                    ) : (
                                        <span class={cn(s.queryDetailsTooltipValue, theme.text.t3)}>
                                            {content}
                                        </span>
                                    ),
                            })}
                        </div>
                    </Show>
                </div>
            </Show>
        </div>
    );
};
