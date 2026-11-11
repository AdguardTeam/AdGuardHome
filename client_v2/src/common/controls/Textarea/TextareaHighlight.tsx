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

type HighlightBlock = {
    isComment: boolean;
    text: string;
};

export const TextareaHighlight = (props: Props) => {
    const prefixes = createMemo(() => props.commentPrefixes || [COMMENT_LINE_DEFAULT_TOKEN]);

    const isCommentLine = (line: string) => {
        const trimmed = line.trimStart();
        return prefixes().some((p) => trimmed.startsWith(p));
    };

    // Group adjacent lines by their highlight state. A long comment or rule
    // block then stays a single span instead of one span per line.
    const blocks = createMemo(() => {
        const result: HighlightBlock[] = [];

        (props.value() || '').split('\n').forEach((line) => {
            const isComment = isCommentLine(line);
            const text = line || ' ';
            const previous = result[result.length - 1];

            if (previous?.isComment === isComment) {
                previous.text += `\n${text}`;
            } else {
                result.push({ isComment, text });
            }
        });

        return result;
    });

    return (
        <div class={cn(s.overlay, props.class)} aria-hidden="true">
            <For each={blocks()}>
                {(block, index) => (
                    <>
                        {index() > 0 && '\n'}
                        <span
                            class={cn({
                                [s.commentLine]: block.isComment,
                                [s.overlayNonComment]: !block.isComment,
                            })}
                        >
                            {block.text}
                        </span>
                    </>
                )}
            </For>
        </div>
    );
};
