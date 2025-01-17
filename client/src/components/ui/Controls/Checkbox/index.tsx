import React, { ReactNode } from 'react';
import clsx from 'clsx';

import './checkbox.css';

type Props = {
    title: string;
    subtitle?: ReactNode;
    value: boolean;
    name?: string;
    disabled?: boolean;
    className?: string;
    onChange: (value: boolean) => void;
};

export const Checkbox = ({ title, subtitle, value, name, disabled, className = 'checkbox--form', onChange }: Props) => (
    <label className={clsx('checkbox', className)}>
        <span className="checkbox__marker" />
        <input
            name={name}
            type="checkbox"
            className="checkbox__input"
            disabled={disabled}
            checked={value}
            onChange={(e) => onChange(e.target.checked)}
        />
        <span className="checkbox__label">
            <span className="checkbox__label-text checkbox__label-text--long">
                <span className="checkbox__label-title">{title}</span>

                {subtitle && <span className="checkbox__label-subtitle">{subtitle}</span>}
            </span>
        </span>
    </label>
);
