import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Icon, IconType } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RoutePath, SCROLL_QUERY_KEY } from 'panel/components/Routes/Paths';

import s from './EmptyState.module.pcss';

export type EmptyStateMode = 'default' | 'disabled';

type Props = {
    class?: string;
    mode?: EmptyStateMode;
};

const getEmptyState = (mode: EmptyStateMode) => {
    if (mode === 'disabled') {
        return {
            message: intl.getMessage('period_notify', {
                a: (text: string) => (
                    <Link
                        to={RoutePath.SettingsPage}
                        query={{ [SCROLL_QUERY_KEY]: 'statistics' }}
                    >
                        {text}
                    </Link>
                ),
            }),
            icon: 'settings_info' as IconType,
        };
    }
    return {
        message: intl.getMessage('no_stats_yet'),
        icon: 'not_found_search' as IconType,
    };
};

export const EmptyState = (props: Props) => {
    const state = () => getEmptyState(props.mode || 'default');

    return (
        <div class={cn(s.emptyState, props.class)} data-testid="dashboard-empty-state">
            <Icon icon={state().icon} class={s.emptyStateIcon} />
            <div class={cn(theme.text.t2, s.emptyStateText)}>{state().message}</div>
        </div>
    );
};
