import React from 'react';
import cn from 'clsx';

import s from './Radio.module.pcss';
import { Icon } from 'panel/common/ui/Icon';

type Props<T> = {
    className?: string;
    wrapClass?: string;
    disabled?: boolean;
    handleChange: (e: T) => void;
    value: T;
    options: { text: string; value: T }[];
};

export const Radio = <T extends number | string | boolean = string>({
    className,
    wrapClass,
    disabled,
    handleChange,
    value,
    options,
}: Props<T>) => (
    <div className={cn(s.wrap, wrapClass)}>
        {options.map((o) => (
            <label key={`${o.value}`} htmlFor={String(o.value)} className={cn(s.radio, className)}>
                <input
                    id={String(o.value)}
                    type="radio"
                    className={s.input}
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
                <div className={s.text}>{o.text}</div>
            </label>
        ))}
    </div>
);
