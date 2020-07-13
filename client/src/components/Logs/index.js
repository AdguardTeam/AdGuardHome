import React, { Fragment, useEffect, useState } from 'react';
import PropTypes from 'prop-types';
import { Trans } from 'react-i18next';
import Modal from 'react-modal';
import { useDispatch } from 'react-redux';
import {
    BLOCK_ACTIONS, smallScreenSize,
    TABLE_DEFAULT_PAGE_SIZE,
    TABLE_FIRST_PAGE,
} from '../../helpers/constants';
import Loading from '../ui/Loading';
import Filters from './Filters';
import Table from './Table';
import Disabled from './Disabled';
import { getFilteringStatus } from '../../actions/filtering';
import { getClients } from '../../actions';
import { getDnsConfig } from '../../actions/dnsConfig';
import { getLogsConfig } from '../../actions/queryLogs';
import { addSuccessToast } from '../../actions/toasts';
import './Logs.css';

const INITIAL_REQUEST = true;
const INITIAL_REQUEST_DATA = ['', TABLE_FIRST_PAGE, INITIAL_REQUEST];

export const processContent = (data, buttonType) => Object.entries(data)
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
                className={`key__${key} ${keyClass} ${(isBoolean && value === true) ? 'font-weight-bold' : ''}`}>
                <Trans>{isButton ? value : key}</Trans>
            </div>
            <div className={`value__${key} text-pre text-truncate`}>
                <Trans>{(isTitle || isButton || isBoolean) ? '' : value || '—'}</Trans>
            </div>
        </Fragment>;
    });


const Logs = (props) => {
    const dispatch = useDispatch();
    const [isSmallScreen, setIsSmallScreen] = useState(window.innerWidth < smallScreenSize);
    const [detailedDataCurrent, setDetailedDataCurrent] = useState({});
    const [buttonType, setButtonType] = useState(BLOCK_ACTIONS.BLOCK);
    const [isModalOpened, setModalOpened] = useState(false);
    const [isLoading, setIsLoading] = useState(false);

    const {
        filtering,
        setLogsPage,
        setLogsPagination,
        setLogsFilter,
        toggleDetailedLogs,
        dashboard,
        dnsConfig,
        queryLogs: {
            filter,
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

    useEffect(() => {
        mediaQuery.addListener(mediaQueryHandler);

        return () => mediaQuery.removeListener(mediaQueryHandler);
    }, []);

    const closeModal = () => setModalOpened(false);

    const getLogs = (older_than, page, initial) => {
        if (props.queryLogs.enabled) {
            props.getLogs({
                older_than,
                page,
                pageSize: TABLE_DEFAULT_PAGE_SIZE,
                initial,
            });
        }
    };

    useEffect(() => {
        (async () => {
            setIsLoading(true);
            dispatch(setLogsPage(TABLE_FIRST_PAGE));
            dispatch(getFilteringStatus());
            dispatch(getClients());
            try {
                await Promise.all([
                    getLogs(...INITIAL_REQUEST_DATA),
                    dispatch(getLogsConfig()),
                    dispatch(getDnsConfig()),
                ]);
            } catch (err) {
                console.error(err);
            } finally {
                setIsLoading(false);
            }
        })();
    }, []);

    const refreshLogs = async () => {
        setIsLoading(true);
        await Promise.all([
            dispatch(setLogsPage(TABLE_FIRST_PAGE)),
            getLogs(...INITIAL_REQUEST_DATA),
        ]);
        dispatch(addSuccessToast('query_log_updated'));
        setIsLoading(false);
    };

    return (
        <>
            {enabled && processingGetConfig && <Loading />}
            {enabled && !processingGetConfig && (
                <Fragment>
                    <Filters
                        filter={filter}
                        setIsLoading={setIsLoading}
                        processingGetLogs={processingGetLogs}
                        processingAdditionalLogs={processingAdditionalLogs}
                        setLogsFilter={setLogsFilter}
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
                </Fragment>
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
    setLogsFilter: PropTypes.func.isRequired,
    setLogsPage: PropTypes.func.isRequired,
    toggleDetailedLogs: PropTypes.func.isRequired,
    dnsConfig: PropTypes.object.isRequired,
};

export default Logs;
