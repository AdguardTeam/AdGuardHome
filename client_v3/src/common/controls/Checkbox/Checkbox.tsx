import { type JSX, Show } from 'solid-js';
import cn from 'clsx';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './Checkbox.module.pcss';

type Props = JSX.InputHTMLAttributes<HTMLInputElement> & {
    class?: string;
    labelClass?: string;
    overflow?: boolean;
    onClick?: (e: MouseEvent) => void;
    children?: JSX.Element;
    plusStyle?: boolean;
    verticalAlign?: 'center' | 'start' | 'end';
    ref?: HTMLInputElement | ((el: HTMLInputElement) => void);
};

export const Checkbox = (props: Props) => {
    const setRef = (el: HTMLInputElement) => {
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    return (
        <label
            for={props.id}
            class={cn(
                s.checkbox,
                { [s.disabled]: props.disabled },
                props.verticalAlign && s[props.verticalAlign],
                props.class,
            )}
            onClick={(e) => (props.onClick as any)?.(e)}
        >
            <input
                ref={(el) => setRef(el)}
                id={props.id}
                name={props.name}
                type="checkbox"
                class={s.input}
                onChange={(e) => (props.onChange as any)?.(e)}
                checked={props.checked}
                disabled={props.disabled}
            />
            <div class={s.handler}>
                <Show
                    when={props.plusStyle}
                    fallback={
                        <Icon
                            icon={(props.checked ? 'checkbox_on' : 'checkbox_off') as any}
                            class={cn(s.icon, { [s.active]: props.checked })}
                        />
                    }
                >
                    <Icon
                        icon={(props.checked ? 'checkbox_minus' : 'checkbox_plus') as any}
                        class={cn(s.icon, { [s.active]: props.checked })}
                    />
                </Show>
            </div>
            <Show when={props.children}>
                <div
                    class={cn(
                        s.label,
                        { [theme.common.textOverflow]: props.overflow },
                        props.labelClass,
                    )}
                >
                    {props.children}
                </div>
            </Show>
        </label>
    );
};
