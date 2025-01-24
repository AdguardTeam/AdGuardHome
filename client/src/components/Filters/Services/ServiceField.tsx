import React from 'react';
import cn from 'classnames';
import { FieldValues, ControllerRenderProps } from 'react-hook-form';

type Props = ControllerRenderProps<FieldValues> & {
    placeholder: string;
    disabled?: boolean;
    className?: string;
    icon?: string;
    error?: string;
};

export const ServiceField = React.forwardRef<HTMLInputElement, Props>(
    ({ name, value, onChange, onBlur, placeholder, disabled, className, icon, error, ...rest }, ref) => (
        <>
            <label className={cn('service custom-switch', className)}>
                <input
                    name={name}
                    type="checkbox"
                    className="custom-switch-input"
                    checked={!!value}
                    onChange={onChange}
                    onBlur={onBlur}
                    ref={ref}
                    disabled={disabled}
                    {...rest}
                />

                <span className="service__switch custom-switch-indicator"></span>

                <span className="service__text" title={placeholder}>
                    {placeholder}
                </span>
                {icon && <div dangerouslySetInnerHTML={{ __html: window.atob(icon) }} className="service__icon" />}
            </label>

            {!disabled && error && <span className="form__message form__message--error">{error}</span>}
        </>
    ),
);

ServiceField.displayName = 'ServiceField';
