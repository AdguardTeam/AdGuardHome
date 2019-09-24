import React from 'react';
import PropTypes from 'prop-types';

const wrapCell = ({ value }) => (
    <div className="logs__row logs__row--overflow">
        <span className="logs__text" title={value}>
            {value || '–'}
        </span>
    </div>
);

wrapCell.propTypes = {
    value: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
};

export default wrapCell;
