
import cn from 'clsx';

import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';

import s from './EmptyState.module.pcss';

export const EmptyState = () => (
    <div class={s.emptyState}>
        <Icon icon="not_found_search" class={s.emptyStateIcon} />
        <div class={cn(theme.text.t3, s.emptyStateText)}>{intl.getMessage('no_stats_yet')}</div>
    </div>
);
