import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { Icon, type IconType } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import { Dropdown } from 'panel/common/ui/Dropdown';
import cn from 'clsx';
import { formatCompactNumber, formatNumber } from 'panel/helpers/helpers';

import s from './StatRow.module.pcss';

export type StatRowProps = {
    icon: IconType;
    label: string;
    value: string | number;
    percent?: number;
    isTotal?: boolean;
    isQueriesValue?: boolean;
    tooltip: string;
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

    return (
        <div class={cn(s.statRow, s[props.rowTheme])}>
            <Dropdown
                trigger="hover"
                position="bottomLeft"
                noIcon
                disableAnimation
                overlayClass={s.queryTooltipOverlay}
                menu={<div class={s.statTooltip}>{props.tooltip}</div>}
            >
                <div class={cn(theme.text.t3, theme.text.condenced, s.statRowLeft)}>
                    <Icon icon={props.icon} class={s.tableRowIcon} />
                    {props.label}
                </div>
            </Dropdown>

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
                        <Dropdown
                            trigger="hover"
                            position="top"
                            noIcon
                            disableAnimation
                            overlayClass={s.queryTooltipOverlay}
                            menu={
                                <div class={s.queryTooltip}>
                                    {typeof props.value === 'number'
                                        ? formatNumber(props.value)
                                        : props.value}{' '}
                                    {intl.getMessage('queries').toLowerCase()}
                                </div>
                            }
                        >
                            <div class={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                                {typeof props.value === 'number'
                                    ? formatCompactNumber(props.value)
                                    : props.value}

                                <div
                                    class={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}
                                >
                                    <Show
                                        when={props.isTotal}
                                        fallback={
                                            <Show
                                                when={
                                                    props.percent !== undefined &&
                                                    props.percent! > 0
                                                }
                                            >
                                                <span>({props.percent!.toFixed(1)}%)</span>
                                            </Show>
                                        }
                                    >
                                        <span>({intl.getMessage('total')})</span>
                                    </Show>
                                </div>
                            </div>
                        </Dropdown>
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
