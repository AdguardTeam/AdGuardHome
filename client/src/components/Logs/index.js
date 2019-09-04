import React, { Component, Fragment } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { saveAs } from 'file-saver/FileSaver';
import escapeRegExp from 'lodash/escapeRegExp';
import endsWith from 'lodash/endsWith';
import { Trans, withNamespaces } from 'react-i18next';
import { HashLink as Link } from 'react-router-hash-link';

import { formatTime, getClientName } from '../../helpers/helpers';
import { SERVICES, FILTERED_STATUS } from '../../helpers/constants';
import { getTrackerData } from '../../helpers/trackers/trackers';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import Loading from '../ui/Loading';
import PopoverFiltered from '../ui/PopoverFilter';
import Popover from '../ui/Popover';
import './Logs.css';

const DOWNLOAD_LOG_FILENAME = 'dns-logs.txt';
const FILTERED_REASON = 'Filtered';
const RESPONSE_FILTER = {
    ALL: 'all',
    FILTERED: 'filtered',
};

class Logs extends Component {
    componentDidMount() {
        this.getLogs();
        this.props.getFilteringStatus();
        this.props.getClients();
    }

    componentDidUpdate(prevProps) {
        // get logs when queryLog becomes enabled
        if (this.props.dashboard.queryLogEnabled && !prevProps.dashboard.queryLogEnabled) {
            this.props.getLogs();
        }
    }

    getLogs = () => {
        // get logs on initialization if queryLogIsEnabled
        if (this.props.dashboard.queryLogEnabled) {
            this.props.getLogs();
        }
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

    checkFiltered = reason => reason.indexOf(FILTERED_REASON) === 0;

    checkRewrite = reason => reason === FILTERED_STATUS.REWRITE;

    checkWhiteList = reason => reason === FILTERED_STATUS.NOT_FILTERED_WHITE_LIST;

    getTimeCell = ({ value }) => (
        <div className="logs__row">
            <span className="logs__text" title={value}>
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

            if (
                typeof filterItem !== 'undefined' &&
                typeof filters[filterItem] !== 'undefined'
            ) {
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
        const { dashboard } = this.props;
        const { reason, domain } = original;
        const isFiltered = this.checkFiltered(reason);
        const isRewrite = this.checkRewrite(reason);
        const clientName =
            getClientName(dashboard.clients, value) || getClientName(dashboard.autoClients, value);
        let client = value;

        if (clientName) {
            client = (
                <span>
                    {clientName} <small>({value})</small>
                </span>
            );
        }

        return (
            <Fragment>
                <div className="logs__row">{client}</div>
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

    renderLogs(logs) {
        const { t } = this.props;
        const columns = [
            {
                Header: t('time_table_header'),
                accessor: 'time',
                maxWidth: 90,
                filterable: false,
                Cell: this.getTimeCell,
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
                filterMethod: (filter, row) => {
                    if (filter.value === RESPONSE_FILTER.FILTERED) {
                        // eslint-disable-next-line no-underscore-dangle
                        const { reason } = row._original;
                        return (
                            this.checkFiltered(reason) ||
                            this.checkWhiteList(reason)
                        );
                    }
                    return true;
                },
                Filter: ({ filter, onChange }) => (
                    <select
                        className="form-control"
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
                maxWidth: 220,
                minWidth: 220,
                Cell: this.getClientCell,
            },
        ];

        if (logs) {
            return (
                <ReactTable
                    className="logs__table"
                    filterable
                    data={logs}
                    columns={columns}
                    showPagination={true}
                    defaultPageSize={50}
                    minRows={7}
                    previousText={t('previous_btn')}
                    nextText={t('next_btn')}
                    loadingText={t('loading_table_status')}
                    pageText={t('page_table_footer_text')}
                    ofText={t('of_table_footer_text')}
                    rowsText={t('rows_table_footer_text')}
                    noDataText={t('no_logs_found')}
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

        return null;
    }

    handleDownloadButton = async (e) => {
        e.preventDefault();
        const data = await this.props.downloadQueryLog();
        const jsonStr = JSON.stringify(data);
        const dataBlob = new Blob([jsonStr], { type: 'text/plain;charset=utf-8' });
        saveAs(dataBlob, DOWNLOAD_LOG_FILENAME);
    };

    renderButtons(queryLogEnabled, logStatusProcessing) {
        if (queryLogEnabled) {
            return (
                <Fragment>
                    <button
                        className="btn btn-gray btn-sm mr-2"
                        type="submit"
                        onClick={() => this.props.toggleLogStatus(queryLogEnabled)}
                        disabled={logStatusProcessing}
                    >
                        <Trans>disabled_log_btn</Trans>
                    </button>
                    <button
                        className="btn btn-primary btn-sm mr-2"
                        type="submit"
                        onClick={this.handleDownloadButton}
                    >
                        <Trans>download_log_file_btn</Trans>
                    </button>
                    <button
                        className="btn btn-outline-primary btn-sm"
                        type="submit"
                        onClick={this.getLogs}
                    >
                        <Trans>refresh_btn</Trans>
                    </button>
                </Fragment>
            );
        }

        return (
            <button
                className="btn btn-success btn-sm mr-2"
                type="submit"
                onClick={() => this.props.toggleLogStatus(queryLogEnabled)}
                disabled={logStatusProcessing}
            >
                <Trans>enabled_log_btn</Trans>
            </button>
        );
    }

    render() {
        const { queryLogs, dashboard, t } = this.props;
        const { queryLogEnabled } = dashboard;
        return (
            <Fragment>
                <PageTitle title={t('query_log')} subtitle={t('last_dns_queries')}>
                    <div className="page-title__actions">
                        {this.renderButtons(queryLogEnabled, dashboard.logStatusProcessing)}
                    </div>
                </PageTitle>
                <Card>
                    {queryLogEnabled &&
                        queryLogs.getLogsProcessing &&
                        dashboard.processingClients && <Loading />}
                    {queryLogEnabled &&
                        !queryLogs.getLogsProcessing &&
                        !dashboard.processingClients &&
                        this.renderLogs(queryLogs.logs)}
                </Card>
            </Fragment>
        );
    }
}

Logs.propTypes = {
    getLogs: PropTypes.func.isRequired,
    queryLogs: PropTypes.object.isRequired,
    dashboard: PropTypes.object.isRequired,
    toggleLogStatus: PropTypes.func.isRequired,
    downloadQueryLog: PropTypes.func.isRequired,
    getFilteringStatus: PropTypes.func.isRequired,
    filtering: PropTypes.object.isRequired,
    setRules: PropTypes.func.isRequired,
    addSuccessToast: PropTypes.func.isRequired,
    getClients: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Logs);
