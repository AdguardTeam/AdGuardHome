import React from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import Form from './Form';

const Filters = ({ filter, refreshLogs, setIsLoading }) => (
        <div className="page-header page-header--logs">
            <h1 className="page-title page-title--large">
                <Trans>query_log</Trans>
                <button
                    type="button"
                    className="btn btn-icon--green ml-3 bg-transparent"
                    onClick={refreshLogs}
                >
                    <svg className="icons icon--small">
                        <use xlinkHref="#update" />
                    </svg>
                </button>
            </h1>
            <Form
                responseStatusClass="d-sm-block"
                initialValues={filter}
                setIsLoading={setIsLoading}
            />
        </div>
);

Filters.propTypes = {
    filter: PropTypes.object.isRequired,
    refreshLogs: PropTypes.func.isRequired,
    processingGetLogs: PropTypes.bool.isRequired,
    setIsLoading: PropTypes.func.isRequired,
};

export default Filters;
