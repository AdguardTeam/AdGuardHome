import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import escapeRegExp from 'lodash/escapeRegExp';
import endsWith from 'lodash/endsWith';
import { Trans, withNamespaces } from 'react-i18next';
import { HashLink as Link } from 'react-router-hash-link';

import {
    formatTime,
    formatDateTime,
    isToday,
    checkFiltered,
    checkRewrite,
    checkRewriteHosts,
    checkWhiteList,
    checkBlackList,
    checkBlockedService,
} from '../../helpers/helpers';
import { SERVICES, TABLE_DEFAULT_PAGE_SIZE, CUSTOM_FILTERING_RULES_ID, FILTERED } from '../../helpers/constants';
import { getTrackerData } from '../../helpers/trackers/trackers';
import { formatClientCell } from '../../helpers/formatClientCell';

import Filters from './Filters';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import Loading from '../ui/Loading';
import PopoverFiltered from '../ui/PopoverFilter';
import Popover from '../ui/Popover';
import './Logs.css';
import CellWrap from '../ui/CellWrap';

const TABLE_FIRST_PAGE = 0;
const INITIAL_REQUEST = true;
const INITIAL_REQUEST_DATA = ['', TABLE_FIRST_PAGE, INITIAL_REQUEST];

class Logs extends Component {
    componentDidMount() {
        this.props.setLogsPage(TABLE_FIRST_PAGE);
        this.getLogs(...INITIAL_REQUEST_DATA);
        this.props.getFilteringStatus();
        this.props.getLogsConfig();
    }

    getLogs = (older_than, page, initial) => {
        if (this.props.queryLogs.enabled) {
            this.props.getLogs({
                older_than, page, pageSize: TABLE_DEFAULT_PAGE_SIZE, initial,
            });
        }
    };

    refreshLogs = () => {
        window.location.reload();
    };

    renderTooltip = (isFiltered, rule, filter, service) =>
        isFiltered && <PopoverFiltered rule={rule} filter={filter} service={service} />;

    renderResponseList = (response, status) => {
        if (response.length > 0) {
            const listItems = response.map((response, index) => (
                <li key={index} title={response} className="logs__list-item">
                    {response}
                </li>
            ));

            return <ul className="list-unstyled">{listItems}</ul>;
        }

        return (
            <div>
                <Trans values={{ value: status }}>query_log_response_status</Trans>
            </div>
        );
    };

    toggleBlocking = (type, domain) => {
        const { userRules } = this.props.filtering;
        const { t } = this.props;
        const lineEnding = !endsWith(userRules, '\n') ? '\n' : '';
        const baseRule = `||${domain}^$important`;
        const baseUnblocking = `@@${baseRule}`;
        const blockingRule = type === 'block' ? baseUnblocking : baseRule;
        const unblockingRule = type === 'block' ? baseRule : baseUnblocking;
        const preparedBlockingRule = new RegExp(`(^|\n)${escapeRegExp(blockingRule)}($|\n)`);
        const preparedUnblockingRule = new RegExp(`(^|\n)${escapeRegExp(unblockingRule)}($|\n)`);

        if (userRules.match(preparedBlockingRule)) {
            this.props.setRules(userRules.replace(`${blockingRule}`, ''));
            this.props.addSuccessToast(`${t('rule_removed_from_custom_filtering_toast')}: ${blockingRule}`);
        } else if (!userRules.match(preparedUnblockingRule)) {
            this.props.setRules(`${userRules}${lineEnding}${unblockingRule}\n`);
            this.props.addSuccessToast(`${t('rule_added_to_custom_filtering_toast')}: ${unblockingRule}`);
        }

        this.props.getFilteringStatus();
    };

    renderBlockingButton(isFiltered, domain) {
        const buttonClass = isFiltered ? 'btn-outline-secondary' : 'btn-outline-danger';
        const buttonText = isFiltered ? 'unblock_btn' : 'block_btn';
        const buttonType = isFiltered ? 'unblock' : 'block';

        return (
            <div className="logs__action">
                <button
                    type="button"
                    className={`btn btn-sm ${buttonClass}`}
                    onClick={() => this.toggleBlocking(buttonType, domain)}
                    disabled={this.props.filtering.processingRules}
                >
                    <Trans>{buttonText}</Trans>
                </button>
            </div>
        );
    }

    getDateCell = row => CellWrap(
        row,
        (isToday(row.value) ? formatTime : formatDateTime),
        formatDateTime,
    );

    getDomainCell = (row) => {
        const response = row.value;
        const trackerData = getTrackerData(response);

        return (
            <div className="logs__row" title={response}>
                <div className="logs__text">{response}</div>
                {trackerData && <Popover data={trackerData} />}
            </div>
        );
    };

