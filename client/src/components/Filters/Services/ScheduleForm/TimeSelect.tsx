import React, { useState } from 'react';

import { getTimeFromMs, convertTimeToMs } from './helpers';

interface TimeSelectProps {
    value: number;
    onChange: (time: number) => void;
}

export const TimeSelect = ({ value, onChange }: TimeSelectProps) => {
    const { hours: initialHours, minutes: initialMinutes } = getTimeFromMs(value);

    const [hours, setHours] = useState(initialHours);
    const [minutes, setMinutes] = useState(initialMinutes);

    const hourOptions = Array.from({ length: 24 }, (_, i) => i.toString().padStart(2, '0'));

    const minuteOptions = Array.from({ length: 60 }, (_, i) => i.toString().padStart(2, '0'));

    const onHourChange = (event: any) => {
        setHours(event.target.value);
        onChange(convertTimeToMs(event.target.value, minutes));
    };

    const onMinuteChange = (event: any) => {
        setMinutes(event.target.value);
        onChange(convertTimeToMs(hours, event.target.value));
    };

    return (
        <div className="schedule__time-select">
            <select value={hours} onChange={onHourChange} className="form-control custom-select">
                {hourOptions.map((hour) => (
                    <option key={hour} value={hour}>
                        {hour}
                    </option>
                ))}
            </select>
            &nbsp;:&nbsp;
            <select value={minutes} onChange={onMinuteChange} className="form-control custom-select">
                {minuteOptions.map((minute) => (
                    <option key={minute} value={minute}>
                        {minute}
                    </option>
                ))}
            </select>
        </div>
    );
};
