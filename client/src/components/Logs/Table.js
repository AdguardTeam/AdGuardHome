import React from 'react';
import PropTypes from 'prop-types';
import { useTranslation, Trans } from 'react-i18next';
import ReactTable from 'react-table';
import classNames from 'classnames';
import endsWith from 'lodash/endsWith';
import escapeRegExp from 'lodash/escapeRegExp';
import {
    BLOCK_ACTIONS,
    DEFAULT_SHORT_DATE_FORMAT_OPTIONS,
    LONG_TIME_FORMAT,
    FILTERED_STATUS_TO_META_MAP,
    TABLE_DEFAULT_PAGE_SIZE,
    SCHEME_TO_PROTOCOL_MAP,
    CUSTOM_FILTERING_RULES_ID, FILTERED_STATUS,
} from '../../helpers/constants';
import getDateCell from './Cells/getDateCell';
import getDomainCell from './Cells/getDomainCell';
import getClientCell from './Cells/getClientCell';
import getResponseCell from './Cells/getResponseCell';

import {
    captitalizeWords,
    checkFiltered,
    formatDateTime,
    formatElapsedMs,
    formatTime,

} from '../../helpers/helpers';
import Loading from '../ui/Loading';
import { getSourceData } from '../../helpers/trackers/trackers';

