import React from 'react';

import { formatTime, formatDateTime } from '../../../helpers/helpers';
import {
    DEFAULT_SHORT_DATE_FORMAT_OPTIONS,
    DEFAULT_TIME_FORMAT,
} from '../../../helpers/constants';

const getDateCell = (row, isDetailed) => {
    const { time } = row.original;

    if (!time) {
        return '–';
    }

    const formattedTime = formatTime(time, DEFAULT_TIME_FORMAT);
    const formattedDate = formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS);

    return (
        <div className="logs__cell">
            <div className="logs__time" title={formattedTime}>{formattedTime}</div>
            {isDetailed && <div className="detailed-info d-none d-sm-block text-truncate"
                                title={formattedDate}>{formattedDate}</div>}
        </div>
    );
};

export default getDateCell;
