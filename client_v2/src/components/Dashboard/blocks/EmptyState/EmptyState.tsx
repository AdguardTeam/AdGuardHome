import React from 'react';

import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';

import s from './EmptyState.module.pcss';

export const EmptyState = () => (
    <div className={s.emptyState}>
        <Icon icon="not_found_search" className={s.emptyStateIcon} />
        <div className={s.emptyStateText}>{intl.getMessage('no_stats_yet')}</div>
    </div>
);
