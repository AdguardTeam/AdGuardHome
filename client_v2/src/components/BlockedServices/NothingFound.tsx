import intl from 'panel/common/intl';
import { Icon } from 'panel/common/ui/Icon';

import s from './BlockedServices.module.pcss';

export const NothingFound = () => {
    return (
        <div class={s.nothingFound} data-testid="blocked-services-nothing-found">
            <Icon icon="not_found_search" color="gray" class={s.nothingFoundIcon} />
            {intl.getMessage('nothing_found')}
        </div>
    );
};
