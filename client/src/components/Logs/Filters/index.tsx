import React from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';

import { FiltersForm } from './Form';
import { refreshFilteredLogs } from '../../../actions/queryLogs';
import { addSuccessToast } from '../../../actions/toasts';

interface FiltersProps {
    filter: object;
    processingGetLogs: boolean;
    setIsLoading: (...args: unknown[]) => unknown;
}

const Filters = ({ filter, setIsLoading }: FiltersProps) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const refreshLogs = async () => {
        setIsLoading(true);
        await dispatch(refreshFilteredLogs());
        dispatch(addSuccessToast('query_log_updated'));
        setIsLoading(false);
    };

    return (
        <div className="page-header page-header--logs">
            <h1 className="page-title page-title--large">
                {t('query_log')}

                <button
                    type="button"
                    className="btn btn-icon--green logs__refresh"
                    title={t('refresh_btn')}
                    onClick={refreshLogs}>
                    <svg className="icons icon--24">
                        <use xlinkHref="#update" />
                    </svg>
                </button>
            </h1>
            <FiltersForm responseStatusClass="d-sm-block" setIsLoading={setIsLoading} initialValues={filter} />
        </div>
    );
};

export default Filters;
