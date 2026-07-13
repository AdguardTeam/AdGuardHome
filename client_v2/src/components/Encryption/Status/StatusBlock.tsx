import { type JSX } from 'solid-js';
import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

export type StatusVariant = 'success' | 'error' | 'warning';

type Props = {
    variant: StatusVariant;
    title: JSX.Element;
    children?: JSX.Element;
};

export const StatusBlock = (props: Props) => (
    <div class={cn(s.status, theme.text.t3)}>
        <div
            class={cn(s.statusTitle, {
                [s.statusTitle_success]: props.variant === 'success',
                [s.statusTitle_error]: props.variant === 'error',
                [s.statusTitle_warning]: props.variant === 'warning',
            })}
        >
            {props.title}
        </div>
        {props.children}
    </div>
);
