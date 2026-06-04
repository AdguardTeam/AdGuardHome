import React from 'react';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';
import theme from 'panel/lib/theme';

import s from './TableEmptyState.module.pcss';

export const TableEmptyState = () => (
    <div className={s.emptyTableContent}>
        <Icon icon="not_found_search" color="gray" className={s.emptyTableIcon} />
        <div className={cn(theme.text.t3, s.emptyTableDesc)}>
            {intl.getMessage('clients_not_found')}
        </div>
    </div>
);
