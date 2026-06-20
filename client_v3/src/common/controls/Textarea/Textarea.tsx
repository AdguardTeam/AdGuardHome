import { type JSX, Show } from 'solid-js';
import cn from 'clsx';

import s from './styles.module.pcss';

type Props = JSX.TextareaHTMLAttributes<HTMLTextAreaElement> & {
    label?: JSX.Element;
    size?: 'small' | 'medium' | 'large';
    errorMessage?: string;
    ref?: HTMLTextAreaElement | ((el: HTMLTextAreaElement) => void);
};

export const Textarea = (props: Props) => {
    const setRef = (el: HTMLTextAreaElement) => {
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
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
                placeholder={props.placeholder}
                value={props.value as string}
                cols={props.cols}
                rows={props.rows}
                onChange={(e) => (props.onChange as any)?.(e)}
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
