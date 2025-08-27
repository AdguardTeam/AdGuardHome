import React, { ComponentProps, ReactNode, forwardRef } from 'react';
import cn from 'clsx';

import s from './styles.module.pcss';

type Props = ComponentProps<'textarea'> & {
    label?: ReactNode;
    size?: 'small' | 'medium' | 'large';
    errorMessage?: string;
};

export const Textarea = forwardRef<HTMLTextAreaElement, Props>(
    (
        {
            id,
            label,
            size,
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
            ...rest
        }: Props,
        ref,
    ) => (
        <div className={s.textareaWrapper}>
            {label && (
                <label className={s.label} htmlFor={id}>
                    {label}
                </label>
            )}
            <textarea
                className={cn(s.textarea, s[size], { [s.error]: !!errorMessage }, className)}
                id={id}
                placeholder={placeholder}
                value={value}
                cols={cols}
                rows={rows}
                onChange={onChange}
                wrap={wrap}
                maxLength={maxLength}
                disabled={disabled}
                ref={ref}
                {...rest}
            />
            {errorMessage && <div className={s.errorMessage}>{errorMessage}</div>}
        </div>
    ),
);

Textarea.displayName = 'Textarea';
