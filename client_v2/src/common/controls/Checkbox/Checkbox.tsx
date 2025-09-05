import React, { ComponentProps, MouseEvent, ReactNode } from 'react';
import cn from 'clsx';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './Checkbox.module.pcss';

type Props = ComponentProps<'input'> & {
    className?: string;
    labelClassName?: string;
    overflow?: boolean;
    onClick?: (e: MouseEvent<HTMLElement>) => void;
    children?: ReactNode;
    plusStyle?: boolean;
    verticalAlign?: 'center' | 'start' | 'end';
};

export const Checkbox = ({
    checked,
    children,
    className,
    labelClassName,
    disabled,
    onChange,
    id,
    name,
    overflow,
    onClick,
    plusStyle,
    verticalAlign,
}: Props) => (
    <label
        htmlFor={id}
        className={cn(s.checkbox, { [s.disabled]: disabled }, s[verticalAlign], className)}
        onClick={onClick}>
        <input
            id={id}
            name={name}
            type="checkbox"
            className={s.input}
            onChange={onChange}
            checked={checked}
            disabled={disabled}
        />
        <div className={s.handler}>
            {plusStyle ? (
                <Icon
                    icon={checked ? 'checkbox_minus' : 'checkbox_plus'}
                    className={cn(s.icon, { [s.active]: checked })}
                />
            ) : (
                <Icon icon={checked ? 'checkbox_on' : 'checkbox_off'} className={cn(s.icon, { [s.active]: checked })} />
            )}
        </div>
        {children && (
            <div className={cn(s.label, { [theme.common.textOverflow]: overflow }, labelClassName)}>{children}</div>
        )}
    </label>
);
