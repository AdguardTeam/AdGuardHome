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
    domain: string;
    trackerData?: TrackerData;
};

export const TrackerTooltip = (props: Props) => {
    const sourceData = () => (props.trackerData ? getSourceData(props.trackerData) : null);

    return (
        <div class={s.tooltip}>
            <Show when={props.trackerData}>
                {(tracker) => (
                    <>
                        <div class={cn(theme.text.t2, theme.text.semibold, s.tooltipTitle)}>
                            {intl.getMessage('found_in_known_domains')}
                        </div>

                        <div class={s.tooltipRow}>
                            <span class={cn(theme.text.t3, theme.text.semibold, s.tooltipLabel)}>
                                {intl.getMessage('name_tooltip')}:
                            </span>
                            <a
                                href={tracker().url}
                                target="_blank"
                                rel="noopener noreferrer"
                                class={cn(theme.text.t3, s.tooltipLink)}
                            >
                                {tracker().name}
                            </a>
                        </div>

                        <div class={s.tooltipRow}>
                            <span class={cn(theme.text.t3, theme.text.semibold, s.tooltipLabel)}>
                                {intl.getMessage('category_tooltip')}:
                            </span>

                            <span class={cn(theme.text.t3, s.tooltipValue)}>
                                {captitalizeWords(tracker().category)}
                            </span>
                        </div>

                        <Show when={sourceData()}>
                            {(source) => (
                                <div class={s.tooltipRow}>
                                    <span
                                        class={cn(
                                            theme.text.t3,
                                            theme.text.semibold,
                                            s.tooltipLabel,
                                        )}
                                    >
                                        {intl.getMessage('source_tooltip')}:
                                    </span>
                                    <a
                                        href={source().url}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        class={cn(theme.text.t3, s.tooltipLink)}
                                    >
                                        {source().name}
                                    </a>
                                </div>
                            )}
                        </Show>
                    </>
                )}
            </Show>

            <div class={cn(s.tooltipRow, s.tooltipDomainRow)}>
                <span class={cn(theme.text.t3, theme.text.semibold, s.tooltipLabel)}>
                    {intl.getMessage('domain')}:
                </span>
                <span class={cn(theme.text.t3, s.tooltipValue, s.tooltipDomainValue)}>
                    {props.domain}
                </span>
            </div>
        </div>
    );
};
