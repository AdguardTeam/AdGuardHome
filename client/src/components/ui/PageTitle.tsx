import React from 'react';

import './PageTitle.css';

interface PageTitleProps {
    title: string;
    subtitle?: string;
    children?: React.ReactNode;
    containerClass?: string;
}

const PageTitle = ({ title, subtitle, children, containerClass }: PageTitleProps) => (
    <div className="page-header">
        <div className={containerClass}>
            <h1 className="page-title pr-2">{title}</h1>
            {children}
        </div>

        {subtitle && <div className="page-subtitle">{subtitle}</div>}
    </div>
);

export default PageTitle;
