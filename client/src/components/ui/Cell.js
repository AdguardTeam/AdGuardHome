import React from 'react';
import PropTypes from 'prop-types';
import LogsSearchLink from './LogsSearchLink';
import { formatNumber } from '../../helpers/helpers';

const Cell = ({
    value, percent, color, search,
}) => <div className="stats__row">
    <div className="stats__row-value mb-1">
        <strong><LogsSearchLink search={search}>{formatNumber(value)}</LogsSearchLink></strong>
        <small className="ml-3 text-muted">{percent}%</small>
    </div>
    <div className="progress progress-xs">
        <div
            className="progress-bar"
            style={{
                width: `${percent}%`,
                backgroundColor: color,
            }}
        />
    </div>
</div>;

Cell.propTypes = {
    value: PropTypes.number.isRequired,
    percent: PropTypes.number.isRequired,
    color: PropTypes.string.isRequired,
    search: PropTypes.string,
    onSearchRedirect: PropTypes.func,
};

export default Cell;
