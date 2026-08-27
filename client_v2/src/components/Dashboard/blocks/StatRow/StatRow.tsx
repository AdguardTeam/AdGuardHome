import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { Icon, type IconType } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import { Tooltip } from 'panel/common/ui/Tooltip';
import { QueriesTooltip } from 'panel/common/ui/QueriesTooltip';
import { Link } from 'panel/common/ui/Link';
import { type QueryParams, type RoutePathKey } from 'panel/components/Routes/Paths';
import cn from 'clsx';
import { formatCompactNumber } from 'panel/helpers/helpers';

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

    return (
        <div class={cn(s.statRow, s[props.rowTheme])}>
            <div class={s.statRowDropdown}>
                <Tooltip
                    position="bottomLeft"
                    overlayClass={s.queryTooltipOverlay}
                    content={<div class={cn(theme.text.t3, s.statTooltip)}>{props.tooltip}</div>}
                    class={theme.common.noShrink}
                >
                    <div class={cn(theme.text.t3, theme.text.condenced, s.statRowLeft)}>
                        <Icon icon={props.icon} class={s.tableRowIcon} />
                        {props.label}
                    </div>
                </Tooltip>
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
                        <QueriesTooltip count={props.value as number}>
                            <div class={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                                <Show when={props.linkTo} fallback={queriesValue()}>
                                    <Link
                                        to={props.linkTo}
                                        query={props.query}
                                        class={cn(
                                            theme.text.t3,
                                            theme.text.condenced,
                                            s.queryCount,
                                            s.queryCountLink,
                                        )}
                                    >
                                        {queriesValue()}
                                    </Link>
                                </Show>
                                {queriesPercent()}
                            </div>
                        </QueriesTooltip>
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
        </div>
    );
};
