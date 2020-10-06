import React from 'react';
import PropTypes from 'prop-types';

import './PageTitle.css';

const PageTitle = ({
    title, subtitle, children, containerClass,
}) => <div className="page-header">
    <div className={containerClass}>
        <h1 className="page-title pr-2">{title}</h1>
        {children}
    </div>
    {subtitle && <div className="page-subtitle">
        {subtitle}
    </div>}
</div>;

PageTitle.propTypes = {
    title: PropTypes.string.isRequired,
    subtitle: PropTypes.string,
    children: PropTypes.node,
    containerClass: PropTypes.string,
};

export default PageTitle;
