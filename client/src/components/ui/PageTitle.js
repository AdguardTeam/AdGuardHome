import React from 'react';
import PropTypes from 'prop-types';

import './PageTitle.css';

const PageTitle = props => (
    <div className="page-header">
        <h1 className="page-title">
            {props.title}
            {props.subtitle && <span className="page-subtitle">{props.subtitle}</span>}
            {props.children}
        </h1>
    </div>
);

PageTitle.propTypes = {
    title: PropTypes.string.isRequired,
    subtitle: PropTypes.string,
    children: PropTypes.node,
};

export default PageTitle;
