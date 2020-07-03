import React from 'react';
import PropTypes from 'prop-types';

const CellWrap = ({ value }, formatValue, formatTitle = formatValue) => {
    if (!value) {
        return '–';
    }
    const cellValue = typeof formatValue === 'function' ? formatValue(value) : value;
    const cellTitle = typeof formatTitle === 'function' ? formatTitle(value) : value;

    return (
        <div className="logs__row o-hidden">
            <span className="logs__text logs__text--full" title={cellTitle}>
                {cellValue}
            </span>
        </div>
    );
};

CellWrap.propTypes = {
    value: PropTypes.oneOfType([
        PropTypes.string,
        PropTypes.number,
    ]),
    formatValue: PropTypes.func,
    formatTitle: PropTypes.func,
};

export default CellWrap;
