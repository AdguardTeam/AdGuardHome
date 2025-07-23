import React, { ComponentProps, ReactNode } from 'react';
import cn from 'clsx';

import s from './Button.module.pcss';

export type ButtonProps = ComponentProps<'button'> & {
    size?: 'small' | 'medium' | 'big';
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    children?: ReactNode;
    leftAddon?: ReactNode;
    rightAddon?: ReactNode;
};

export const Button = ({
    id,
    size = 'medium',
    type = 'button',
    variant = 'primary',
    children,
    className,
    onClick,
    disabled,
    leftAddon,
    rightAddon,
}: ButtonProps) => (
    <button
        id={id}
        type={type}
        className={cn(
            s.button,
            s[variant],
            {
                [s.height_s]: size === 'small',
                [s.height_m]: size === 'medium',
                [s.height_l]: size === 'big',
            },
            className,
        )}
        onClick={onClick}
        disabled={disabled}>
        <div className={s.leftAddon}>{leftAddon}</div>
        {children}
        <div className={s.rightAddon}>{rightAddon}</div>
    </button>
);
