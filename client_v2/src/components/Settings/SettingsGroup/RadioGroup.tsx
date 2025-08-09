import React, { ReactNode } from 'react';
import cn from 'clsx';

import { Radio } from 'panel/common/controls/Radio';

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
                    <div className={s.title}>{title}</div>
                    {description && <div className={s.desc}>{description}</div>}
                </div>
                <div className={s.input} />
            </div>

            <div className={s.content}>
                <Radio<T> disabled={disabled} value={value} options={options} handleChange={onChange} name={name} />
                {children}
            </div>
        </div>
    );
};
