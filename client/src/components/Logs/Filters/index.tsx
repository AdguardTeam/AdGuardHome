import React, { useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useDispatch } from 'react-redux';

import { refreshFilteredLogs } from '../../../actions/queryLogs';
import { addSuccessToast } from '../../../actions/toasts';
import AutoRefresh from './AutoRefresh';
import { Form } from './Form';

interface FiltersProps {
    processingGetLogs: boolean;
    setIsLoading: (...args: unknown[]) => unknown;
}

const Filters = ({ setIsLoading }: FiltersProps) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const isRefreshingRef = useRef(false);
    const refreshLogs = async (silently = false) => {
        if (isRefreshingRef.current) {
            return;
        }

        isRefreshingRef.current = true;
        setIsLoading(true);
        await dispatch(refreshFilteredLogs());

        if (!silently) {
            dispatch(addSuccessToast('query_log_updated'));
        }

        setIsLoading(false);
        isRefreshingRef.current = false;
    };

    return (
        <div className="page-header page-header--logs">
            <h1 className="page-title page-title--large">
                {t('query_log')}

                <button
                    type="button"
                    className="btn btn-icon--green logs__refresh"
                    title={t('refresh_btn')}
                    onClick={() => refreshLogs()}>
                    <svg className="icons icon--24">
                        <use xlinkHref="#update" />
                    </svg>
                </button>

                <AutoRefresh refreshLogs={refreshLogs} />
            </h1>
            <Form setIsLoading={setIsLoading} />
        </div>
    );
};

export default Filters;
