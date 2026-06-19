import classnames from 'clsx';
import theme from 'panel/lib/theme';
import { COMMENT_LINE_DEFAULT_TOKEN } from './constants';

type CommentLineTokens = string[];

const renderHighlightedLine = (
    line: string,
    idx: number,
    commentLineTokens: CommentLineTokens = [COMMENT_LINE_DEFAULT_TOKEN],
) => {
    const isComment = commentLineTokens.some((token) => line.trim().startsWith(token));

    const lineClassName = classnames({
        [theme.highlight.textGray]: isComment,
        [theme.highlight.textTransparent]: !isComment,
    });

    return <div class={lineClassName}>{line || '\n'}</div>;
};

export const getTextareaCommentsHighlight = (
    ref: HTMLElement | undefined,
    lines: string,
    commentLineTokens: CommentLineTokens = [COMMENT_LINE_DEFAULT_TOKEN],
    className = '',
) => {
    const renderLine = (line: string, idx: number) =>
        renderHighlightedLine(line, idx, commentLineTokens);

    return (
        <code class={classnames(theme.highlight.textOutput, className)} ref={ref}>
            {lines.split('\n').map(renderLine)}
        </code>
    );
};

export const syncScroll = (e: UIEvent, ref: HTMLElement | undefined) => {
    if (!ref) {
        return;
    }

    const target = e.target as HTMLElement;
    // eslint-disable-next-line no-param-reassign
    ref.scrollTop = target.scrollTop;
};
