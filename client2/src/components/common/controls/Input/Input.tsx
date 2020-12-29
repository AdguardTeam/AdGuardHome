import React, { FC, FocusEvent, KeyboardEvent, ClipboardEvent, ChangeEvent, useState } from 'react';
import { Input as InputControl } from 'antd';
import { InputProps as InputControlProps } from 'antd/lib/input';
import cn from 'classnames';

import { Icon } from 'Common/ui';
import theme from 'Lib/theme';

interface AdminInterfaceProps {
    autoComplete?: InputControlProps['autoComplete'];
    autoFocus?: InputControlProps['autoFocus'];
    className?: string;
    description?: string;
    disabled?: boolean;
    error?: boolean;
    id?: string;
    inputMode?: InputControlProps['inputMode'];
    label?: string;
    wrapperClassName?: string;
    name: string;
    onBlur?: (e: FocusEvent<HTMLInputElement>) => void;
    onChange?: (data: string, e?: ChangeEvent<HTMLInputElement>) => void;
    onFocus?: (e: FocusEvent<HTMLInputElement>) => void;
    onKeyDown?: (e: KeyboardEvent<HTMLInputElement>) => void;
    onPaste?: (e: ClipboardEvent<HTMLInputElement>) => void;
    pattern?: InputControlProps['pattern'];
    placeholder: string;
    prefix?: InputControlProps['prefix'];
    size?: InputControlProps['size'];
    suffix?: InputControlProps['suffix'];
    type: InputControlProps['type'];
    value: string | number;
}

const InputComponent: FC<AdminInterfaceProps> = ({
    autoComplete,
    autoFocus,
    className,
    description,
    disabled,
    error,
    id,
    inputMode,
    label,
    wrapperClassName,
    name,
    onBlur,
    onChange,
    onFocus,
    onKeyDown,
    onPaste,
    pattern,
    placeholder,
    prefix,
    size = 'middle',
    suffix,
    type,
    value,
}) => {
    const [inputType, setInputType] = useState(type);

    const inputClass = cn(
        'input',
        { input_big: size === 'large' },
        { input_medium: size === 'middle' },
        { input_small: size === 'small' },
        className,
    );

    const handleBlur = (e: FocusEvent<HTMLInputElement>) => {
        if (onBlur) {
            onBlur(e);
        }
    };

    const showPassword = () => {
        if (inputType === 'password') {
            setInputType('text');
        } else {
            setInputType('password');
        }
    };

    const showPasswordIcon = () => {
        const icon = inputType === 'password' ? 'visibility_disable' : 'visibility_enable';
        return (
            <Icon
                icon={icon}
                className={theme.form.reveal}
                onClick={showPassword}
            />
        );
    };

    const validSuffix = (
        <>
            {!!suffix && suffix}
            {(type === 'password') && showPasswordIcon()}
        </>
    );

    let descriptionView = null;
    if (description) {
        descriptionView = (
            <div className={theme.form.label}>
                {description}
            </div>
        );
    }

    return (
        <label htmlFor={id || name} className={cn(theme.form.group, wrapperClassName)}>
            {label && (
                <div className={theme.form.label}>
                    {label}
                </div>
            )}
            <InputControl
                autoComplete={autoComplete}
                autoFocus={autoFocus}
                className={inputClass}
                disabled={disabled}
                formNoValidate
                id={id || name}
                inputMode={inputMode}
                name={name}
                onBlur={handleBlur}
                onChange={(e) => onChange && onChange(e.target.value ? e.target.value : '', e)}
                onFocus={onFocus}
                onKeyDown={onKeyDown}
                onPaste={onPaste}
                pattern={pattern}
                placeholder={placeholder}
                prefix={prefix}
                size="large"
                suffix={validSuffix}
                type={inputType}
                value={value}
                data-error={error}
            />
            {descriptionView}
        </label>
    );
};

export default InputComponent;
