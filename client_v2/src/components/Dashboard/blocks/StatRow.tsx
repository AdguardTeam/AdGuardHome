import React from 'react';

import intl from 'panel/common/intl';
import { Icon, IconType } from 'panel/common/ui/Icon';
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
    tooltip: string,
    rowTheme: 'dnsQueries' | 'adsBlocked' | 'threatsBlocked' | 'adultWebsitesBlocked' | 'safeSearchUsed' | 'averageProcessingTime';
};

export const StatRow = ({
    icon,
    label,
    value,
    percent,
    isTotal,
    isQueriesValue = true,
    tooltip,
    rowTheme }: StatRowProps) => (
    <div className={cn(s.statRow, s[rowTheme])}>
        <Dropdown
            trigger="hover"
            position="bottomLeft"
            noIcon
            disableAnimation
            overlayClassName={s.queryTooltipOverlay}
            menu={<div className={s.statTooltip}>{tooltip}</div>}
        >
            <div className={cn(theme.text.t3, theme.text.condenced, s.statRowLeft)}>
                {<Icon icon={icon} className={s.tableRowIcon} />}

                {label}
            </div>
        </Dropdown>

        <div className={s.statRowValue}>
            {isQueriesValue ? (
                <Dropdown
                    trigger="hover"
                    position="top"
                    noIcon
                    disableAnimation
                    overlayClassName={s.queryTooltipOverlay}
                    menu={
                        <div className={s.queryTooltip}>
                            {typeof value === 'number' ? formatNumber(value) : value} {intl.getMessage('queries').toLowerCase()}
                        </div>
                    }
                >
                    <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                        {typeof value === 'number' ? formatCompactNumber(value) : value}

                        <div className={cn(theme.text.t3, theme.text.condenced, s.queryPercent)}>
                            {isTotal ? (
                                <span>({intl.getMessage('total')})</span>
                            ) : percent !== undefined && percent > 0 && (
                                <span>({percent.toFixed(1)}%)</span>
                            )}
                        </div>
                    </div>
                </Dropdown>
            ) : (
                <div className={cn(theme.text.t3, theme.text.condenced, s.queryCount)}>
                    {value}
                </div>
            )}

            {isQueriesValue && (
                <div className={s.queryBar}>
                    <div
                        className={s.queryBarFill}
                        style={{ width: `${isTotal ? 100 : (percent || 0)}%` }}
                    />
                </div>
            )}
        </div>
    </div>
);
