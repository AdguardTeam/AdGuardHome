import cn from 'clsx';
import { Show } from 'solid-js';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Icon, IconType } from 'panel/common/ui/Icon';
import { enableStatistics, statsState } from 'panel/stores/stats';

import s from './EmptyState.module.pcss';

export type EmptyStateMode = 'default' | 'disabled';

type Props = {
    class?: string;
    mode?: EmptyStateMode;
    onEnable?: () => void;
};

const getEmptyState = (mode: EmptyStateMode) => {
    if (mode === 'disabled') {
        return {
            message: intl.getMessage('dashboard_statistics_disabled'),
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

    const handleEnable = () => {
        if (props.onEnable) {
            props.onEnable();
        } else {
            enableStatistics();
        }
    };

    return (
        <div class={cn(s.emptyState, props.class)} data-testid="dashboard-empty-state">
            <Icon icon={state().icon} class={s.emptyStateIcon} />
            <div
                class={cn(
                    theme.text.t2,
                    props.mode === 'disabled' && theme.text.condenced,
                    s.emptyStateText,
                )}
            >
                {state().message}
            </div>
            <Show when={props.mode === 'disabled'}>
                <button
                    type="button"
                    class={cn(
                        theme.text.t3,
                        theme.link.link,
                        theme.link.hoverDecoration,
                        s.enableButton,
                    )}
                    disabled={statsState.processingSetConfig}
                    onClick={handleEnable}
                >
                    {intl.getMessage('enable')}
                </button>
            </Show>
        </div>
    );
};
