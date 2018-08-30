import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { saveAs } from 'file-saver/FileSaver';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import Loading from '../ui/Loading';
import { normalizeLogs } from '../../helpers/helpers';


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

    renderLogs(logs) {
        const columns = [{
            Header: 'Time',
            accessor: 'time',
            maxWidth: 150,
        }, {
            Header: 'Domain name',
            accessor: 'domain',
        }, {
            Header: 'Type',
            accessor: 'type',
            maxWidth: 100,
        }, {
            Header: 'Response',
            accessor: 'response',
            Cell: (row) => {
                const responses = row.value;
                if (responses.length > 0) {
                    const liNodes = responses.map((response, index) =>
                        (<li key={index}>{response}</li>));
                    return (<ul className="list-unstyled">{liNodes}</ul>);
                }
                return 'Empty';
            },
        }];

        if (logs) {
            const normalizedLogs = normalizeLogs(logs);
            return (<ReactTable
                data={normalizedLogs}
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
