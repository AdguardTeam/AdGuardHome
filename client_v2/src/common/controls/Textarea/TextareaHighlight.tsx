import { For, type Accessor, createEffect, createMemo } from 'solid-js';
import cn from 'clsx';

import { COMMENT_LINE_DEFAULT_TOKEN, type CommentLineToken } from 'panel/helpers/constants';

import s from './styles.module.pcss';

type Props = {
    value: Accessor<string>;
    scrollTop: Accessor<number>;
    class?: string;
    /** Comment line prefixes to highlight. Defaults to {@link COMMENT_LINE_DEFAULT_TOKEN}. */
    commentPrefixes?: readonly CommentLineToken[];
};

export const TextareaHighlight = (props: Props) => {
    const lines = () => (props.value() || '').split('\n');
    let ref: HTMLDivElement | undefined;

    const prefixes = createMemo(() => props.commentPrefixes || [COMMENT_LINE_DEFAULT_TOKEN]);

    const isComment = (line: string) => {
        const trimmed = line.trimStart();
        return prefixes().some((p) => trimmed.startsWith(p));
    };

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
                                [s.commentLine]: isComment(line),
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
