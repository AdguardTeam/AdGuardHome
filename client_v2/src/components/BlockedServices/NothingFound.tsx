import React from 'react';
import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';

import s from './BlockedServices.module.pcss';

export const NothingFound = () => {
    return (
        <div className={s.nothingFound}>
            <Icon icon="not_found_search" color="gray" className={s.nothingFoundIcon} />
            {intl.getMessage('nothing_found')}
        </div>
    );
};
