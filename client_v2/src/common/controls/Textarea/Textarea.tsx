import { type JSX, Show, createSignal, createEffect } from 'solid-js';
import cn from 'clsx';

import s from './styles.module.pcss';
import { TextareaHighlight } from './TextareaHighlight';

import type { CommentLineToken } from 'panel/helpers/constants';

type TextareaChangeEvent = Event & {
    currentTarget: HTMLTextAreaElement;
    target: HTMLTextAreaElement;
};

type Props = Omit<
    JSX.TextareaHTMLAttributes<HTMLTextAreaElement>,
    'onChange' | 'onBlur' | 'onInput' | 'onScroll'
> & {
    label?: JSX.Element;
    size?: 'small' | 'medium' | 'large';
    errorMessage?: string;
    highlightComments?: boolean;
    commentPrefixes?: readonly CommentLineToken[];
    ref?: HTMLTextAreaElement | ((el: HTMLTextAreaElement) => void);
    onChange?: (event: TextareaChangeEvent) => void;
    onInput?: (event: TextareaChangeEvent) => void;
    onBlur?: (event: FocusEvent) => void;
    onScroll?: (
        event: Event & { currentTarget: HTMLTextAreaElement; target: HTMLTextAreaElement },
    ) => void;
};

export const Textarea = (props: Props) => {
    const [scrollTop, setScrollTop] = createSignal(0);
    const [currentValue, setCurrentValue] = createSignal('');

    // Sync from external prop changes (e.g. dialog open/close resets);
    // also handles initial value on mount. Runs before browser paint so
    // there is no visible flicker from the empty initial signal.
    createEffect(() => {
        setCurrentValue(props.value as string);
    });

    const highlightEnabled = () => !!props.highlightComments;

    const setRef = (el: HTMLTextAreaElement) => {
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    const handleChange = (e: TextareaChangeEvent) => {
        props.onChange?.(e);
    };

    const handleInput = (e: TextareaChangeEvent) => {
        setCurrentValue(e.target.value);
        props.onInput?.(e);
    };

    const handleBlur = (e: FocusEvent) => {
        props.onBlur?.(e);
    };

    const handleScroll = (
        e: Event & { currentTarget: HTMLTextAreaElement; target: HTMLTextAreaElement },
    ) => {
        setScrollTop(e.target.scrollTop);
        props.onScroll?.(e);
    };

    return (
        <div class={s.textareaWrapper}>
            <Show when={props.label}>
                <label class={s.label} for={props.id}>
                    {props.label}
                </label>
            </Show>
            <div class={s.textareaContainer}>
                <textarea
                    class={cn(
                        s.textarea,
                        props.size && s[props.size],
                        { [s.error]: !!props.errorMessage },
                        { [s.transparentText]: highlightEnabled() },
                        props.class,
                    )}
                    id={props.id}
                    name={props.name}
                    placeholder={props.placeholder}
                    value={props.value as string}
                    cols={props.cols}
                    rows={props.rows}
                    onChange={handleChange}
                    onInput={handleInput}
                    onBlur={handleBlur}
                    onScroll={handleScroll}
                    wrap={props.wrap}
                    maxLength={props.maxLength}
                    disabled={props.disabled}
                    ref={(el) => setRef(el)}
                />
                <Show when={highlightEnabled()}>
                    <TextareaHighlight
                        value={currentValue}
                        scrollTop={scrollTop}
                        commentPrefixes={props.commentPrefixes}
                    />
                </Show>
            </div>
            <Show when={props.errorMessage}>
                <div class={s.errorMessage}>{props.errorMessage}</div>
            </Show>
        </div>
    );
};
