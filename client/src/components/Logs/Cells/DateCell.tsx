import React from 'react';
import { useSelector } from 'react-redux';

import { formatDateTime, formatTime } from '../../../helpers/helpers';
import { DEFAULT_SHORT_DATE_FORMAT_OPTIONS, DEFAULT_TIME_FORMAT } from '../../../helpers/constants';
import { RootState } from '../../../initialState';

interface DateCellProps {
    time: string;
}

const DateCell = ({ time }: DateCellProps) => {
    const isDetailed = useSelector((state: RootState) => state.queryLogs.isDetailed);

    if (!time) {
        return <>–</>;
    }

    const formattedTime = formatTime(time, DEFAULT_TIME_FORMAT);

    const formattedDate = formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS);

    return (
        <div className="logs__cell logs__cell logs__cell--date text-truncate" role="gridcell">
            <div className="logs__time" title={formattedTime}>
                {formattedTime}
            </div>
            {isDetailed && (
                <div className="detailed-info d-none d-sm-block text-truncate" title={formattedDate}>
                    {formattedDate}
                </div>
            )}
        </div>
    );
};

export default DateCell;
