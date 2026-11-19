import { render } from 'solid-js/web';
import { createDecorator, type Preview } from 'storybook-solidjs-vite';

import { Icons } from '../src/common/ui/Icons';

import '../src/index.pcss';

interface StoryGlobals {
    theme?: 'light' | 'dark';
}

// Icon components reference `<use href="#name">`, and a docs page renders
// several stories at once, so the sprite is mounted once per document.
const spriteHost = document.createElement('div');
document.body.appendChild(spriteHost);
render(() => <Icons />, spriteHost);

const withTheme = createDecorator((Story, context) => {
    const theme = (context.globals as StoryGlobals).theme === 'dark' ? 'dark' : 'light';

    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme;

    return Story();
});

const preview: Preview = {
    tags: ['autodocs'],
    decorators: [withTheme],
    initialGlobals: {
        theme: 'light',
    },
    globalTypes: {
        theme: {
            description: 'UI theme',
            toolbar: {
                title: 'Theme',
                icon: 'circlehollow',
                items: [
                    { value: 'light', title: 'Light' },
                    { value: 'dark', title: 'Dark' },
                ],
                dynamicTitle: true,
            },
        },
    },
    parameters: {
        controls: {
            matchers: {
                color: /(background|color)$/i,
                date: /Date$/i,
            },
        },
    },
};

export default preview;
