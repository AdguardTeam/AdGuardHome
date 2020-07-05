import React from 'react';
import PropTypes from 'prop-types';

import './PageTitle.css';

const PageTitle = ({ title, subtitle, children }) => (
    <div className="page-header">
        <h1 className="page-title">
            {title}
            {children}
        </h1>
        {subtitle && (
            <div className="page-subtitle">
                {subtitle}
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
