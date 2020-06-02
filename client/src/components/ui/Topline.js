import React from 'react';
import PropTypes from 'prop-types';

import './Topline.css';

const Topline = (props) => (
    <div className={`alert alert-${props.type} topline`}>
        <div className="container">
            {props.children}
        </div>
    </div>
);

Topline.propTypes = {
    children: PropTypes.node.isRequired,
    type: PropTypes.string.isRequired,
};

export default Topline;
