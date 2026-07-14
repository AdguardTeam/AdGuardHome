import { type JSX, createSignal, createEffect, Show, untrack } from 'solid-js';
import cn from 'clsx';
import { Icon } from 'panel/common/ui/Icon';

import intl from 'panel/common/intl';
import s from './Input.module.pcss';

type InputChangeEvent = Event & {
    currentTarget: HTMLInputElement;
    target: HTMLInputElement;
};

type Props = Omit<JSX.InputHTMLAttributes<HTMLInputElement>, 'size' | 'onChange' | 'onBlur'> & {
    label?: JSX.Element;
    class?: string;
    innerClass?: string;
    prefixIcon?: JSX.Element;
    suffixIcon?: JSX.Element;
    borderless?: boolean;
    invalid?: boolean;
    maxLength?: number;
    error?: boolean;
    errorMessage?: string;
    isClearable?: boolean;
    onClear?: () => void;
    inputError?: string;
    size?: 'small' | 'medium' | 'large';
    value?: string | number | readonly string[];
    defaultValue?: string;
    onChange?: (event: InputChangeEvent) => void;
    onBlur?: (event: FocusEvent) => void;
    ref?: HTMLInputElement | ((el: HTMLInputElement) => void);
    onKeyDown?: (e: KeyboardEvent) => void;
};

const hasInputValue = (value: string | number | readonly string[] | undefined) => {
    if (Array.isArray(value)) {
        return value.length > 0;
    }
    return String(value ?? '').length > 0;
};

export const Input = (props: Props) => {
    let inputRef: HTMLInputElement | undefined;
    const [focused, setFocused] = createSignal(false);
    const [hasValue, setHasValue] = createSignal(
        hasInputValue(untrack(() => props.value) ?? untrack(() => props.defaultValue)),
    );

    createEffect(() => {
        if (props.value !== undefined) {
            setHasValue(hasInputValue(props.value));
        }
    });

    const setInputRef = (el: HTMLInputElement) => {
        inputRef = el;
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    const showClearButton = () =>
        Boolean(props.isClearable && !props.disabled && !props.readOnly && hasValue());
    const hasActions = () => Boolean(props.suffixIcon || showClearButton());

    const handleChange = (event: InputChangeEvent) => {
        setHasValue((event.currentTarget as HTMLInputElement).value.length > 0);
        props.onChange?.(event);
    };

    const handleClear = () => {
        if (!inputRef) {
            props.onClear?.();
            setHasValue(false);
            return;
        }

        inputRef.value = '';
        props.onChange?.({
            target: inputRef,
            currentTarget: inputRef,
        } as unknown as InputChangeEvent);

        setHasValue(false);
        props.onClear?.();
    };

    const handleKeyDown = (e: KeyboardEvent) => {
        // Prevent minus key entry in number inputs to avoid negative values
        if (props.type === 'number' && e.key === '-') {
            e.preventDefault();
        }
        // Forward to any external onKeyDown handler
        props.onKeyDown?.(e);
    };

    const computedErrorMessage = () => props.inputError ?? props.errorMessage;

    return (
        <>
            <Show when={props.label}>
                <label class={s.inputLabel} for={props.id}>
                    {props.label}
                </label>
            </Show>
            <div
                class={cn(
                    s.inputWrapper,
                    {
                        [s.borderless]: props.borderless,
                        [s.small]: props.size === 'small',
                        [s.medium]: props.size === 'medium',
                        [s.large]: props.size === 'large',
                    },
                    props.class,
                    {
                        [s.prefix]: props.prefixIcon,
                        [s.suffix]: hasActions(),
                        [s.invalid]: props.invalid,
                        [s.focused]: focused(),
                        [s.disabled]: props.disabled,
                        [s.error]: props.error || !!computedErrorMessage(),
                    },
                )}
            >
                <Show when={props.prefixIcon}>{props.prefixIcon}</Show>
                <input
                    ref={(el) => setInputRef(el)}
                    accept={props.accept}
                    autofocus={props.autofocus}
                    class={cn(s.input, props.innerClass, {
                        [s.prefix]: props.prefixIcon,
                        [s.postfix]: hasActions(),
                    })}
                    onChange={handleChange}
                    onInput={(e) => (props.onInput as any)?.(e)}
                    onKeyDown={handleKeyDown}
                    type={props.type}
                    id={props.id}
                    name={props.name}
                    placeholder={props.placeholder}
                    value={props.value as string | number}
                    onFocus={() => setFocused(true)}
                    onBlur={(e) => {
                        props.onBlur?.(e);
                        setFocused(false);
                    }}
                    maxLength={props.maxLength}
                    disabled={props.disabled}
                    autocomplete={props.autocomplete}
                />
                <Show when={hasActions()}>
                    <div class={s.actions}>
                        <Show when={showClearButton()}>
                            <button
                                type="button"
                                class={s.clearButton}
                                aria-label={intl.getMessage('aria_clear_input')}
                                data-testid="input-clear-button"
                                onMouseDown={(event) => event.preventDefault()}
                                onClick={handleClear}
                            >
                                <Icon icon="cross" />
                            </button>
                        </Show>
                        <Show when={props.suffixIcon}>
                            <div class={s.suffixIcon}>{props.suffixIcon}</div>
                        </Show>
                    </div>
                </Show>
            </div>
            <Show when={computedErrorMessage()}>
                <div class={s.inputError}>{computedErrorMessage()}</div>
            </Show>
        </>
    );
};
