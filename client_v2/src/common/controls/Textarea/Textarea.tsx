import { type JSX, Show } from 'solid-js';
import cn from 'clsx';

import s from './styles.module.pcss';

type TextareaChangeEvent = Event & {
    currentTarget: HTMLTextAreaElement;
    target: HTMLTextAreaElement;
};

type Props = Omit<JSX.TextareaHTMLAttributes<HTMLTextAreaElement>, 'onChange' | 'onBlur'> & {
    label?: JSX.Element;
    size?: 'small' | 'medium' | 'large';
    errorMessage?: string;
    ref?: HTMLTextAreaElement | ((el: HTMLTextAreaElement) => void);
    onChange?: (event: TextareaChangeEvent) => void;
    onBlur?: (event: FocusEvent) => void;
};

export const Textarea = (props: Props) => {
    const setRef = (el: HTMLTextAreaElement) => {
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    const handleChange = (e: TextareaChangeEvent) => {
        props.onChange?.(e);
    };

    const handleBlur = (e: FocusEvent) => {
        props.onBlur?.(e);
    };

    return (
        <div class={s.textareaWrapper}>
            <Show when={props.label}>
                <label class={s.label} for={props.id}>
                    {props.label}
                </label>
            </Show>
            <textarea
                class={cn(
                    s.textarea,
                    props.size && s[props.size],
                    { [s.error]: !!props.errorMessage },
                    props.class,
                )}
                id={props.id}
                name={props.name}
                placeholder={props.placeholder}
                value={props.value as string}
                cols={props.cols}
                rows={props.rows}
                onChange={handleChange}
                onBlur={handleBlur}
                wrap={props.wrap}
                maxLength={props.maxLength}
                disabled={props.disabled}
                ref={(el) => setRef(el)}
            />
            <Show when={props.errorMessage}>
                <div class={s.errorMessage}>{props.errorMessage}</div>
            </Show>
        </div>
    );
};
