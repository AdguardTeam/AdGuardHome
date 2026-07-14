import cn from 'clsx';

import intl from 'panel/common/intl';
import { Icon, IconType } from 'panel/common/ui/Icon';
import { Link } from 'panel/common/ui/Link';
import { RoutePath, SCROLL_QUERY_KEY } from 'panel/components/Routes/Paths';

import theme from 'panel/lib/theme';
import s from './EmptyState.module.pcss';

export type EmptyStateMode = 'default' | 'disabled' | 'rotation-disabled';

type Props = {
    class?: string;
    mode: EmptyStateMode;
    messageClass?: string;
};

const getEmptyState = (mode: EmptyStateMode) => {
    switch (mode) {
        case 'disabled':
        case 'rotation-disabled':
            return {
                message: (
                    <>
                        {intl.getMessage('query_log_nothing_available_rotation')}
                        <div class={s.enableLinkWrap}>
                            <Link to={RoutePath.SettingsPage} query={{ [SCROLL_QUERY_KEY]: 'query-log' }}>{intl.getMessage('enable')}</Link>
                        </div>
                    </>
                ),
                variant: 'rotation-disabled' as const,
                icon: 'settings_info' as IconType,
            };
        default:
            return {
                message: intl.getMessage('query_log_nothing_available'),
                variant: 'default' as const,
                icon: 'not_found_search' as IconType,
            };
    }
};

export const EmptyState = (props: Props) => {
    const state = () => getEmptyState(props.mode);

    return (
        <div class={cn(s.root, props.class)} data-testid="query-log-empty-state">
            <div class={s.iconWrap}>
                <Icon icon={state().icon} color="gray" class={s.icon} />
            </div>

            <div class={cn(s.message, theme.text.t2, props.messageClass)}>{state().message}</div>
        </div>
    );
};
