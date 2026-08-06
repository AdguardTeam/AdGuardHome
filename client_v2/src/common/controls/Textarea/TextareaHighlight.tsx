import { For, type Accessor, createMemo } from 'solid-js';
import cn from 'clsx';

import { COMMENT_LINE_DEFAULT_TOKEN, type CommentLineToken } from 'panel/helpers/constants';

import s from './styles.module.pcss';

type Props = {
    value: Accessor<string>;
    class?: string;
    /** Comment line prefixes to highlight. Defaults to {@link COMMENT_LINE_DEFAULT_TOKEN}. */
    commentPrefixes?: readonly CommentLineToken[];
};

export const TextareaHighlight = (props: Props) => {
    const lines = () => (props.value() || '').split('\n');

    const prefixes = createMemo(() => props.commentPrefixes || [COMMENT_LINE_DEFAULT_TOKEN]);

    const isComment = (line: string) => {
        const trimmed = line.trimStart();
        return prefixes().some((p) => trimmed.startsWith(p));
    };

    return (
        <div class={cn(s.overlay, props.class)} aria-hidden="true">
            <For each={lines()}>
                {(line, index) => (
                    <>
                        {index() > 0 && '\n'}
                        <span
                            class={cn({
                                [s.commentLine]: isComment(line),
                                [s.overlayNonComment]: !isComment(line),
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
