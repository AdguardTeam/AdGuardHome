import React, { ReactNode } from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { Icon, IconType } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RoutePath } from 'panel/components/Routes/Paths';

import theme from 'panel/lib/theme';
import s from './EmptyState.module.pcss';

export type EmptyStateMode = 'default' | 'disabled' | 'rotation-disabled';

type Props = {
    className?: string;
    mode: EmptyStateMode;
    messageClassName?: string;
};

const getEmptyState = (
    mode: EmptyStateMode,
): {
    message: ReactNode;
    variant: EmptyStateMode;
    icon: IconType;
} => {
    switch (mode) {
        case 'disabled':
        case 'rotation-disabled':
            return {
                message: intl.getMessage('query_log_nothing_available_rotation', {
                    a: (text: string) => <Link to={RoutePath.SettingsPage}>{text}</Link>,
                }),
                variant: 'rotation-disabled',
                icon: 'settings_info',
            };
        default:
            return {
                message: intl.getMessage('query_log_nothing_available'),
                variant: 'default',
                icon: 'not_found_search',
            };
    }
};

export const EmptyState = ({ className, mode, messageClassName }: Props) => {
    const { message, icon } = getEmptyState(mode);

    return (
        <div className={cn(s.root, className)} data-testid="query-log-empty-state">
            <div className={s.iconWrap}>
                <Icon icon={icon} color="gray" className={s.icon} />
            </div>

            <div className={cn(s.message, theme.text.t2, messageClassName)}>{message}</div>
        </div>
    );
};
