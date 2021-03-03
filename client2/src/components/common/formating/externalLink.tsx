import React from 'react';
import theme from 'Lib/theme';

export const externalLink = (link: string) => (e: string) => (
    <a
        href={link}
        target="_blank"
        rel="noopener noreferrer"
        className={theme.link.link}
    >
        {e}
    </a>
);
