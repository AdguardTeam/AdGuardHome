import React from 'react';
import PropTypes from 'prop-types';

import './Card.css';

const Card = props => (
    <div className={props.type ? `card ${props.type}` : 'card'}>
        {props.title &&
        <div className="card-header with-border">
            <div className="card-inner">
                <div className="card-title">
                    {props.title}
                </div>

                {props.subtitle &&
                    <div className="card-subtitle" dangerouslySetInnerHTML={{ __html: props.subtitle }} />
                }
            </div>

            {props.refresh &&
                <div className="card-options">
                    {props.refresh}
                </div>
            }
        </div>}
        <div className={props.bodyType ? props.bodyType : 'card-body'}>
            {props.children}
        </div>
    </div>
);

Card.propTypes = {
    title: PropTypes.string,
    subtitle: PropTypes.string,
    bodyType: PropTypes.string,
    type: PropTypes.string,
    refresh: PropTypes.node,
    children: PropTypes.node.isRequired,
};

export default Card;
