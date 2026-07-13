import { type JSX, Show } from 'solid-js';
import cn from 'clsx';

import { Radio } from 'panel/common/controls/Radio';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type Option<T> = { text: string; value: T };

type Props<T = string | number | boolean> = {
    title?: string;
    description?: string;
    disabled?: boolean;
    value: T;
    options: Option<T>[];
    onChange: (value: T) => void;
    class?: string;
    children?: JSX.Element;
    name?: string;
};

export const RadioGroup = <T extends number | string | boolean>(props: Props<T>) => {
    return (
        <div class={cn(s.radio, props.class)}>
            <Show when={props.title}>
                <div class={s.row}>
                    <div class={s.text}>
                        <div class={cn(s.title, theme.text.t2, theme.text.semibold)}>
                            {props.title}
                        </div>
                        <Show when={props.description}>
                            <div class={cn(s.desc, theme.text.t3)}>{props.description}</div>
                        </Show>
                    </div>
                    <div class={s.input} />
                </div>
            </Show>

            <div class={s.content}>
                <Radio
                    disabled={props.disabled}
                    value={props.value as any}
                    options={props.options as any}
                    handleChange={props.onChange as any}
                    name={props.name}
                />
                {props.children}
            </div>
        </div>
    );
};
