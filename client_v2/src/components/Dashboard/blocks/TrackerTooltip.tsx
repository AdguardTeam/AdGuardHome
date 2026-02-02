import React from 'react';

import intl from 'panel/common/intl';
import { captitalizeWords } from 'panel/helpers/helpers';
import { getSourceData } from 'panel/helpers/trackers/trackers';
import theme from 'panel/lib/theme';
import cn from 'clsx';

import s from '../Dashboard.module.pcss';

export type TrackerData = {
    id: string;
    name: string;
    url: string;
    category: string;
    source: number;
    sourceData: { name: string; url: string } | null;
};

type Props = {
    trackerData: TrackerData;
};

export const TrackerTooltip = ({ trackerData }: Props) => {
    const sourceData = getSourceData(trackerData);

    return (
        <div className={s.tooltip}>
            <div className={cn(theme.text.t3, s.tooltipTitle)}>
                {intl.getMessage('found_in_known_domains')}
            </div>

            <div className={s.tooltipRow}>
                <span className={cn(theme.text.t4, s.tooltipLabel)}>{intl.getMessage('name_tooltip')}:</span>
                <a
                    href={trackerData.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className={cn(theme.text.t4, s.tooltipLink)}
                >
                    {trackerData.name}
                </a>
            </div>

            <div className={s.tooltipRow}>
                <span className={cn(theme.text.t4, s.tooltipLabel)}>
                    {intl.getMessage('category_tooltip')}:
                </span>

                <span className={cn(theme.text.t4, theme.text.semibold, s.tooltipValue)}>
                    {captitalizeWords(trackerData.category)}
                </span>
            </div>

            {sourceData && (
                <div className={s.tooltipRow}>
                    <span className={cn(theme.text.t4, s.tooltipLabel)}>
                        {intl.getMessage('source_tooltip')}:
                    </span>

                    <a
                        href={sourceData.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className={cn(theme.text.t4, s.tooltipLink)}
                    >
                        {sourceData.name}
                    </a>
                </div>
            )}
        </div>
    );
};
