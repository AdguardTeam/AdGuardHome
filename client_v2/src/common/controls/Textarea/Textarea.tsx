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
    disabled,
}: Props) => (
    <div className={s.textareaWrapper}>
        {label && (
            <label className={s.label} htmlFor={id}>
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
            disabled={disabled}
        />
        {errorMessage && <div className={s.errorMessage}>{errorMessage}</div>}
    </div>
);
