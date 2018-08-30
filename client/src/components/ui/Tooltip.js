import React from 'react';
import PropTypes from 'prop-types';

import './Tooltip.css';

const Tooltip = props => (
    <div data-tooltip={props.text} className="tooltip-custom"></div>
);

Tooltip.propTypes = {
    text: PropTypes.string.isRequired,
};

export default Tooltip;
