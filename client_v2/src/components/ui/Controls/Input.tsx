import React, { ComponentProps, forwardRef, ReactNode } from 'react';
import clsx from 'clsx';

type Props = ComponentProps<'input'> & {
    label?: string;
    desc?: string;
    leftAddon?: ReactNode;
    rightAddon?: ReactNode;
    error?: string;
    trimOnBlur?: boolean;
};

export const Input = forwardRef<HTMLInputElement, Props>(
    ({ name, label, desc, className, leftAddon, rightAddon, error, trimOnBlur, onBlur, ...rest }, ref) => (
        <div className={clsx('form-group', { 'has-error': !!error })}>
            {label && (
                <label className={clsx('form__label', { 'form__label--with-desc': !!desc })} htmlFor={name}>
                    {label}
                </label>
            )}
            {desc && <div className="form__desc form__desc--top">{desc}</div>}
            <div className="input-group">
                {leftAddon && <div>{leftAddon}</div>}
                <input
                    className={clsx('form-control', { 'is-invalid': !!error }, className)}
                    ref={ref}
                    onBlur={(e) => {
                        if (trimOnBlur) {
                            e.target.value = e.target.value.trim();
                            rest.onChange(e);
                        }
                        if (onBlur) {
                            onBlur(e);
                        }
                    }}
                    {...rest}
                />
                {rightAddon && <div>{rightAddon}</div>}
            </div>
            {error && <div className="form__message form__message--error mt-1">{error}</div>}
        </div>
    ),
);

Input.displayName = 'Input';
