import React from 'react';
import PropTypes from 'prop-types';

import './Card.css';

const Card = ({
    type, id, title, subtitle, refresh, bodyType, children,
}) => (
    <div className={type ? `card ${type}` : 'card'} id={id || ''}>
        {(title || subtitle) && (
            <div className="card-header with-border">
                <div className="card-inner">
                    {title && (
                        <div className="card-title">
                            {title}
                        </div>
                    )}

                    {subtitle && (
                        <div
                            className="card-subtitle"
                            dangerouslySetInnerHTML={{ __html: subtitle }}
                        />
                    )}
                </div>

                {refresh && (
                    <div className="card-options">
                        {refresh}
                    </div>
                )}
            </div>
        )}
        <div className={bodyType || 'card-body'}>
            {children}
        </div>
    </div>
);

Card.propTypes = {
    id: PropTypes.string,
    title: PropTypes.string,
    subtitle: PropTypes.string,
    bodyType: PropTypes.string,
    type: PropTypes.string,
    refresh: PropTypes.node,
    children: PropTypes.node.isRequired,
};

export default Card;
