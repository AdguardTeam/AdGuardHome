import React from 'react';
import classnames from 'classnames';
import { COMMENT_LINE_DEFAULT_TOKEN } from './constants';

const renderHighlightedLine = (line: any, idx: any, commentLineTokens = [COMMENT_LINE_DEFAULT_TOKEN]) => {
    const isComment = commentLineTokens.some((token) => line.trim().startsWith(token));

    const lineClassName = classnames({
        'text-gray': isComment,
        'text-transparent': !isComment,
    });

    return (
        <div className={lineClassName} key={idx}>
            {line || '\n'}
        </div>
    );
};
export const getTextareaCommentsHighlight = (ref: any, lines: any, commentLineTokens?: any, className = '') => {
    const renderLine = (line: any, idx: any) => renderHighlightedLine(line, idx, commentLineTokens);

    return (
        <code className={classnames('text-output font-monospace', className)} ref={ref}>
            {lines?.split('\n').map(renderLine)}
        </code>
    );
};

export const syncScroll = (e: any, ref: any) => {
    // eslint-disable-next-line no-param-reassign
    ref.current.scrollTop = e.target.scrollTop;
};
