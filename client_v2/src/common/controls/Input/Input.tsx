import React, { ComponentProps, forwardRef, type ReactNode, useState } from 'react';
import cn from 'clsx';

import s from './Input.module.pcss';

type Props = Omit<ComponentProps<'input'>, 'size'> & {
    label?: ReactNode;
    className?: string;
    innerClassName?: string;
    prefixIcon?: ReactNode;
    suffixIcon?: ReactNode;
    borderless?: boolean;
    invalid?: boolean;
    maxLength?: number | undefined;
    error?: boolean;
    errorMessage?: string;
    size?: 'small' | 'medium' | 'large';
};

export const Input = forwardRef<HTMLInputElement, Props>(
    (
        {
            id,
            accept,
            label,
            placeholder,
            type,
            onChange,
            onBlur,
            value,
            className,
            innerClassName,
            prefixIcon,
            suffixIcon,
            borderless,
            invalid,
            autoFocus,
            maxLength,
            disabled,
            error,
            errorMessage,
            size = 'large',
            autoComplete,
        },
        ref,
    ) => {
        const [focused, setFocused] = useState(false);

        const inputWrapperClass = cn(
            s.inputWrapper,
            {
                [s.borderless]: borderless,
                [s.small]: size === 'small',
                [s.medium]: size === 'medium',
                [s.large]: size === 'large',
            },
            className,
        );

        return (
            <>
                {label && (
                    <label className={s.inputLabel} htmlFor={id}>
                        {label}
                    </label>
                )}
                <div
                    className={cn(
                        inputWrapperClass,
                        {
                            [s.prefix]: prefixIcon,
                            [s.suffix]: suffixIcon,
                            [s.invalid]: invalid,
                            [s.focused]: focused,
                            [s.disabled]: disabled,
                            [s.error]: error || !!errorMessage,
                        },
                        className,
                    )}>
                    {prefixIcon && prefixIcon}
                    <input
                        ref={ref}
                        accept={accept}
                        autoFocus={autoFocus}
                        className={cn(s.input, innerClassName, {
                            [s.prefix]: prefixIcon,
                            [s.postfix]: suffixIcon,
                        })}
                        onChange={onChange}
                        type={type}
                        id={id}
                        placeholder={placeholder}
                        value={value}
                        onFocus={() => setFocused(true)}
                        onBlur={(e) => {
                            if (onBlur) {
                                onBlur(e);
                            }

                            setFocused(false);
                        }}
                        maxLength={maxLength}
                        disabled={disabled}
                        autoComplete={autoComplete}
                    />
                    {suffixIcon && suffixIcon}
                </div>
                {errorMessage && <div className={s.inputError}>{errorMessage}</div>}
            </>
        );
    },
);

Input.displayName = 'Input';
