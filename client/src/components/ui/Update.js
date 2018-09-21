import React from 'react';
import PropTypes from 'prop-types';

import './Update.css';

const Update = props => (
    <div className="alert alert-info update">
        <div className="container">
            {props.announcement} <a href={props.announcementUrl} target="_blank" rel="noopener noreferrer">Click here</a> for more info.
        </div>
    </div>
);

Update.propTypes = {
    announcement: PropTypes.string.isRequired,
    announcementUrl: PropTypes.string.isRequired,
};

export default Update;