const Table = (props) => {
    const {
        setDetailedDataCurrent,
        setButtonType,
        setModalOpened,
        isSmallScreen,
        setIsLoading,
        filtering,
        isDetailed,
        toggleDetailedLogs,
        setLogsPage,
        setLogsPagination,
        processingGetLogs,
        logs,
        pages,
        page,
        isLoading,
    } = props;

    const [t] = useTranslation();

    const toggleBlocking = (type, domain) => {
        const {
            setRules, getFilteringStatus, addSuccessToast,
        } = props;
        const { userRules } = filtering;

        const lineEnding = !endsWith(userRules, '\n') ? '\n' : '';
        const baseRule = `||${domain}^$important`;
        const baseUnblocking = `@@${baseRule}`;

        const blockingRule = type === BLOCK_ACTIONS.BLOCK ? baseUnblocking : baseRule;
        const unblockingRule = type === BLOCK_ACTIONS.BLOCK ? baseRule : baseUnblocking;
        const preparedBlockingRule = new RegExp(`(^|\n)${escapeRegExp(blockingRule)}($|\n)`);
        const preparedUnblockingRule = new RegExp(`(^|\n)${escapeRegExp(unblockingRule)}($|\n)`);

        const matchPreparedBlockingRule = userRules.match(preparedBlockingRule);
        const matchPreparedUnblockingRule = userRules.match(preparedUnblockingRule);

        if (matchPreparedBlockingRule) {
            setRules(userRules.replace(`${blockingRule}`, ''));
            addSuccessToast(`${t('rule_removed_from_custom_filtering_toast')}: ${blockingRule}`);
        } else if (!matchPreparedUnblockingRule) {
            setRules(`${userRules}${lineEnding}${unblockingRule}\n`);
            addSuccessToast(`${t('rule_added_to_custom_filtering_toast')}: ${unblockingRule}`);
        } else if (matchPreparedUnblockingRule) {
            addSuccessToast(`${t('rule_added_to_custom_filtering_toast')}: ${unblockingRule}`);
            return;
        } else if (!matchPreparedBlockingRule) {
            addSuccessToast(`${t('rule_removed_from_custom_filtering_toast')}: ${blockingRule}`);
            return;
        }

        getFilteringStatus();
    };

    const getFilterName = (filters, whitelistFilters, filterId, t) => {
        if (filterId === CUSTOM_FILTERING_RULES_ID) {
            return t('custom_filter_rules');
        }

        const filter = filters.find((filter) => filter.id === filterId)
            || whitelistFilters.find((filter) => filter.id === filterId);
        let filterName = '';

        if (filter) {
            filterName = filter.name;
        }

        if (!filterName) {
            filterName = t('unknown_filter', { filterId });
        }

        return filterName;
    };


    const columns = [
        {
            Header: t('time_table_header'),
            accessor: 'time',
            Cell: (row) => getDateCell(row, isDetailed),
            minWidth: 70,
            maxHeight: 60,
            headerClassName: 'logs__text',
        },
        {
            Header: t('request_table_header'),
            accessor: 'domain',
            Cell: (row) => {
                const {
                    isDetailed,
                    autoClients,
                    dnssec_enabled,
                } = props;

                return getDomainCell({
                    row,
                    t,
                    isDetailed,
                    toggleBlocking,
                    autoClients,
                    dnssec_enabled,
                });
            },
            minWidth: 180,
            maxHeight: 60,
            headerClassName: 'logs__text',
        },
        {
            Header: t('response_table_header'),
            accessor: 'response',
            Cell: (row) => getResponseCell(
                row,
                filtering,
                t,
                isDetailed,
                getFilterName,
            ),
            minWidth: 150,
            maxHeight: 60,
            headerClassName: 'logs__text',
        },
        {
            Header: () => {
                const plainSelected = classNames('cursor--pointer', {
                    'icon--selected': !isDetailed,
                });

                const detailedSelected = classNames('cursor--pointer', {
                    'icon--selected': isDetailed,
                });

                return <div className="d-flex justify-content-between">
                    {t('client_table_header')}
                    {<span>
                        <svg
                            className={`icons icon--small icon--active mr-2 cursor--pointer ${plainSelected}`}
                            onClick={() => toggleDetailedLogs(false)}
                        >
                            <title><Trans>compact</Trans></title>
                            <use xlinkHref='#list' />
                        </svg>
                    <svg
                        className={`icons icon--small icon--active cursor--pointer ${detailedSelected}`}
                        onClick={() => toggleDetailedLogs(true)}
                    >
                        <title><Trans>default</Trans></title>
                        <use xlinkHref='#detailed_list' />
                    </svg>
                        </span>}
                </div>;
            },
            accessor: 'client',
            Cell: (row) => {
                const {
                    isDetailed,
                    autoClients,
                    filtering: { processingRules },
                } = props;

                return getClientCell({
                    row,
                    t,
                    isDetailed,
                    toggleBlocking,
                    autoClients,
                    processingRules,
                });
            },
            minWidth: 123,
            maxHeight: 60,
            headerClassName: 'logs__text',
            className: 'pb-0',
        },
    ];

    const changePage = async (page) => {
        setIsLoading(true);

        const { oldest, getLogs, pages } = props;
        const isLastPage = pages && (page + 1 === pages);

        await Promise.all([
            setLogsPage(page),
            setLogsPagination({
                page,
                pageSize: TABLE_DEFAULT_PAGE_SIZE,
            }),
        ].concat(isLastPage ? getLogs(oldest, page) : []));

        setIsLoading(false);
    };

    const tableClass = classNames('logs__table', {
        'logs__table--detailed': isDetailed,
    });

    return (
        <ReactTable
            manual
            minRows={0}
            page={page}
            pages={pages}
            columns={columns}
            filterable={false}
            sortable={false}
            resizable={false}
            data={logs || []}
            loading={isLoading}
            showPageJump={false}
            showPageSizeOptions={false}
            onPageChange={changePage}
            className={tableClass}
            defaultPageSize={TABLE_DEFAULT_PAGE_SIZE}
            loadingText={
                <>
                    <Loading />
                    <h6 className="loading__text">{t('loading_table_status')}</h6>
                </>
            }
            getLoadingProps={() => ({ className: 'loading__container' })}
            rowsText={t('rows_table_footer_text')}
            noDataText={!processingGetLogs
            && <label className="logs__text logs__text--bold">{t('nothing_found')}</label>}
            pageText=''
            ofText=''
            showPagination={logs.length > 0}
            getPaginationProps={() => ({ className: 'custom-pagination custom-pagination--padding' })}
            getTbodyProps={() => ({ className: 'd-block' })}
            previousText={
                <svg className="icons icon--small icon--gray w-100 h-100 cursor--pointer">
                    <title><Trans>previous_btn</Trans></title>
                    <use xlinkHref="#arrow-left" />
                </svg>}
            nextText={
                <svg className="icons icon--small icon--gray w-100 h-100 cursor--pointer">
                    <title><Trans>next_btn</Trans></title>
                    <use xlinkHref="#arrow-right" />
                </svg>}
            renderTotalPagesCount={() => false}
            getTrGroupProps={(_state, rowInfo) => {
                if (!rowInfo) {
                    return {};
                }

                const { reason } = rowInfo.original;
                const colorClass = FILTERED_STATUS_TO_META_MAP[reason] ? FILTERED_STATUS_TO_META_MAP[reason].color : 'white';

                return { className: colorClass };
            }}
            getTrProps={(state, rowInfo) => ({
                className: isDetailed ? 'row--detailed' : '',
                onClick: () => {
                    if (isSmallScreen) {
                        const { dnssec_enabled, autoClients } = props;
                        const {
                            answer_dnssec,
                            client,
                            domain,
                            elapsedMs,
                            info,
                            reason,
                            response,
                            time,
                            tracker,
                            upstream,
                            type,
                            client_proto,
                            filterId,
                            rule,
                            originalResponse,
                        } = rowInfo.original;

                        const hasTracker = !!tracker;

                        const autoClient = autoClients
                            .find((autoClient) => autoClient.name === client);

                        const { whois_info } = info;
                        const country = whois_info?.country;
                        const city = whois_info?.city;
                        const network = whois_info?.orgname;

                        const source = autoClient?.source;

                        const formattedElapsedMs = formatElapsedMs(elapsedMs, t);
                        const isFiltered = checkFiltered(reason);

                        const isBlocked = reason === FILTERED_STATUS.FILTERED_BLACK_LIST
                            || reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;

                        const buttonType = isFiltered ? BLOCK_ACTIONS.UNBLOCK : BLOCK_ACTIONS.BLOCK;
                        const onToggleBlock = () => {
                            toggleBlocking(buttonType, domain);
                        };

                        const isBlockedByResponse = originalResponse.length > 0 && isBlocked;
                        const status = t(isBlockedByResponse ? 'blocked_by_cname_or_ip' : FILTERED_STATUS_TO_META_MAP[reason]?.label || reason);
                        const statusBlocked = <div className="bg--danger">{status}</div>;

                        const protocol = t(SCHEME_TO_PROTOCOL_MAP[client_proto]) || '';

                        const sourceData = getSourceData(tracker);

                        const detailedData = {
                            time_table_header: formatTime(time, LONG_TIME_FORMAT),
                            date: formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS),
                            encryption_status: status,
                            domain,
                            type_table_header: type,
                            protocol,
                            known_tracker: hasTracker && 'title',
                            table_name: tracker?.name,
                            category_label: hasTracker && captitalizeWords(tracker.category),
                            tracker_source: hasTracker && sourceData
                                && <a href={sourceData.url} target="_blank" rel="noopener noreferrer"
                                   className="link--green">{sourceData.name}</a>,
                            response_details: 'title',
                            install_settings_dns: upstream,
                            elapsed: formattedElapsedMs,
                            response_table_header: response?.join('\n'),
                            client_details: 'title',
                            ip_address: client,
                            name: info?.name,
                            country,
                            city,
                            network,
                            source_label: source,
                            validated_with_dnssec: dnssec_enabled ? Boolean(answer_dnssec) : false,
                            [buttonType]: <div onClick={onToggleBlock}
                                               className="title--border bg--danger text-center">{t(buttonType)}</div>,
                        };

                        const { filters, whitelistFilters } = filtering;

                        const filter = getFilterName(filters, whitelistFilters, filterId, t);

                        const detailedDataBlocked = {
                            time_table_header: formatTime(time, LONG_TIME_FORMAT),
                            date: formatDateTime(time, DEFAULT_SHORT_DATE_FORMAT_OPTIONS),
                            encryption_status: statusBlocked,
                            domain,
                            type_table_header: type,
                            protocol,
                            known_tracker: 'title',
                            table_name: tracker?.name,
                            category_label: hasTracker && captitalizeWords(tracker.category),
                            source_label: hasTracker && sourceData
                                && <a href={sourceData.url} target="_blank" rel="noopener noreferrer"
                                   className="link--green">{sourceData.name}</a>,
                            response_details: 'title',
                            install_settings_dns: upstream,
                            elapsed: formattedElapsedMs,
                            filter,
                            rule_label: rule,
                            response_table_header: response?.join('\n'),
                            original_response: originalResponse?.join('\n'),
                            [buttonType]: <div onClick={onToggleBlock}
                                               className="title--border text-center">{t(buttonType)}</div>,
                        };

                        const detailedDataCurrent = isBlocked ? detailedDataBlocked : detailedData;

                        setDetailedDataCurrent(detailedDataCurrent);
                        setButtonType(buttonType);
                        setModalOpened(true);
                    }
                },
            })}
        />
    );
};

