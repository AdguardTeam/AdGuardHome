import React, { ComponentProps, forwardRef } from 'react';
import clsx from 'clsx';
import { trimLinesAndRemoveEmpty } from '../../../helpers/helpers';

type Props = ComponentProps<'textarea'> & {
    className?: string;
    wrapperClassName?: string;
    label?: string;
    desc?: string;
    error?: string;
    trimOnBlur?: boolean;
};

export const Textarea = forwardRef<HTMLTextAreaElement, Props>(
    ({ name, label, desc, className, wrapperClassName, error, trimOnBlur, onBlur, ...rest }, ref) => (
        <div className={clsx('form-group', wrapperClassName, { 'has-error': !!error })}>
            {label && (
                <label className={clsx('form__label', { 'form__label--with-desc': !!desc })} htmlFor={name}>
                    {label}
                </label>
            )}
            {desc && <div className="form__desc form__desc--top">{desc}</div>}
            <textarea
                className={clsx(
                    'form-control form-control--textarea form-control--textarea-small font-monospace',
                    className,
                )}
                ref={ref}
                onBlur={(e) => {
                    if (trimOnBlur) {
                        const normalizedValue = trimLinesAndRemoveEmpty(e.target.value);
                        rest.onChange(normalizedValue);
                    }
                    if (onBlur) {
                        onBlur(e);
                    }
                }}
                {...rest}
            />
            {error && <div className="form__message form__message--error">{error}</div>}
        </div>
    ),
);

Textarea.displayName = 'Textarea';
