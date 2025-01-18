import React, { ComponentProps, forwardRef } from 'react';
import clsx from 'clsx';

interface Props extends ComponentProps<'textarea'> {
    className?: string;
    label?: string;
    error?: string;
}

export const Textarea = forwardRef<HTMLTextAreaElement, Props>(
    ({ name, label, className, error, onClick, ...rest }, ref) => (
        <div className={clsx('form-group', { 'has-error': !!error })}>
            {label && (
                <label className="form__label" htmlFor={name}>
                    {label}
                </label>
            )}
            <textarea
                onClick={onClick}
                className={clsx(
                    'form-control form-control--textarea form-control--textarea-small font-monospace',
                    className,
                )}
                ref={ref}
                {...rest}
            />
            {error && <div className="form__message form__message--error">{error}</div>}
        </div>
    ),
);

Textarea.displayName = 'Textarea';
