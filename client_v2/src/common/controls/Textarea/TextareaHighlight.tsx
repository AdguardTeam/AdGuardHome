import { For, type Accessor, createEffect } from 'solid-js';
import cn from 'clsx';

import { isCommentLine } from 'panel/helpers/helpers';

import s from './styles.module.pcss';

type Props = {
    value: Accessor<string>;
    scrollTop: Accessor<number>;
    class?: string;
};

export const TextareaHighlight = (props: Props) => {
    const lines = () => (props.value() || '').split('\n');
    let ref: HTMLDivElement | undefined;

    createEffect(() => {
        const el = ref;
        if (el) {
            el.scrollTop = props.scrollTop();
        }
    });

    return (
        <div
            ref={(el) => {
                ref = el;
            }}
            class={cn(s.overlay, props.class)}
            aria-hidden="true"
        >
            <For each={lines()}>
                {(line, index) => (
                    <>
                        {index() > 0 && '\n'}
                        <span
                            class={cn({
                                [s.commentLine]: isCommentLine(line),
                            })}
                        >
                            {line || ' '}
                        </span>
                    </>
                )}
            </For>
        </div>
    );
};
