import React from 'react';
import PropTypes from 'prop-types';

import Card from '../ui/Card';

const Status = props => (
    <div className="status">
        <Card bodyType="card-body card-body--status">
            <div className="h4 font-weight-light mb-4">
                You are currently not using AdGuard Home
            </div>
            <button className="btn btn-success" onClick={props.handleStatusChange}>
                Enable AdGuard Home
            </button>
        </Card>
    </div>
);

Status.propTypes = {
    handleStatusChange: PropTypes.func.isRequired,
};

export default Status;
