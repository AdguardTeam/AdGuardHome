import React from 'react';
import PropTypes from 'prop-types';

import { getTimeFromMs } from './helpers';

export const TimePeriod = ({
    startTimeMs,
    endTimeMs,
}) => {
    const startTime = getTimeFromMs(startTimeMs);
    const endTime = getTimeFromMs(endTimeMs);

    return (
        <div className="schedule__time">
            <time>{startTime.hours}:{startTime.minutes}</time>
            &nbsp;–&nbsp;
            <time>{endTime.hours}:{endTime.minutes}</time>
        </div>
    );
};

TimePeriod.propTypes = {
    startTimeMs: PropTypes.number.isRequired,
    endTimeMs: PropTypes.number.isRequired,
};
