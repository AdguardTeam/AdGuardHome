import React, { forwardRef, useEffect, useRef, useState } from 'react';
import type { ComponentProps, ChangeEvent, ReactNode } from 'react';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';

import intl from 'panel/common/intl';
import s from './Input.module.pcss';

type Props = Omit<ComponentProps<'input'>, 'size'> & {
    label?: ReactNode;
    className?: string;
    innerClassName?: string;
    prefixIcon?: ReactNode;
    suffixIcon?: ReactNode;
    borderless?: boolean;
    invalid?: boolean;
    maxLength?: number;
    error?: boolean;
    errorMessage?: string;
    isClearable?: boolean;
    onClear?: () => void;
    inputError?: string;
    size?: 'small' | 'medium' | 'large';
};

const assignInputRef = (ref: React.ForwardedRef<HTMLInputElement>, node: HTMLInputElement | null) => {
    if (typeof ref === 'function') {
        ref(node);
        return;
    }

    if (ref) {
        const mutableRef = ref;
        mutableRef.current = node;
    }
};

const hasInputValue = (value: ComponentProps<'input'>['value'] | ComponentProps<'input'>['defaultValue']) => {
    if (Array.isArray(value)) {
        return value.length > 0;
    }

    return String(value ?? '').length > 0;
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
            isClearable,
            onClear,
            inputError,
            size = 'large',
            autoComplete,
            ...rest
        },
        ref,
    ) => {
        const [focused, setFocused] = useState(false);
        const inputRef = useRef<HTMLInputElement | null>(null);
        const [hasValue, setHasValue] = useState(() => hasInputValue(value ?? rest.defaultValue));

        useEffect(() => {
            if (value !== undefined) {
                setHasValue(hasInputValue(value));
            }
        }, [value]);

        const setInputRef = (node: HTMLInputElement | null) => {
            inputRef.current = node;
            assignInputRef(ref, node);
        };

        const showClearButton = Boolean(isClearable && !disabled && !rest.readOnly && hasValue);
        const hasActions = Boolean(suffixIcon || showClearButton);

        const handleChange: ComponentProps<'input'>['onChange'] = (event) => {
            setHasValue(event.currentTarget.value.length > 0);
            onChange?.(event);
        };

        const handleClear = () => {
            const node = inputRef.current;
            if (!node) {
                onClear?.();
                setHasValue(false);
                return;
            }

            node.value = '';
            onChange?.({
                target: node,
                currentTarget: node,
            } as ChangeEvent<HTMLInputElement>);

            setHasValue(false);
            onClear?.();
        };

        const computedErrorMessage = inputError ?? errorMessage;

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
                            [s.suffix]: hasActions,
                            [s.invalid]: invalid,
                            [s.focused]: focused,
                            [s.disabled]: disabled,
                            [s.error]: error || !!computedErrorMessage,
                        },
                        className,
                    )}
                >
                    {prefixIcon && prefixIcon}
                    <input
                        ref={setInputRef}
                        accept={accept}
                        autoFocus={autoFocus}
                        className={cn(s.input, innerClassName, {
                            [s.prefix]: prefixIcon,
                            [s.postfix]: hasActions,
                        })}
                        onChange={handleChange}
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
                        {...rest}
                    />
                    {hasActions && (
                        <div className={s.actions}>
                            {showClearButton && (
                                <button
                                    type="button"
                                    className={s.clearButton}
                                    aria-label={intl.getMessage('aria_clear_input')}
                                    data-testid="input-clear-button"
                                    onMouseDown={(event) => event.preventDefault()}
                                    onClick={handleClear}
                                >
                                    <Icon icon="cross" />
                                </button>
                            )}
                            {suffixIcon && <div className={s.suffixIcon}>{suffixIcon}</div>}
                        </div>
                    )}
                </div>
                {computedErrorMessage && <div className={s.inputError}>{computedErrorMessage}</div>}
            </>
        );
    },
);

Input.displayName = 'Input';
