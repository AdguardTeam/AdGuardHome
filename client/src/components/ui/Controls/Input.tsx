import React, { ComponentProps, forwardRef, ReactNode } from 'react';
import clsx from 'clsx';

type Props = ComponentProps<'input'> & {
    label?: string;
    leftAddon?: ReactNode;
    rightAddon?: ReactNode;
    error?: string;
};

export const Input = forwardRef<HTMLInputElement, Props>(
    ({ name, label, className, leftAddon, rightAddon, error, ...rest }, ref) => (
        <div className={clsx('form-group', { 'has-error': !!error })}>
            {label && (
                <label className="form__label" htmlFor={name}>
                    {label}
                </label>
            )}
            <div className="input-group">
                {leftAddon && <div>{leftAddon}</div>}
                <input className={clsx('form-control', { 'is-invalid': !!error }, className)} ref={ref} {...rest} />
                {rightAddon && <div>{rightAddon}</div>}
            </div>
            {error && <div className="form__message form__message--error mt-1">{error}</div>}
        </div>
    ),
);

Input.displayName = 'Input';
