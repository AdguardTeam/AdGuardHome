import { type JSX, splitProps } from 'solid-js';
import cn from 'clsx';

import s from './Button.module.pcss';

export type ButtonProps = JSX.ButtonHTMLAttributes<HTMLButtonElement> & {
    size?: 'small' | 'medium' | 'big';
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'secondary-danger';
    leftAddon?: JSX.Element;
    rightAddon?: JSX.Element;
    className?: string;
};

export const Button = (props: ButtonProps) => {
    const [local, rest] = splitProps(props, [
        'id',
        'size',
        'type',
        'variant',
        'children',
        'class',
        'className',
        'onClick',
        'disabled',
        'leftAddon',
        'rightAddon',
    ]);

    return (
        <button
            id={local.id}
            type={local.type || 'button'}
            class={cn(
                s.button,
                s[local.variant || 'primary'],
                {
                    [s.height_s]: (local.size || 'medium') === 'small',
                    [s.height_m]: (local.size || 'medium') === 'medium',
                    [s.height_l]: (local.size || 'medium') === 'big',
                },
                local.class,
                local.className,
            )}
            onClick={local.onClick}
            disabled={local.disabled}
            {...rest}
        >
            <div class={s.leftAddon}>{local.leftAddon}</div>
            {local.children}
            <div class={s.rightAddon}>{local.rightAddon}</div>
        </button>
    );
};
