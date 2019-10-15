import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import escapeRegExp from 'lodash/escapeRegExp';
import endsWith from 'lodash/endsWith';
import { Trans, withNamespaces } from 'react-i18next';
import { HashLink as Link } from 'react-router-hash-link';
import debounce from 'lodash/debounce';

import {
    formatTime,
    formatDateTime,
    isValidQuestionType,
} from '../../helpers/helpers';
import { SERVICES, FILTERED_STATUS, DEBOUNCE_TIMEOUT, DEFAULT_LOGS_FILTER } from '../../helpers/constants';
import { getTrackerData } from '../../helpers/trackers/trackers';
import { formatClientCell } from '../../helpers/formatClientCell';

import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import Loading from '../ui/Loading';
import PopoverFiltered from '../ui/PopoverFilter';
import Popover from '../ui/Popover';
import Tooltip from '../ui/Tooltip';
import './Logs.css';

const TABLE_FIRST_PAGE = 0;
const TABLE_DEFAULT_PAGE_SIZE = 50;
const INITIAL_REQUEST_DATA = ['', DEFAULT_LOGS_FILTER, TABLE_FIRST_PAGE, TABLE_DEFAULT_PAGE_SIZE];
const FILTERED_REASON = 'Filtered';
const RESPONSE_FILTER = {
    ALL: 'all',
    FILTERED: 'filtered',
};

class Logs extends Component {
    componentDidMount() {
        this.getLogs(...INITIAL_REQUEST_DATA);
        this.props.getFilteringStatus();
        this.props.getClients();
        this.props.getLogsConfig();
    }

    getLogs = (lastRowTime, filter, page, pageSize, filtered) => {
        if (this.props.queryLogs.enabled) {
            this.props.getLogs({
                lastRowTime, filter, page, pageSize, filtered,
            });
        }
    };

    refreshLogs = () => {
        window.location.reload();
    };

    handleLogsFiltering = debounce((lastRowTime, filter, page, pageSize, filtered) => {
        this.props.getLogs({
            lastRowTime,
            filter,
            page,
            pageSize,
            filtered,
        });
    }, DEBOUNCE_TIMEOUT);

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

    checkFiltered = reason => reason.indexOf(FILTERED_REASON) === 0;

    checkRewrite = reason => reason === FILTERED_STATUS.REWRITE;

    checkWhiteList = reason => reason === FILTERED_STATUS.NOT_FILTERED_WHITE_LIST;

