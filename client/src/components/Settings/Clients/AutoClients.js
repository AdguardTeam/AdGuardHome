import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withNamespaces } from 'react-i18next';
import ReactTable from 'react-table';

import Card from '../../ui/Card';

class AutoClients extends Component {
    getStats = (ip, stats) => {
        if (stats) {
            const statsForCurrentIP = stats.find(item => item.name === ip);
            return statsForCurrentIP && statsForCurrentIP.count;
        }

        return '';
    };

    cellWrap = ({ value }) => (
        <div className="logs__row logs__row--overflow">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    columns = [
        {
            Header: this.props.t('table_client'),
            accessor: 'ip',
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('table_name'),
            accessor: 'name',
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('source_label'),
            accessor: 'source',
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('requests_count'),
            accessor: 'statistics',
            Cell: (row) => {
                const clientIP = row.original.ip;
                const clientStats = clientIP && this.getStats(clientIP, this.props.topClients);

                if (clientStats) {
                    return (
                        <div className="logs__row">
                            <div className="logs__text" title={clientStats}>
                                {clientStats}
                            </div>
                        </div>
                    );
                }

                return '–';
            },
        },
    ];

    render() {
        const { t, autoClients } = this.props;

        return (
            <Card
                title={t('auto_clients_title')}
                subtitle={t('auto_clients_desc')}
                bodyType="card-body box-body--settings"
            >
                <ReactTable
                    data={autoClients || []}
                    columns={this.columns}
                    className="-striped -highlight card-table-overflow"
                    showPagination={true}
                    defaultPageSize={10}
                    minRows={5}
                    previousText={t('previous_btn')}
                    nextText={t('next_btn')}
                    loadingText={t('loading_table_status')}
                    pageText={t('page_table_footer_text')}
                    ofText={t('of_table_footer_text')}
                    rowsText={t('rows_table_footer_text')}
                    noDataText={t('clients_not_found')}
                />
            </Card>
        );
    }
}

AutoClients.propTypes = {
    t: PropTypes.func.isRequired,
    autoClients: PropTypes.array.isRequired,
    topClients: PropTypes.array.isRequired,
};

export default withNamespaces()(AutoClients);
