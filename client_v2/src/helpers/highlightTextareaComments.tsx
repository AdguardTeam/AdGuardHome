import React, { type RefObject, type UIEvent } from 'react';
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

    return (
        <div className={lineClassName} key={idx}>
            {line || '\n'}
        </div>
    );
};

export const getTextareaCommentsHighlight = (
    ref: RefObject<HTMLElement>,
    lines: string,
    commentLineTokens: CommentLineTokens = [COMMENT_LINE_DEFAULT_TOKEN],
    className = '',
) => {
    const renderLine = (line: string, idx: number) =>
        renderHighlightedLine(line, idx, commentLineTokens);

    return (
        <code className={classnames(theme.highlight.textOutput, className)} ref={ref}>
            {lines.split('\n').map(renderLine)}
        </code>
    );
};

export const syncScroll = (e: UIEvent<HTMLElement>, ref: RefObject<HTMLElement>) => {
    if (!ref.current) {
        return;
    }

    const target = e.target as HTMLElement;
    // eslint-disable-next-line no-param-reassign
    ref.current.scrollTop = target.scrollTop;
};
