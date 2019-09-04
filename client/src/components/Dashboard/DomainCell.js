import React from 'react';
import PropTypes from 'prop-types';

import { getTrackerData } from '../../helpers/trackers/trackers';
import Popover from '../ui/Popover';

const DomainCell = ({ value }) => {
    const trackerData = getTrackerData(value);

    return (
        <div className="logs__row">
            <div className="logs__text" title={value}>
                {value}
            </div>
            {trackerData && <Popover data={trackerData} />}
        </div>
    );
};

DomainCell.propTypes = {
    value: PropTypes.string.isRequired,
};

export default DomainCell;
