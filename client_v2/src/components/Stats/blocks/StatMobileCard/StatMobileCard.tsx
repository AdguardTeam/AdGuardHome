import { For, Show, type JSX } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import { Link } from 'panel/common/ui/Link';
import type { QueryParams, RoutePathKey } from 'panel/components/Routes/Paths';

import s from './StatMobileCard.module.pcss';

type TitleLink = {
    to: RoutePathKey;
    query?: QueryParams;
    title?: string;
};

export type StatCardItem = {
    label: string;
    value: JSX.Element;
    /** 0..100 share; renders a progress bar under this row. */
    progress?: number;
    /** Color variant for the row's progress bar fill. */
    tone?: 'default' | 'danger';
    /** 'row' (default): label left / value right. 'column': label above value. */
    layout?: 'row' | 'column';
};

type Props = {
    /** Card title (client name), displayed bold at the top. */
    title?: JSX.Element;
    /** When set, the title is rendered as a link (keeps its color, underlines on hover). */
    titleLink?: TitleLink;
    /** Status line rendered right under the title, e.g. "Blocked". */
    status?: JSX.Element;
    items: StatCardItem[];
    actions?: JSX.Element;
};

export const StatMobileCard = (props: Props) => (
    <div class={s.card} data-testid="stats-mobile-card">
        <Show when={props.title}>
            <span class={cn(theme.text.t2, s.title)}>
                <Show when={props.titleLink} fallback={props.title}>
                    {(link) => (
                        <Link
                            to={link().to}
                            query={link().query}
                            class={s.titleLink}
                            title={link().title}
                        >
                            {props.title}
                        </Link>
                    )}
                </Show>
            </span>
        </Show>
        <Show when={props.status}>
            <div class={s.row}>{props.status}</div>
        </Show>
        <For each={props.items}>
            {(item) => (
                <>
                    <div class={cn(theme.text.t3, s.row, item.layout === 'column' && s.rowColumn)}>
                        <span class={cn(theme.text.condenced, s.rowLabel)}>{item.label}</span>
                        <span class={cn(s.rowValue, item.layout === 'column' && s.rowValueColumn)}>
                            {item.value}
                        </span>
                    </div>
                    <Show when={typeof item.progress === 'number'}>
                        <div class={s.progress}>
                            <div
                                class={cn(
                                    s.progressFill,
                                    item.tone === 'danger' && s.progressFillDanger,
                                )}
                                style={{ width: `${item.progress}%` }}
                            />
                        </div>
                    </Show>
                </>
            )}
        </For>
        <Show when={props.actions}>
            <div class={s.actions}>{props.actions}</div>
        </Show>
    </div>
);
