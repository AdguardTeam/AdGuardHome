import React from 'react';
import PropTypes from 'prop-types';

import './PageTitle.css';

const PageTitle = (props) => (
    <div className="page-header">
        <h1 className="page-title">
            {props.title}
            {props.children}
        </h1>
        {props.subtitle && (
            <div className="page-subtitle">
                {props.subtitle}
            </div>
        )}
    </div>
);

PageTitle.propTypes = {
    title: PropTypes.string.isRequired,
    subtitle: PropTypes.string,
    children: PropTypes.node,
};

export default PageTitle;
