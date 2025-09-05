import React, { forwardRef } from 'react';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';
import s from './Radio.module.pcss';

type Props<T> = {
    className?: string;
    wrapClass?: string;
    disabled?: boolean;
    handleChange: (e: T) => void;
    value: T;
    options: { text: string; value: T; description?: React.ReactNode }[];
    name?: string;
    textClassName?: string;
    verticalAlign?: 'center' | 'start' | 'end';
};

export const Radio = forwardRef<HTMLDivElement, Props<any>>(
    <T extends number | string | boolean = string>(
        { className, wrapClass, disabled, handleChange, value, options, name, textClassName, verticalAlign }: Props<T>,
        ref: React.Ref<HTMLDivElement>,
    ) => (
        <div ref={ref} className={cn(s.wrap, wrapClass)}>
            {options.map((o) => (
                <label
                    key={`${o.value}`}
                    htmlFor={name ? `${name}-${o.value}` : String(o.value)}
                    className={cn(s.radio, className, s[verticalAlign])}>
                    <input
                        id={name ? `${name}-${o.value}` : String(o.value)}
                        type="radio"
                        className={s.input}
                        name={name}
                        onChange={() => handleChange(o.value)}
                        checked={value === o.value}
                        disabled={disabled}
                    />
                    <div className={s.handler}>
                        <Icon
                            icon={value === o.value ? 'radio_on' : 'radio_off'}
                            className={cn(s.icon, { [s.active]: value === o.value })}
                        />
                    </div>
                    <div className={cn(s.text, textClassName)}>
                        <div>{o.text}</div>
                        {o.description && <div className={cn(theme.text.t4, s.description)}>{o.description}</div>}
                    </div>
                </label>
            ))}
        </div>
    ),
);

Radio.displayName = 'Radio';
