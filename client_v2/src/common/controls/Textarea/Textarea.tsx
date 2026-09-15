import { type JSX, Show, createSignal, createEffect } from 'solid-js';
import cn from 'clsx';

import { Icon } from 'panel/common/ui/Icon';
import intl from 'panel/common/intl';

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
    size?: 'small' | 'medium' | 'large' | 'compact';
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
    /** Shows the clear (×) button while the field has a value. */
    isClearable?: boolean;
    /** Called after the field has been emptied via the clear button. */
    onClear?: () => void;
    'data-testid'?: string;
};

export const Textarea = (props: Props) => {
    let textareaRef: HTMLTextAreaElement | undefined;
    const [currentValue, setCurrentValue] = createSignal('');

    // Sync from external prop changes (e.g. dialog open/close resets);
    // also handles initial value on mount. Runs before browser paint so
    // there is no visible flicker from the empty initial signal.
    createEffect(() => {
        setCurrentValue(props.value as string);
    });

    const highlightEnabled = () => !!props.highlightComments;

    // Convert rows to CSS height for the shared scroll container.
    // line-height: 1.3 * font-size: 16px = 20.8px per row.
    // Add 34px for overlay padding (16px × 2) + scrollArea border (1px × 2).
    const LINE_HEIGHT_PX = 20.8;
    const CHROME_PX = 34;
    const scrollAreaStyle = (): JSX.CSSProperties | undefined => {
        if (!highlightEnabled() || props.rows == null) return undefined;
        return { height: `${Number(props.rows) * LINE_HEIGHT_PX + CHROME_PX}px` };
    };

    const setRef = (el: HTMLTextAreaElement) => {
        textareaRef = el;
        if (typeof props.ref === 'function') {
            props.ref(el);
        }
    };

    const showClearButton = () =>
        Boolean(props.isClearable && !props.disabled && !props.readOnly && currentValue());

    const handleClear = () => {
        if (textareaRef) {
            textareaRef.value = '';
            // Keep the controlled `value` prop in sync: consumers read the
            // value off the event target (same contract as Input).
            props.onChange?.({
                target: textareaRef,
                currentTarget: textareaRef,
            } as unknown as TextareaChangeEvent);
        }
        setCurrentValue('');
        props.onClear?.();
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
        props.onScroll?.(e);
    };

    return (
        <div class={s.textareaWrapper}>
            <Show when={props.label}>
                <label class={s.label} for={props.id}>
                    {props.label}
                </label>
            </Show>
            <div class={s.fieldWrapper}>
                <Show
                    when={highlightEnabled()}
                    fallback={
                        <textarea
                            class={cn(
                                s.textarea,
                                props.size && s[props.size],
                                { [s.error]: !!props.errorMessage },
                                { [s.clearable]: showClearButton() },
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
                            data-testid={props['data-testid']}
                            ref={(el: HTMLTextAreaElement) => setRef(el)}
                        />
                    }
                >
                    <div
                        class={cn(s.scrollArea, props.size && s[props.size], {
                            [s.error]: !!props.errorMessage,
                        })}
                        style={scrollAreaStyle()}
                    >
                        <div class={s.contentWrapper}>
                            <TextareaHighlight
                                value={currentValue}
                                commentPrefixes={props.commentPrefixes}
                            />
                            <textarea
                                class={cn(s.transparentText, props.class)}
                                id={props.id}
                                name={props.name}
                                placeholder={props.placeholder}
                                value={props.value as string}
                                cols={props.cols}
                                onChange={handleChange}
                                onInput={handleInput}
                                onBlur={handleBlur}
                                onScroll={handleScroll}
                                wrap={props.wrap}
                                maxLength={props.maxLength}
                                disabled={props.disabled}
                                data-testid={props['data-testid']}
                                ref={(el: HTMLTextAreaElement) => setRef(el)}
                            />
                        </div>
                    </div>
                </Show>
                <Show when={showClearButton()}>
                    <button
                        type="button"
                        class={s.clearButton}
                        aria-label={intl.getMessage('aria_clear_input')}
                        data-testid="textarea-clear-button"
                        onMouseDown={(event) => event.preventDefault()}
                        onClick={handleClear}
                    >
                        <Icon icon="cross" />
                    </button>
                </Show>
            </div>
            <Show when={props.errorMessage}>
                <div
                    class={s.errorMessage}
                    data-testid={props['data-testid'] ? `${props['data-testid']}-error` : undefined}
                >
                    {props.errorMessage}
                </div>
            </Show>
        </div>
    );
};
