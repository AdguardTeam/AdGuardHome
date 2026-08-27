import { For, Show, type JSX } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './StatMobileCard.module.pcss';

export type StatCardItem = {
    label: string;
    value: JSX.Element;
};

type Props = {
    items: StatCardItem[];
    actions?: JSX.Element;
};

export const StatMobileCard = (props: Props) => (
    <div class={s.card} data-testid="stats-mobile-card">
        <For each={props.items}>
            {(item) => (
                <div class={cn(s.row, theme.text.t3)}>
                    <span class={cn(theme.text.condenced, s.rowLabel)}>{item.label}</span>
                    <span class={s.rowValue}>{item.value}</span>
                </div>
            )}
        </For>
        <Show when={props.actions}>
            <div class={s.actions}>{props.actions}</div>
        </Show>
    </div>
);
