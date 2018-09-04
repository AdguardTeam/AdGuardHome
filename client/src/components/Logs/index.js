import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { saveAs } from 'file-saver/FileSaver';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import Loading from '../ui/Loading';
import Tooltip from '../ui/Tooltip';
import './Logs.css';

const DOWNLOAD_LOG_FILENAME = 'dns-logs.txt';

class Logs extends Component {
    componentDidMount() {
        // get logs on initialization if queryLogIsEnabled
        if (this.props.dashboard.queryLogEnabled) {
            this.props.getLogs();
        }
    }

    componentDidUpdate(prevProps) {
        // get logs when queryLog becomes enabled
        if (this.props.dashboard.queryLogEnabled && !prevProps.dashboard.queryLogEnabled) {
            this.props.getLogs();
        }
    }

    renderTooltip(isFiltered, rule) {
        if (rule) {
            return (isFiltered && <Tooltip text={rule}/>);
        }
        return '';
    }

    renderLogs(logs) {
        const columns = [{
            Header: 'Time',
            accessor: 'time',
            maxWidth: 110,
        }, {
            Header: 'Domain name',
            accessor: 'domain',
            Cell: (row) => {
                const response = row.value;

                return (
                    <div className="logs__row logs__row--overflow" title={response}>
                        <div className="logs__text">
                            {response}
                        </div>
                    </div>
                );
            },
        }, {
            Header: 'Type',
            accessor: 'type',
            maxWidth: 60,
        }, {
            Header: 'Response',
            accessor: 'response',
            Cell: (row) => {
                const responses = row.value;
                const isFiltered = row ? row.original.reason.indexOf('Filtered') === 0 : false;
                const rule = row && row.original && row.original.rule;
                if (responses.length > 0) {
                    const liNodes = responses.map((response, index) =>
                        (<li key={index} title={response}>{response}</li>));
                    return (
                        <div className="logs__row">
                            { this.renderTooltip(isFiltered, rule)}
                            <ul className="list-unstyled">{liNodes}</ul>
                        </div>
                    );
                }
                return (
                    <div className="logs__row">
                        { this.renderTooltip(isFiltered, rule) }
                        <span>Empty</span>
                    </div>
                );
            },
        }, {
            Header: 'Client',
            accessor: 'client',
            maxWidth: 250,
        },
        ];

        if (logs) {
            return (<ReactTable
                data={logs}
                columns={columns}
                showPagination={false}
                minRows={7}
                noDataText="No logs found"
                defaultSorted={[
                    {
                        id: 'time',
                        desc: true,
                    },
                ]}
                getTrProps={(_state, rowInfo) => {
                    // highlight filtered requests
                    if (!rowInfo) {
                        return {};
                    }
                    return {
                        className: (rowInfo.original.reason.indexOf('Filtered') === 0 ? 'red' : ''),
                    };
                }}
                />);
        }
        return undefined;
    }

    handleDownloadButton = async (e) => {
        e.preventDefault();
        const data = await this.props.downloadQueryLog();
        const jsonStr = JSON.stringify(data);
        const dataBlob = new Blob([jsonStr], { type: 'text/plain;charset=utf-8' });
        saveAs(dataBlob, DOWNLOAD_LOG_FILENAME);
    };

    renderButtons(queryLogEnabled) {
        return (<div className="card-actions-top">
                <button
                    className="btn btn-success btn-standart mr-2"
                    type="submit"
                    onClick={() => this.props.toggleLogStatus(queryLogEnabled)}
                >{queryLogEnabled ? 'Disable log' : 'Enable log'}</button>
                {queryLogEnabled &&
                <button
                    className="btn btn-primary btn-standart"
                    type="submit"
                    onClick={this.handleDownloadButton}
                >Download log file</button> }
            </div>);
    }

    render() {
        const { queryLogs, dashboard } = this.props;
        const { queryLogEnabled } = dashboard;
        return (
            <div>
                <PageTitle title="Query Log" subtitle="DNS queries log" />
                <Card>
                    {this.renderButtons(queryLogEnabled)}
                    {queryLogEnabled && queryLogs.processing && <Loading />}
                    {queryLogEnabled && !queryLogs.processing &&
                        this.renderLogs(queryLogs.logs)}
                </Card>
            </div>
        );
    }
}

Logs.propTypes = {
    getLogs: PropTypes.func,
    queryLogs: PropTypes.object,
    dashboard: PropTypes.object,
    toggleLogStatus: PropTypes.func,
    downloadQueryLog: PropTypes.func,
};

export default Logs;
