import { type JSX, Show, For } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import s from './Radio.module.pcss';

type Props<T = string | number | boolean> = {
    class?: string;
    wrapClass?: string;
    disabled?: boolean;
    handleChange: (e: T) => void;
    value: T;
    options: { text: string; value: T; description?: JSX.Element; disabled?: boolean }[];
    name?: string;
    textClass?: string;
    verticalAlign?: 'center' | 'start' | 'end';
    ref?: HTMLDivElement | ((el: HTMLDivElement) => void);
    inModal?: boolean;
};

export const Radio = <T extends number | string | boolean = string>(props: Props<T>) => {
    const setRef = (el: HTMLDivElement) => {
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    return (
        <div ref={(el) => setRef(el)} class={cn(s.wrap, props.wrapClass)}>
            <For each={props.options}>
                {(o) => (
                    <label
                        for={props.name ? `${props.name}-${o.value}` : String(o.value)}
                        class={cn(
                            s.radio,
                            props.class,
                            props.verticalAlign && s[props.verticalAlign],
                            {
                                [s.modal]: props.inModal,
                            },
                        )}
                    >
                        <input
                            id={props.name ? `${props.name}-${o.value}` : String(o.value)}
                            type="radio"
                            class={s.input}
                            name={props.name}
                            onChange={() => props.handleChange(o.value)}
                            checked={props.value === o.value}
                            disabled={props.disabled || o.disabled}
                        />
                        <div class={s.handler}>
                            <Icon
                                icon={(props.value === o.value ? 'radio_on' : 'radio_off') as any}
                                class={cn(s.icon, { [s.active]: props.value === o.value })}
                            />
                        </div>
                        <div class={cn(s.text, { [s.disabled]: o.disabled }, props.textClass)}>
                            <div>{o.text}</div>
                            <Show when={o.description}>
                                <div class={cn(theme.text.t4, s.description)}>{o.description}</div>
                            </Show>
                        </div>
                    </label>
                )}
            </For>
        </div>
    );
};
