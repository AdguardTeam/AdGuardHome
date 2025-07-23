import type { Preview } from '@storybook/react-webpack5';
import React from 'react';
import { Icons } from '../src/common/ui/Icons';

import '../src/index.pcss';

const preview: Preview = {
    parameters: {
        controls: {
            matchers: {
                color: /(background|color)$/i,
                date: /Date$/i,
            },
        },
    },
    decorators: [
        (Story) => React.createElement(
            React.Fragment,
            null,
            React.createElement(Icons),
            React.createElement(Story)
        ),
    ],
};

export default preview;
