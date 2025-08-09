import React, { forwardRef, ReactNode } from 'react';
import clsx from 'clsx';

import './checkbox.css';

type Props = {
    title: string;
    subtitle?: ReactNode;
    value: boolean;
    name?: string;
    disabled?: boolean;
    className?: string;
    error?: string;
    onChange: (value: boolean) => void;
    onBlur?: () => void;
};

export const Checkbox = forwardRef<HTMLInputElement, Props>(
    (
        { title, subtitle, value, name, disabled, error, className = 'checkbox--form', onChange, onBlur, ...rest },
        ref,
    ) => (
        <>
            <label className={clsx('checkbox', className)}>
                <span className="checkbox__marker" />
                <input
                    name={name}
                    type="checkbox"
                    className="checkbox__input"
                    disabled={disabled}
                    checked={value}
                    onChange={(e) => onChange(e.target.checked)}
                    onBlur={onBlur}
                    ref={ref}
                    {...rest}
                />
                <span className="checkbox__label">
                    <span className="checkbox__label-text checkbox__label-text--long">
                        <span className="checkbox__label-title">{title}</span>

                        {subtitle && <span className="checkbox__label-subtitle">{subtitle}</span>}
                    </span>
                </span>
            </label>
            {error && <div className="form__message form__message--error">{error}</div>}
        </>
    ),
);

Checkbox.displayName = 'Checkbox';
