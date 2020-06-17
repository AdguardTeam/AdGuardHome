import React from 'react';
import PropTypes from 'prop-types';

import './Tooltip.css';

const Tooltip = ({ text, type = '' }) => <div data-tooltip={text}
                                              className={`tooltip-custom ml-1 ${type}`} />;

Tooltip.propTypes = {
    text: PropTypes.string.isRequired,
    type: PropTypes.string,
};

export default Tooltip;
