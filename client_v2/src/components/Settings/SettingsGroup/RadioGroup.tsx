import React, { ReactNode } from 'react';
import cn from 'clsx';

import { Radio } from 'panel/common/controls/Radio';
import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

type Option<T> = { text: string; value: T };

type Props<T> = {
    title: string;
    description?: string;
    disabled?: boolean;
    value: T;
    options: Option<T>[];
    onChange: (value: T) => void;
    className?: string;
    children?: ReactNode;
    name?: string;
};

export const RadioGroup = <T extends number | string | boolean>({
    title,
    description,
    disabled,
    value,
    options,
    onChange,
    className,
    children,
    name,
}: Props<T>) => {
    return (
        <div className={cn(s.switch, className)}>
            <div className={s.row}>
                <div className={s.text}>
                    <div className={cn(s.title, theme.text.t2, theme.text.semibold)}>{title}</div>
                    {description && <div className={cn(s.desc, theme.text.t3)}>{description}</div>}
                </div>
                <div className={s.input} />
            </div>

            <div className={s.content}>
                <Radio disabled={disabled} value={value} options={options} handleChange={onChange} name={name} />
                {children}
            </div>
        </div>
    );
};