    getTimeCell = ({ value }) => (
        <div className="logs__row">
            <span className="logs__text" title={formatDateTime(value)}>
                {formatTime(value)}
            </span>
        </div>
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

    getResponseCell = ({ value: responses, original }) => {
        const {
            reason, filterId, rule, status,
        } = original;
        const { t, filtering } = this.props;
        const { filters } = filtering;

        const isFiltered = this.checkFiltered(reason);
        const filterKey = reason.replace(FILTERED_REASON, '');
        const parsedFilteredReason = t('query_log_filtered', { filter: filterKey });
        const isRewrite = this.checkRewrite(reason);
        const isWhiteList = this.checkWhiteList(reason);
        const isBlockedService = reason === FILTERED_STATUS.FILTERED_BLOCKED_SERVICE;
        const currentService = SERVICES.find(service => service.id === original.serviceName);
        const serviceName = currentService && currentService.name;
        let filterName = '';

        if (filterId === 0) {
            filterName = t('custom_filter_rules');
        } else {
            const filterItem = Object.keys(filters).filter(key => filters[key].id === filterId)[0];

            if (typeof filterItem !== 'undefined' && typeof filters[filterItem] !== 'undefined') {
                filterName = filters[filterItem].name;
            }

            if (!filterName) {
                filterName = t('unknown_filter', { filterId });
            }
        }

        return (
            <div className="logs__row logs__row--column">
                <div className="logs__text-wrap">
                    {(isFiltered || isBlockedService) && (
                        <span className="logs__text" title={parsedFilteredReason}>
                            {parsedFilteredReason}
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
                </div>
                <div className="logs__list-wrap">
                    {this.renderResponseList(responses, status)}
                    {isWhiteList && this.renderTooltip(isWhiteList, rule, filterName)}
                </div>
            </div>
        );
    };

    getClientCell = ({ original, value }) => {
        const { dashboard, t } = this.props;
        const { clients, autoClients } = dashboard;
        const { reason, domain } = original;
        const isFiltered = this.checkFiltered(reason);
        const isRewrite = this.checkRewrite(reason);

        return (
            <Fragment>
                <div className="logs__row logs__row--overflow logs__row--column">
                    {formatClientCell(value, clients, autoClients, t)}
                </div>
                {isRewrite ? (
                    <div className="logs__action">
                        <Link to="/dns#rewrites" className="btn btn-sm btn-outline-primary">
                            <Trans>configure</Trans>
                        </Link>
                    </div>
                ) : (
                    this.renderBlockingButton(isFiltered, domain)
                )}
            </Fragment>
        );
    };

    getFilterInput = ({ filter, onChange }) => (
        <Fragment>
            <div className="logs__input-wrap">
                <input
                    type="text"
                    className="form-control"
                    onChange={event => onChange(event.target.value)}
                    value={filter ? filter.value : ''}
                />
                <span className="logs__notice">
                    <Tooltip text={this.props.t('query_log_strict_search')} type='tooltip-custom--logs' />
                </span>
            </div>
        </Fragment>
    );

    getFilters = (filtered) => {
        const filteredObj = filtered.reduce((acc, cur) => ({ ...acc, [cur.id]: cur.value }), {});
        const {
            domain, client, type, response,
        } = filteredObj;

        return {
            filter_domain: domain || '',
            filter_client: client || '',
            filter_question_type: isValidQuestionType(type) ? type.toUpperCase() : '',
            filter_response_status: response === RESPONSE_FILTER.FILTERED ? response : '',
        };
    };

    fetchData = (state) => {
        const { pageSize, page, pages } = state;
        const { allLogs, filter } = this.props.queryLogs;
        const isLastPage = pages && (page + 1 === pages);

        if (isLastPage) {
            const lastRow = allLogs[allLogs.length - 1];
            const lastRowTime = (lastRow && lastRow.time) || '';
            this.getLogs(lastRowTime, filter, page, pageSize, true);
        } else {
            this.props.setLogsPagination({ page, pageSize });
        }
    };

    handleFilterChange = (filtered) => {
        const filters = this.getFilters(filtered);
        this.props.setLogsFilter(filters);
        this.handleLogsFiltering('', filters, TABLE_FIRST_PAGE, TABLE_DEFAULT_PAGE_SIZE, true);
    }

    showTotalPagesCount = (pages) => {
        const { total, isEntireLog } = this.props.queryLogs;
        const showEllipsis = !isEntireLog && total >= 500;

        return (
            <span className="-totalPages">
                {pages || 1}{showEllipsis && '…' }
            </span>
        );
    }

    renderLogs() {
        const { queryLogs, dashboard, t } = this.props;
        const { processingClients } = dashboard;
        const {
            processingGetLogs, processingGetConfig, logs, pages,
        } = queryLogs;
        const isLoading = processingGetLogs || processingClients || processingGetConfig;

        const columns = [
            {
                Header: t('time_table_header'),
                accessor: 'time',
                maxWidth: 100,
                filterable: false,
                Cell: this.getTimeCell,
            },
            {
                Header: t('domain_name_table_header'),
                accessor: 'domain',
                minWidth: 180,
                Cell: this.getDomainCell,
                Filter: this.getFilterInput,
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
                filterMethod: (filter, row) => {
                    if (filter.value === RESPONSE_FILTER.FILTERED) {
                        // eslint-disable-next-line no-underscore-dangle
                        const { reason } = row._original;
                        return this.checkFiltered(reason) || this.checkWhiteList(reason);
                    }
                    return true;
                },
                Filter: ({ filter, onChange }) => (
                    <select
                        className="form-control custom-select"
                        onChange={event => onChange(event.target.value)}
                        value={filter ? filter.value : RESPONSE_FILTER.ALL}
                    >
                        <option value={RESPONSE_FILTER.ALL}>
                            <Trans>show_all_filter_type</Trans>
                        </option>
                        <option value={RESPONSE_FILTER.FILTERED}>
                            <Trans>show_filtered_type</Trans>
                        </option>
                    </select>
                ),
            },
            {
                Header: t('client_table_header'),
                accessor: 'client',
                maxWidth: 240,
                minWidth: 240,
                Cell: this.getClientCell,
                Filter: this.getFilterInput,
            },
        ];

        return (
            <ReactTable
                manual
                filterable
                minRows={5}
                pages={pages}
                columns={columns}
                sortable={false}
                data={logs || []}
                loading={isLoading}
                showPageJump={false}
                onFetchData={this.fetchData}
                onFilteredChange={this.handleFilterChange}
                className="logs__table"
                showPagination={true}
                defaultPageSize={TABLE_DEFAULT_PAGE_SIZE}
                previousText={t('previous_btn')}
                nextText={t('next_btn')}
                loadingText={t('loading_table_status')}
                pageText={t('page_table_footer_text')}
                ofText={t('of_table_footer_text')}
                rowsText={t('rows_table_footer_text')}
                noDataText={t('no_logs_found')}
                renderTotalPagesCount={this.showTotalPagesCount}
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

                    if (this.checkFiltered(reason)) {
                        return {
                            className: 'red',
                        };
                    } else if (this.checkWhiteList(reason)) {
                        return {
                            className: 'green',
                        };
                    } else if (this.checkRewrite(reason)) {
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
        const { enabled, processingGetConfig } = queryLogs;

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
                {enabled && !processingGetConfig && <Card>{this.renderLogs()}</Card>}
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
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Logs);
