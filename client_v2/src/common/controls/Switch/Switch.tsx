import { type JSX, Show } from 'solid-js';
import cn from 'clsx';

import s from './Switch.module.pcss';

type Props = Omit<JSX.InputHTMLAttributes<HTMLInputElement>, 'onChange' | 'type'> & {
    id: string;
    checked: boolean;
    disabled?: boolean;
    labelClass?: string;
    class?: string;
    wrapperClass?: string;
    onChange: (e: Event) => void;
    children?: JSX.Element;
    ref?: HTMLInputElement | ((el: HTMLInputElement) => void);
};

export const Switch = (props: Props) => {
    const setRef = (el: HTMLInputElement) => {
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    const switchControls = (
        <>
            <input
                id={props.id}
                type="checkbox"
                class={s.input}
                onChange={(e) => props.onChange?.(e)}
                checked={props.checked}
                disabled={props.disabled}
                ref={(el) => setRef(el)}
            />
            <div class={s.handler} />
            <Show when={props.children}>
                <div class={cn(s.label, props.labelClass)}>{props.children}</div>
            </Show>
        </>
    );

    return (
        <label for={props.id} class={cn(s.switch, props.class, { [s.disabled]: props.disabled })}>
            <Show when={props.wrapperClass} fallback={switchControls}>
                <div class={props.wrapperClass}>{switchControls}</div>
            </Show>
        </label>
    );
};
