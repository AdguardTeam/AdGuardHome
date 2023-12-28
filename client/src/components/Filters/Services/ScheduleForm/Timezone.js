import React from 'react';
import ct from 'countries-and-timezones';
import { useTranslation } from 'react-i18next';
import PropTypes from 'prop-types';

import { LOCAL_TIMEZONE_VALUE } from '../../../../helpers/constants';

export const Timezone = ({
    timezone,
    setTimezone,
}) => {
    const [t] = useTranslation();

    const onTimeZoneChange = (event) => {
        setTimezone(event.target.value);
    };

    const timezones = ct.getAllTimezones();

    return (
        <div className="schedule__timezone">
            <label className="form__label form__label--with-desc mb-2">
                {t('schedule_timezone')}
            </label>

            <select
                className="form-control custom-select"
                value={timezone}
                onChange={onTimeZoneChange}
            >
                <option value={LOCAL_TIMEZONE_VALUE}>
                    {t('schedule_timezone')}
                </option>
                {/* TODO: get timezones from backend method when the method is ready */}
                {Object.keys(timezones).map((zone) => (
                    <option key={zone} value={zone}>
                        {zone} (GMT{timezones[zone].utcOffsetStr})
                    </option>
                ))}
            </select>
        </div>
    );
};

Timezone.propTypes = {
    timezone: PropTypes.string.isRequired,
    setTimezone: PropTypes.func.isRequired,
};
