import intl from 'panel/common/intl';

import { DayKey } from './helpers';

export const getDayName = (day: DayKey): string => {
    switch (day) {
        case 'mon':
            return intl.getMessage('monday');
        case 'tue':
            return intl.getMessage('tuesday');
        case 'wed':
            return intl.getMessage('wednesday');
        case 'thu':
            return intl.getMessage('thursday');
        case 'fri':
            return intl.getMessage('friday');
        case 'sat':
            return intl.getMessage('saturday');
        case 'sun':
            return intl.getMessage('sunday');
    }
};
