import React from 'react';
import PropTypes from 'prop-types';
import { useTranslation } from 'react-i18next';
import Form from './Form';

const Filters = ({ filter, refreshLogs, setIsLoading }) => {
    const { t } = useTranslation();

    return <div className="page-header page-header--logs">
        <h1 className="page-title page-title--large">
            {t('query_log')}
            <button
                    type="button"
                    className="btn btn-icon--green logs__refresh"
                    onClick={refreshLogs}
            >
                <svg className="icons icon--24">
                    <use xlinkHref="#update" />
                </svg>
            </button>
        </h1>
        <Form
                responseStatusClass="d-sm-block"
                initialValues={filter}
                setIsLoading={setIsLoading}
        />
    </div>;
};

Filters.propTypes = {
    filter: PropTypes.object.isRequired,
    refreshLogs: PropTypes.func.isRequired,
    processingGetLogs: PropTypes.bool.isRequired,
    setIsLoading: PropTypes.func.isRequired,
};

export default Filters;