Table.propTypes = {
    logs: PropTypes.array.isRequired,
    pages: PropTypes.number.isRequired,
    page: PropTypes.number.isRequired,
    autoClients: PropTypes.array.isRequired,
    defaultPageSize: PropTypes.number,
    oldest: PropTypes.string.isRequired,
    filtering: PropTypes.object.isRequired,
    processingGetLogs: PropTypes.bool.isRequired,
    processingGetConfig: PropTypes.bool.isRequired,
    isDetailed: PropTypes.bool.isRequired,
    setLogsPage: PropTypes.func.isRequired,
    setLogsPagination: PropTypes.func.isRequired,
    getLogs: PropTypes.func.isRequired,
    toggleDetailedLogs: PropTypes.func.isRequired,
    setRules: PropTypes.func.isRequired,
    addSuccessToast: PropTypes.func.isRequired,
    getFilteringStatus: PropTypes.func.isRequired,
    isLoading: PropTypes.bool.isRequired,
    setIsLoading: PropTypes.func.isRequired,
    dnssec_enabled: PropTypes.bool.isRequired,
    setDetailedDataCurrent: PropTypes.func.isRequired,
    setButtonType: PropTypes.func.isRequired,
    setModalOpened: PropTypes.func.isRequired,
    isSmallScreen: PropTypes.bool.isRequired,
};

export default Table;
