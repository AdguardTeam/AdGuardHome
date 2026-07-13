import { render } from 'solid-js/web';
import { HashRouter } from '@solidjs/router';

import '../index.pcss';

import { Setup } from './Setup';
import { Icons } from '../common/ui/Icons';

const root = document.getElementById('root')!;

render(
    () => (
        <HashRouter
            root={() => (
                <>
                    <Setup />
                    <Icons />
                </>
            )}
        />
    ),
    root,
);
