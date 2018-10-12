import React from 'react';
import PropTypes from 'prop-types';

import './Tooltip.css';

const Tooltip = props => (
    <div data-tooltip={props.text} className={`tooltip-custom ${props.type || ''}`}></div>
);

Tooltip.propTypes = {
    text: PropTypes.string.isRequired,
    type: PropTypes.string,
};

export default Tooltip;
