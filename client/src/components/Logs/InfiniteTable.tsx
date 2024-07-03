import React, { Dispatch, SetStateAction, useCallback, useEffect, useRef } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useTranslation } from 'react-i18next';
import throttle from 'lodash/throttle';

import Loading from '../ui/Loading';

import Header from './Cells/Header';
import { getLogs } from '../../actions/queryLogs';

import Row from './Cells';

import { isScrolledIntoView } from '../../helpers/helpers';
import { QUERY_LOGS_PAGE_LIMIT } from '../../helpers/constants';
import { RootState } from '../../initialState';

interface InfiniteTableProps {
    isLoading: boolean;
    items: unknown[];
    isSmallScreen: boolean;
    setDetailedDataCurrent: Dispatch<SetStateAction<any>>;
    setButtonType: (...args: unknown[]) => unknown;
    setModalOpened: (...args: unknown[]) => unknown;
}

const InfiniteTable = ({
    isLoading,
    items,
    isSmallScreen,
    setDetailedDataCurrent,
    setButtonType,
    setModalOpened,
}: InfiniteTableProps) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const loader = useRef(null);
    const loadingRef = useRef(null);

    const isEntireLog = useSelector((state: RootState) => state.queryLogs.isEntireLog);

    const processingGetLogs = useSelector((state: RootState) => state.queryLogs.processingGetLogs);
    const loading = isLoading || processingGetLogs;

    const listener = useCallback(() => {
        if (!loadingRef.current && loader.current && isScrolledIntoView(loader.current)) {
            dispatch(getLogs());
        }
    }, []);

    useEffect(() => {
        loadingRef.current = processingGetLogs;
    }, [processingGetLogs]);

    useEffect(() => {
        listener();
    }, [items.length < QUERY_LOGS_PAGE_LIMIT, isEntireLog]);

    useEffect(() => {
        const THROTTLE_TIME = 100;
        const throttledListener = throttle(listener, THROTTLE_TIME);

        window.addEventListener('scroll', throttledListener);
        return () => {
            window.removeEventListener('scroll', throttledListener);
        };
    }, []);

    const renderRow = (row: any, idx: any) => (
        <Row
            key={idx}
            rowProps={row}
            isSmallScreen={isSmallScreen}
            setDetailedDataCurrent={setDetailedDataCurrent}
            setButtonType={setButtonType}
            setModalOpened={setModalOpened}
        />
    );

    const isNothingFound = items.length === 0 && !processingGetLogs;

    return (
        <div className="logs__table" role="grid">
            {loading && <Loading />}

            <Header />
            {isNothingFound ? (
                <label className="logs__no-data">{t('nothing_found')}</label>
            ) : (
                <>
                    {items.map(renderRow)}
                    {!isEntireLog && (
                        <div ref={loader} className="logs__loading text-center">
                            {t('loading_table_status')}
                        </div>
                    )}
                </>
            )}
        </div>
    );
};

export default InfiniteTable;
