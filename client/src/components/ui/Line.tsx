import React from 'react';
import { ResponsiveLine } from '@nivo/line';
import addDays from 'date-fns/add_days';
import subDays from 'date-fns/sub_days';
import subHours from 'date-fns/sub_hours';
import dateFormat from 'date-fns/format';
import round from 'lodash/round';
import { useSelector } from 'react-redux';
import './Line.css';

import { msToDays, msToHours } from '../../helpers/helpers';
import { TIME_UNITS } from '../../helpers/constants';
import { RootState } from '../../initialState';

interface LineProps {
    data: any[];
    color?: string;
    width?: number;
    height?: number;
}

const Line = ({ data, color = 'black' }: LineProps) => {
    const interval = useSelector((state: RootState) => state.stats.interval);

    const timeUnits = useSelector((state: RootState) => state.stats.timeUnits);

    return (
        <ResponsiveLine
            enableArea
            animate
            enableSlices="x"
            curve="linear"
            colors={[color]}
            data={data}
            theme={{
                crosshair: {
                    line: {
                        stroke: 'currentColor',
                        strokeWidth: 1,
                        strokeOpacity: 0.5,
                    },
                },
            }}
            xScale={{
                type: 'linear',
                min: 0,
                max: 'auto',
            }}
            crosshairType="x"
            axisLeft={null}
            axisBottom={null}
            enableGridX={null}
            enableGridY={null}
            enablePoints={null}
            xFormat={(x: number) => {
                if (timeUnits === TIME_UNITS.HOURS) {
                    const hoursAgo = msToHours(interval) - x - 1;
                    return dateFormat(subHours(Date.now(), hoursAgo), 'D MMM HH:00');
                }

                const daysAgo = subDays(Date.now(), msToDays(interval) - 1);

                return dateFormat(addDays(daysAgo, x), 'D MMM YYYY');
            }}
            yFormat={(y: number) => round(y, 2)}
            sliceTooltip={(slice) => {
                const { xFormatted, yFormatted } = slice.slice.points[0].data;

                return (
                    <div className="line__tooltip">
                        <span className="line__tooltip-text">
                            <strong>{yFormatted}</strong>

                            <br />

                            <small>{xFormatted}</small>
                        </span>
                    </div>
                );
            }}
        />
    );
};

export default Line;
