import React from 'react';
import PropTypes from 'prop-types';

const CellWrap = ({ value }) => (
    <div className="logs__row logs__row--overflow">
        <span className="logs__text logs__text--full" title={value}>
            {value}
        </span>
    </div>
);

CellWrap.propTypes = {
    value: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
};

export default CellWrap;
