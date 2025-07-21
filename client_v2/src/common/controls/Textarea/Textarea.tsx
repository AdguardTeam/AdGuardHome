import React, { ComponentProps } from 'react';
import cn from 'clsx';

import s from './styles.module.pcss';

type Props = ComponentProps<'textarea'> & {
    label?: string;
    errorMessage?: string;
};

export const Textarea = ({
    id,
    label,
    placeholder,
    onChange,
    value,
    rows,
    cols,
    wrap,
    className,
    maxLength,
    errorMessage,
}: Props) => (
    <div className={s.textareaWrapper}>
        {label && (
            <label className={s.textareaLabel} htmlFor={id}>
                {label}
            </label>
        )}
        <textarea
            className={cn(s.textarea, { [s.error]: !!errorMessage }, className)}
            id={id}
            placeholder={placeholder}
            value={value}
            cols={cols}
            rows={rows}
            onChange={onChange}
            wrap={wrap}
            maxLength={maxLength}
        />
        {errorMessage && <div className={s.error}>{errorMessage}</div>}
    </div>
);
