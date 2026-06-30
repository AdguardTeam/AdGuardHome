import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import { captitalizeWords } from 'panel/helpers/helpers';
import { getSourceData } from 'panel/helpers/trackers/trackers';
import theme from 'panel/lib/theme';
import cn from 'clsx';

import s from './TrackerTooltip.module.pcss';

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

export const TrackerTooltip = (props: Props) => {
    const sourceData = () => getSourceData(props.trackerData);

    return (
        <div class={s.tooltip}>
            <div class={cn(theme.text.t3, s.tooltipTitle)}>
                {intl.getMessage('found_in_known_domains')}
            </div>

            <div class={s.tooltipRow}>
                <span class={cn(theme.text.t4, s.tooltipLabel)}>
                    {intl.getMessage('name_tooltip')}:
                </span>
                <a
                    href={props.trackerData.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    class={cn(theme.text.t4, s.tooltipLink)}
                >
                    {props.trackerData.name}
                </a>
            </div>

            <div class={s.tooltipRow}>
                <span class={cn(theme.text.t4, s.tooltipLabel)}>
                    {intl.getMessage('category_tooltip')}:
                </span>

                <span class={cn(theme.text.t4, theme.text.semibold, s.tooltipValue)}>
                    {captitalizeWords(props.trackerData.category)}
                </span>
            </div>

            <Show when={sourceData()}>
                <div class={s.tooltipRow}>
                    <span class={cn(theme.text.t4, s.tooltipLabel)}>
                        {intl.getMessage('source_tooltip')}:
                    </span>
                    <a
                        href={sourceData()!.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        class={cn(theme.text.t4, s.tooltipLink)}
                    >
                        {sourceData()!.name}
                    </a>
                </div>
            </Show>
        </div>
    );
};
