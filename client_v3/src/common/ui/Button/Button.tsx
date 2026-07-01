import { type JSX, splitProps, Show } from 'solid-js';
import cn from 'clsx';

import s from './Button.module.pcss';

export type ButtonProps = JSX.ButtonHTMLAttributes<HTMLButtonElement> & {
    size?: 'very-small' | 'small' | 'medium' | 'big';
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'secondary-danger';
    leftAddon?: JSX.Element;
    rightAddon?: JSX.Element;
    compact?: boolean;
    className?: string;
};

export const Button = (props: ButtonProps) => {
    const [local, rest] = splitProps(props, [
        'variant',
        'size',
        'class',
        'className',
        'children',
        'disabled',
        'leftAddon',
        'rightAddon',
        'compact',
    ]);

    return (
        <button
            {...rest}
            type={props.type || 'button'}
            disabled={local.disabled}
            class={cn(
                s.button,
                s[local.variant || 'primary'],
                {
                    [s.height_xs]: local.size === 'very-small',
                    [s.height_s]: local.size === 'small',
                    [s.height_m]: local.size === 'medium',
                    [s.height_l]: local.size === 'big',
                },
                local.class,
                local.className,
            )}
        >
            <Show when={local.leftAddon || !local.compact}>
                <div class={s.leftAddon}>{local.leftAddon}</div>
            </Show>
            {local.children}
            <Show when={local.rightAddon || !local.compact}>
                <div class={s.rightAddon}>{local.rightAddon}</div>
            </Show>
        </button>
    );
};