    normalizeResponse = response => (
        response.map((response) => {
            const { value, type, ttl } = response;
            return `${type}: ${value} (ttl=${ttl})`;
        })
    );

    getFilterName = (filters, whitelistFilters, filterId, t) => {
        if (filterId === CUSTOM_FILTERING_RULES_ID) {
            return t('custom_filter_rules');
        }

        const filter = filters.find(filter => filter.id === filterId)
            || whitelistFilters.find(filter => filter.id === filterId);
        let filterName = '';

        if (filter) {
            filterName = filter.name;
        }

        if (!filterName) {
            filterName = t('unknown_filter', { filterId });
        }

        return filterName;
    }

    getResponseCell = ({ value: responses, original }) => {
        const {
            reason, filterId, rule, status, originalAnswer,
        } = original;
        const { t, filtering } = this.props;
        const { filters, whitelistFilters } = filtering;

        const isFiltered = checkFiltered(reason);
        const isBlackList = checkBlackList(reason);
        const isRewrite = checkRewrite(reason);
        const isRewriteAuto = checkRewriteHosts(reason);
        const isWhiteList = checkWhiteList(reason);
        const isBlockedService = checkBlockedService(reason);
        const isBlockedCnameIp = originalAnswer;

        const filterKey = reason.replace(FILTERED, '');
        const parsedFilteredReason = t('query_log_filtered', { filter: filterKey });
        const currentService = SERVICES.find(service => service.id === original.serviceName);
        const serviceName = currentService && currentService.name;
        const filterName = this.getFilterName(filters, whitelistFilters, filterId, t);

        if (isBlockedCnameIp) {
            const normalizedAnswer = this.normalizeResponse(originalAnswer);

            return (
                <div className="logs__row logs__row--column">
                    <div className="logs__text-wrap">
                        <span className="logs__text">
                            <Trans>blocked_by_response</Trans>
                        </span>
                        {this.renderTooltip(isFiltered, rule, filterName)}
                    </div>
                    <div className="logs__list-wrap">
                        {this.renderResponseList(normalizedAnswer, status)}
                    </div>
                </div>
            );
        }

        return (
            <div className="logs__row logs__row--column">
                <div className="logs__text-wrap">
                    {(isFiltered || isBlockedService) && !isBlackList && (
                        <span className="logs__text" title={parsedFilteredReason}>
                            {parsedFilteredReason}
                        </span>
                    )}
                    {isBlackList && (
                        <span className="logs__text">
                            <Trans values={{ filter: filterName }}>
                                query_log_filtered
                            </Trans>
                        </span>
                    )}
                    {isBlockedService
                        ? this.renderTooltip(isFiltered, '', '', serviceName)
                        : this.renderTooltip(isFiltered, rule, filterName)}
                    {isRewrite && (
                        <strong>
                            <Trans>rewrite_applied</Trans>
                        </strong>
                    )}
                    {isRewriteAuto && (
                        <span className="logs__text">
                            <strong>
                                <Trans>rewrite_hosts_applied</Trans>
                            </strong>
                        </span>
                    )}
                </div>
                <div className="logs__list-wrap">
                    {this.renderResponseList(responses, status)}
                    {isWhiteList && this.renderTooltip(isWhiteList, rule, filterName)}
                </div>
            </div>
        );
    };

    getClientCell = (row) => {
        const { original } = row;
        const { t } = this.props;
        const { reason, domain } = original;
        const isFiltered = checkFiltered(reason);
        const isRewrite = checkRewrite(reason);
        const isAutoRewrite = checkRewriteHosts(reason);

        if (isAutoRewrite) {
            return (
                <div className="logs__row logs__row--overflow logs__row--column">
                    {formatClientCell(row, t)}
                </div>
            );
        }

        return (
            <Fragment>
                <div className="logs__row logs__row--overflow logs__row--column">
                    {formatClientCell(row, t)}
                </div>
                {isRewrite ? (
                    <div className="logs__action">
                        <Link to="/dns_rewrites" className="btn btn-sm btn-outline-primary">
                            <Trans>configure</Trans>
                        </Link>
                    </div>
                ) : (
                    this.renderBlockingButton(isFiltered, domain)
                )}
            </Fragment>
        );
    };

    fetchData = (state) => {
        const { pages } = state;
        const { oldest, page } = this.props.queryLogs;
        const isLastPage = pages && (page + 1 === pages);

        if (isLastPage) {
            this.getLogs(oldest, page);
        }
    };

    changePage = (page) => {
        this.props.setLogsPage(page);
        this.props.setLogsPagination({ page, pageSize: TABLE_DEFAULT_PAGE_SIZE });
    };

