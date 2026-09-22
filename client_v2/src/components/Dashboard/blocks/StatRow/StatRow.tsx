import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { Icon, type IconType } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import { Link } from 'panel/common/ui/Link';
import { type QueryParams, type RoutePathKey } from 'panel/components/Routes/Paths';
import cn from 'clsx';
import { formatCompactNumber } from 'panel/helpers/helpers';

import { RowTooltip } from '../RowTooltip';
import s from './StatRow.module.pcss';

export type StatRowProps = {
    icon: IconType;
    label: string;
    value: string | number;
    percent?: number;
    isTotal?: boolean;
    isQueriesValue?: boolean;
    tooltip: string;
    linkTo?: RoutePathKey;
    query?: QueryParams;
    rowTheme:
        | 'dnsQueries'
        | 'adsBlocked'
        | 'threatsBlocked'
        | 'adultWebsitesBlocked'
        | 'safeSearchUsed'
        | 'averageProcessingTime';
};

export const StatRow = (props: StatRowProps) => {
    const isQueriesValue = () => props.isQueriesValue !== false;

    const queriesValue = () =>
        typeof props.value === 'number' ? formatCompactNumber(props.value) : props.value;

    const queriesPercent = () => (
        <div class={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}>
            <Show
                when={props.isTotal}
                fallback={
                    <Show when={props.percent !== undefined && props.percent! > 0}>
                        <span>({props.percent!.toFixed(1)}%)</span>
                    </Show>
                }
            >
                <span>({intl.getMessage('total')})</span>
            </Show>
        </div>
    );

    const rowClass = () => cn(s.statRow, s[props.rowTheme], props.linkTo && s.statRowLink);

    const rowContent = () => (
        <>
            <div class={s.statRowDropdown}>
                <div
                    class={cn(
                        theme.text.t3,
                        theme.text.condenced,
                        theme.common.noShrink,
                        s.statRowLeft,
                    )}
                >
                    <Icon icon={props.icon} class={s.tableRowIcon} />
                    {props.label}
                </div>
            </div>

            <div class={s.statRowValue}>
                <Show
                    when={isQueriesValue()}
                    fallback={
                        <div class={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                            {props.value}
                        </div>
                    }
                >
                    <div class={s.dropdownWrapper}>
                        <div class={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                            <span class={s.queryCountValue}>{queriesValue()}</span>
                            {queriesPercent()}
                        </div>
                    </div>
                </Show>

                <Show when={isQueriesValue()}>
                    <div class={s.queryBar}>
                        <div
                            class={s.queryBarFill}
                            style={{ width: `${props.isTotal ? 100 : props.percent || 0}%` }}
                        />
                    </div>
                </Show>
            </div>

            <Show when={isQueriesValue()}>
                <div class={cn(s.queryBar, s.queryBarMobile)}>
                    <div
                        class={s.queryBarFill}
                        style={{ width: `${props.isTotal ? 100 : props.percent || 0}%` }}
                    />
                </div>
            </Show>
        </>
    );

    // The description is plain text, so its overlay must not swallow the
    // pointer: it overlaps the row below and would otherwise keep that row from
    // receiving its own hover.
    return (
        <RowTooltip
            content={<div class={cn(theme.text.t3, s.statTooltip)}>{props.tooltip}</div>}
            overlayClass={s.queryTooltipOverlay}
        >
            <Show when={props.linkTo} fallback={<div class={rowClass()}>{rowContent()}</div>}>
                {(to) => (
                    <Link to={to()} query={props.query} class={rowClass()}>
                        {rowContent()}
                    </Link>
                )}
            </Show>
        </RowTooltip>
    );
};
