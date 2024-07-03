import { useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';
import classNames from 'classnames';
import React from 'react';
import { toggleDetailedLogs } from '../../../actions/queryLogs';

import HeaderCell from './HeaderCell';
import { RootState } from '../../../initialState';

const Header = () => {
    const { t } = useTranslation();
    const dispatch = useDispatch();

    const isDetailed = useSelector((state: RootState) => state.queryLogs.isDetailed);
    const disableDetailedMode = () => dispatch(toggleDetailedLogs(false));
    const enableDetailedMode = () => dispatch(toggleDetailedLogs(true));

    const HEADERS = [
        {
            className: 'logs__cell--date',
            content: 'time_table_header',
        },
        {
            className: 'logs__cell--domain',
            content: 'request_table_header',
        },
        {
            className: 'logs__cell--response',
            content: 'response_table_header',
        },
        {
            className: 'logs__cell--client',

            content: (
                <>
                    {t('client_table_header')}

                    {
                        <span>
                            <svg
                                className={classNames('icons icon--24 icon--green cursor--pointer mr-2', {
                                    'icon--selected': !isDetailed,
                                })}
                                onClick={disableDetailedMode}>
                                <title>{t('compact')}</title>

                                <use xlinkHref="#list" />
                            </svg>

                            <svg
                                className={classNames('icons icon--24 icon--green cursor--pointer', {
                                    'icon--selected': isDetailed,
                                })}
                                onClick={enableDetailedMode}>
                                <title>{t('default')}</title>

                                <use xlinkHref="#detailed_list" />
                            </svg>
                        </span>
                    }
                </>
            ),
        },
    ];

    return (
        <div className="logs__cell--header__container px-5" role="row">
            {HEADERS.map(HeaderCell)}
        </div>
    );
};

export default Header;
