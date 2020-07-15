import React, { Fragment, useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import Modal from 'react-modal';
import { shallowEqual, useDispatch, useSelector } from 'react-redux';
import { useHistory } from 'react-router-dom';
import queryString from 'query-string';
import classNames from 'classnames';
import {
    BLOCK_ACTIONS,
    TABLE_DEFAULT_PAGE_SIZE,
    TABLE_FIRST_PAGE,
    smallScreenSize,
} from '../../helpers/constants';
import Loading from '../ui/Loading';
import Filters from './Filters';
import Table from './Table';
import Disabled from './Disabled';
import { getFilteringStatus } from '../../actions/filtering';
import { getClients } from '../../actions';
import { getDnsConfig } from '../../actions/dnsConfig';
import {
    getLogsConfig,
    refreshFilteredLogs,
    resetFilteredLogs,
    setFilteredLogs,
} from '../../actions/queryLogs';
import { addSuccessToast } from '../../actions/toasts';
import './Logs.css';

const processContent = (data, buttonType) => Object.entries(data)
    .map(([key, value]) => {
        if (!value) {
            return null;
        }

        const isTitle = value === 'title';
        const isButton = key === buttonType;
        const isBoolean = typeof value === 'boolean';
        const isHidden = isBoolean && value === false;

        let keyClass = 'key-colon';

        if (isTitle) {
            keyClass = 'title--border';
        }
        if (isButton || isBoolean) {
            keyClass = '';
        }

        return isHidden ? null : <Fragment key={key}>
            <div
                className={classNames(`key__${key}`, keyClass, {
                    'font-weight-bold': isBoolean && value === true,
                })}>
                <Trans>{isButton ? value : key}</Trans>
            </div>
            <div className={`value__${key} text-pre text-truncate`}>
                <Trans>{(isTitle || isButton || isBoolean) ? '' : value || '—'}</Trans>
            </div>
        </Fragment>;
    });


const Logs = (props) => {
    const dispatch = useDispatch();
    const history = useHistory();

    const {
        response_status: response_status_url_param = '',
        search: search_url_param = '',
    } = queryString.parse(history.location.search);

    const { filter } = useSelector((state) => state.queryLogs, shallowEqual);

    const search = filter?.search || search_url_param;
    const response_status = filter?.response_status || response_status_url_param;

    const [isSmallScreen, setIsSmallScreen] = useState(window.innerWidth < smallScreenSize);
    const [detailedDataCurrent, setDetailedDataCurrent] = useState({});
    const [buttonType, setButtonType] = useState(BLOCK_ACTIONS.BLOCK);
    const [isModalOpened, setModalOpened] = useState(false);
    const [isLoading, setIsLoading] = useState(false);


    useEffect(() => {
        (async () => {
            setIsLoading(true);
            await dispatch(setFilteredLogs({
                search,
                response_status,
            }));
            setIsLoading(false);
        })();
    }, [response_status, search]);

    const {
        filtering,
        setLogsPage,
        setLogsPagination,
        toggleDetailedLogs,
        dashboard,
        dnsConfig,
        queryLogs: {
            enabled,
            processingGetConfig,
            processingAdditionalLogs,
            processingGetLogs,
            oldest,
            logs,
            pages,
            page,
            isDetailed,
        },
    } = props;

    const mediaQuery = window.matchMedia(`(max-width: ${smallScreenSize}px)`);
    const mediaQueryHandler = (e) => {
        setIsSmallScreen(e.matches);
        if (e.matches) {
            toggleDetailedLogs(false);
        }
    };

    const closeModal = () => setModalOpened(false);

    const getLogs = (older_than, page, initial) => {
        if (enabled) {
            props.getLogs({
                older_than,
                page,
                pageSize: TABLE_DEFAULT_PAGE_SIZE,
                initial,
            });
        }
    };

    useEffect(() => {
        try {
            mediaQuery.addEventListener('change', mediaQueryHandler);
        } catch (e1) {
            try {
                // Safari 13.1 do not support mediaQuery.addEventListener('change', handler)
                mediaQuery.addListener(mediaQueryHandler);
            } catch (e2) {
                console.error(e2);
            }
        }

        (async () => {
            setIsLoading(true);
            dispatch(setLogsPage(TABLE_FIRST_PAGE));
            dispatch(getFilteringStatus());
            dispatch(getClients());
            try {
                await Promise.all([
                    dispatch(getLogsConfig()),
                    dispatch(getDnsConfig()),
                ]);
            } catch (err) {
                console.error(err);
            } finally {
                setIsLoading(false);
            }
        })();

        return () => {
            try {
                mediaQuery.removeEventListener('change', mediaQueryHandler);
            } catch (e1) {
                try {
                    mediaQuery.removeListener(mediaQueryHandler);
                } catch (e2) {
                    console.error(e2);
                }
            }

            dispatch(resetFilteredLogs());
        };
    }, []);

    const refreshLogs = async () => {
        setIsLoading(true);
        await Promise.all([
            dispatch(setLogsPage(TABLE_FIRST_PAGE)),
            dispatch(refreshFilteredLogs()),
        ]);
        dispatch(addSuccessToast('query_log_updated'));
        setIsLoading(false);
    };

    return (
        <>
            {enabled && processingGetConfig && <Loading />}
            {enabled && !processingGetConfig && (
                <>
                    <Filters
                        filter={{
                            response_status,
                            search,
                        }}
                        setIsLoading={setIsLoading}
                        processingGetLogs={processingGetLogs}
                        processingAdditionalLogs={processingAdditionalLogs}
                        refreshLogs={refreshLogs}
                    />
                    <Table
                        isLoading={isLoading}
                        setIsLoading={setIsLoading}
                        logs={logs}
                        pages={pages}
                        page={page}
                        autoClients={dashboard.autoClients}
                        oldest={oldest}
                        filtering={filtering}
                        processingGetLogs={processingGetLogs}
                        processingGetConfig={processingGetConfig}
                        isDetailed={isDetailed}
                        setLogsPagination={setLogsPagination}
                        setLogsPage={setLogsPage}
                        toggleDetailedLogs={toggleDetailedLogs}
                        getLogs={getLogs}
                        setRules={props.setRules}
                        addSuccessToast={props.addSuccessToast}
                        getFilteringStatus={props.getFilteringStatus}
                        dnssec_enabled={dnsConfig.dnssec_enabled}
                        setDetailedDataCurrent={setDetailedDataCurrent}
                        setButtonType={setButtonType}
                        setModalOpened={setModalOpened}
                        isSmallScreen={isSmallScreen}
                    />
                    <Modal portalClassName='grid' isOpen={isSmallScreen && isModalOpened}
                           onRequestClose={closeModal}
                           style={{
                               content: {
                                   width: '100%',
                                   height: 'fit-content',
                                   left: 0,
                                   top: 47,
                                   padding: '1rem 1.5rem 1rem',
                               },
                               overlay: {
                                   backgroundColor: 'rgba(0,0,0,0.5)',
                               },
                           }}
                    >
                        <svg
                            className="icon icon--small icon-cross d-block d-md-none cursor--pointer"
                            onClick={closeModal}>
                            <use xlinkHref="#cross" />
                        </svg>
                        {processContent(detailedDataCurrent, buttonType)}
                    </Modal>
                </>
            )}
            {!enabled && !processingGetConfig && (
                <Disabled />
            )}
        </>
    );
};

Logs.propTypes = {
    getLogs: PropTypes.func.isRequired,
    queryLogs: PropTypes.object.isRequired,
    dashboard: PropTypes.object.isRequired,
    getFilteringStatus: PropTypes.func.isRequired,
    filtering: PropTypes.object.isRequired,
    setRules: PropTypes.func.isRequired,
    addSuccessToast: PropTypes.func.isRequired,
    setLogsPagination: PropTypes.func.isRequired,
    setLogsPage: PropTypes.func.isRequired,
    toggleDetailedLogs: PropTypes.func.isRequired,
    dnsConfig: PropTypes.object.isRequired,
};

export default Logs;
