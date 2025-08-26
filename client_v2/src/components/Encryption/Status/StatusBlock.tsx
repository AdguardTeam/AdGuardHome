import React, { ReactNode } from 'react';
import cn from 'clsx';

import theme from 'panel/lib/theme';

import s from './styles.module.pcss';

export type StatusVariant = 'success' | 'error' | 'warning';

type Props = {
    variant: StatusVariant;
    title: ReactNode;
    children?: ReactNode;
};

export const StatusBlock = ({ variant, title, children }: Props) => (
    <div className={cn(s.status, theme.text.t3)}>
        <div
            className={cn(s.statusTitle, {
                [s.statusTitle_success]: variant === 'success',
                [s.statusTitle_error]: variant === 'error',
                [s.statusTitle_warning]: variant === 'warning',
            })}>
            {title}
        </div>
        {children}
    </div>
);
