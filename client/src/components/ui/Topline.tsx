import React from 'react';

import './Topline.css';

interface ToplineProps {
    children: React.ReactNode;
    type: string;
}

const Topline = (props: ToplineProps) => (
    <div className={`alert alert-${props.type} topline`}>
        <div className="container">{props.children}</div>
    </div>
);

export default Topline;