    renderLogs() {
        const { queryLogs, t } = this.props;
        const {
            processingGetLogs, processingGetConfig, logs, pages, page,
        } = queryLogs;
        const isLoading = processingGetLogs || processingGetConfig;

        const columns = [
            {
                Header: t('time_table_header'),
                accessor: 'time',
                minWidth: 105,
                Cell: this.getDateCell,
            },
            {
                Header: t('domain_name_table_header'),
                accessor: 'domain',
                minWidth: 180,
                Cell: this.getDomainCell,
            },
            {
                Header: t('type_table_header'),
                accessor: 'type',
                maxWidth: 60,
            },
            {
                Header: t('response_table_header'),
                accessor: 'response',
                minWidth: 250,
                Cell: this.getResponseCell,
            },
            {
                Header: t('client_table_header'),
                accessor: 'client',
                maxWidth: 240,
                minWidth: 240,
                Cell: this.getClientCell,
            },
        ];

        return (
            <ReactTable
                manual
                minRows={5}
                page={page}
                pages={pages}
                columns={columns}
                filterable={false}
                sortable={false}
                data={logs || []}
                loading={isLoading}
                showPagination={true}
                showPaginationTop={true}
                showPageJump={false}
                showPageSizeOptions={false}
                onFetchData={this.fetchData}
                onPageChange={this.changePage}
                className="logs__table"
                defaultPageSize={TABLE_DEFAULT_PAGE_SIZE}
                previousText={t('previous_btn')}
                nextText={t('next_btn')}
                loadingText={t('loading_table_status')}
                rowsText={t('rows_table_footer_text')}
                noDataText={t('no_logs_found')}
                pageText={''}
                ofText={''}
                renderTotalPagesCount={() => false}
                defaultFilterMethod={(filter, row) => {
                    const id = filter.pivotId || filter.id;
                    return row[id] !== undefined
                        ? String(row[id]).indexOf(filter.value) !== -1
                        : true;
                }}
                defaultSorted={[
                    {
                        id: 'time',
                        desc: true,
                    },
                ]}
                getTrProps={(_state, rowInfo) => {
                    if (!rowInfo) {
                        return {};
                    }

                    const { reason } = rowInfo.original;

                    if (checkFiltered(reason)) {
                        return {
                            className: 'red',
                        };
                    } else if (checkWhiteList(reason)) {
                        return {
                            className: 'green',
                        };
                    } else if (checkRewrite(reason) || checkRewriteHosts(reason)) {
                        return {
                            className: 'blue',
                        };
                    }

                    return {
                        className: '',
                    };
                }}
            />
        );
    }

    render() {
        const { queryLogs, t } = this.props;
        const {
            enabled, processingGetConfig, processingAdditionalLogs, processingGetLogs,
        } = queryLogs;

        const refreshButton = enabled ? (
            <button
                type="button"
                className="btn btn-icon btn-outline-primary btn-sm ml-3"
                onClick={this.refreshLogs}
            >
                <svg className="icons">
                    <use xlinkHref="#refresh" />
                </svg>
            </button>
        ) : (
            ''
        );

        return (
            <Fragment>
                <PageTitle title={t('query_log')}>{refreshButton}</PageTitle>
                {enabled && processingGetConfig && <Loading />}
                {enabled && !processingGetConfig && (
                    <Fragment>
                        <Filters
                            filter={queryLogs.filter}
                            processingGetLogs={processingGetLogs}
                            processingAdditionalLogs={processingAdditionalLogs}
                            setLogsFilter={this.props.setLogsFilter}
                        />
                        <Card>{this.renderLogs()}</Card>
                    </Fragment>
                )}
                {!enabled && !processingGetConfig && (
                    <Card>
                        <div className="lead text-center py-6">
                            <Trans
                                components={[
                                    <Link to="/settings#logs-config" key="0">
                                        link
                                    </Link>,
                                ]}
                            >
                                query_log_disabled
                            </Trans>
                        </div>
                    </Card>
                )}
            </Fragment>
        );
    }
}

Logs.propTypes = {
    getLogs: PropTypes.func.isRequired,
    queryLogs: PropTypes.object.isRequired,
    dashboard: PropTypes.object.isRequired,
    getFilteringStatus: PropTypes.func.isRequired,
    filtering: PropTypes.object.isRequired,
    setRules: PropTypes.func.isRequired,
    addSuccessToast: PropTypes.func.isRequired,
    getClients: PropTypes.func.isRequired,
    getLogsConfig: PropTypes.func.isRequired,
    setLogsPagination: PropTypes.func.isRequired,
    setLogsFilter: PropTypes.func.isRequired,
    setLogsPage: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Logs);
