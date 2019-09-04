import React from 'react';
import PropTypes from 'prop-types';

const Cell = props => (
    <div className="stats__row">
        <div className="stats__row-value mb-1">
            <strong>{props.value}</strong>
            <small className="ml-3 text-muted">{props.percent}%</small>
        </div>
        <div className="progress progress-xs">
            <div
                className="progress-bar"
                style={{
                    width: `${props.percent}%`,
                    backgroundColor: props.color,
                }}
            />
        </div>
    </div>
);

Cell.propTypes = {
    value: PropTypes.number.isRequired,
    percent: PropTypes.number.isRequired,
    color: PropTypes.string.isRequired,
};

export default Cell;
