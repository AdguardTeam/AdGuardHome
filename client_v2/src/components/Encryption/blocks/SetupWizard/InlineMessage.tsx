import type { JSX } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';
import s from './styles.module.pcss';

type InlineMessageKind = 'error' | 'warning';

type Props = {
    kind: InlineMessageKind;
    children: JSX.Element;
    class?: string;
};

export const InlineMessage = (props: Props) => (
    <div
        class={cn(
            s.message,
            theme.text.t3,
            {
                [s.messageError]: props.kind === 'error',
                [s.messageWarning]: props.kind === 'warning',
            },
            props.class,
        )}
    >
        <div>{props.children}</div>
    </div>
);
