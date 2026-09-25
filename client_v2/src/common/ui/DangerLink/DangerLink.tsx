import { type JSX } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './DangerLink.module.pcss';

type Props = {
    children: JSX.Element;
    onClick: () => void;
    class?: string;
    disabled?: boolean;
    testId?: string;
};

export const DangerLink = (props: Props) => (
    <button
        type="button"
        data-testid={props.testId}
        disabled={props.disabled}
        onClick={() => props.onClick()}
        class={cn(theme.text.t2, theme.text.condenced, s.dangerLink, props.class)}
    >
        {props.children}
    </button>
);
