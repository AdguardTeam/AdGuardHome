import React from 'react';
import theme from 'Lib/theme';

const code = (e: string) => {
    return (
        <code className={theme.text.code}>
            {e}
        </code>
    );
};

export default code;
