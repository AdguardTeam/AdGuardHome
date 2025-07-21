import React, { forwardRef, ReactNode } from 'react';

type Props<T> = {
    name: string;
    value: T;
    onChange: (e: T) => void;
    options: { label: string; desc?: ReactNode; value: T }[];
    disabled?: boolean;
    error?: string;
};

export const Radio = forwardRef<HTMLInputElement, Props<string | boolean | number | undefined>>(
    ({ disabled, onChange, value, options, name, error, ...rest }, ref) => {
        const getId = (label: string) => (name ? `${label}_${name}` : label);

        return (
            <div>
                {options.map((o) => {
                    const checked = value === o.value;

                    return (
                        <label
                            key={`${getId(o.label)}`}
                            htmlFor={getId(o.label)}
                            className="custom-control custom-radio">
                            <input
                                id={getId(o.label)}
                                data-testid={o.value}
                                type="radio"
                                className="custom-control-input"
                                onChange={() => onChange(o.value)}
                                checked={checked}
                                disabled={disabled}
                                ref={ref}
                                {...rest}
                            />

                            <span className="custom-control-label">{o.label}</span>

                            {o.desc && <span className="checkbox__label-subtitle">{o.desc}</span>}
                        </label>
                    );
                })}
                {!disabled && error && <span className="form__message form__message--error">{error}</span>}
            </div>
        );
    },
);

Radio.displayName = 'Radio';
