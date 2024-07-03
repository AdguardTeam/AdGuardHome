import React from 'react';

import { getTimeFromMs } from './helpers';

interface TimePeriodProps {
    startTimeMs: number;
    endTimeMs: number;
}

export const TimePeriod = ({ startTimeMs, endTimeMs }: TimePeriodProps) => {
    const startTime = getTimeFromMs(startTimeMs);
    const endTime = getTimeFromMs(endTimeMs);

    return (
        <div className="schedule__time">
            <time>
                {startTime.hours}:{startTime.minutes}
            </time>
            &nbsp;–&nbsp;
            <time>
                {endTime.hours}:{endTime.minutes}
            </time>
        </div>
    );
};
