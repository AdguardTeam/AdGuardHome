import React from 'react';
import PropTypes from 'prop-types';
import { withTranslation, Trans } from 'react-i18next';

import Card from './Card';

const Status = ({ message, buttonMessage, reloadPage }) => (
    <div className="status">
        <Card bodyType="card-body card-body--status">
            <div className="h4 font-weight-light mb-4">
                <Trans>{message}</Trans>
            </div>
            {buttonMessage
            && <button className="btn btn-success" onClick={reloadPage}>
                <Trans>{buttonMessage}</Trans>
            </button>}
        </Card>
    </div>
);

Status.propTypes = {
    message: PropTypes.string.isRequired,
    buttonMessage: PropTypes.string,
    reloadPage: PropTypes.func,
};

export default withTranslation()(Status);
