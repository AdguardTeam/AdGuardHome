import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';
import ReactTable from 'react-table';

import Card from '../../ui/Card';
import CellWrap from '../../ui/CellWrap';

import whoisCell from './whoisCell';
import LogsSearchLink from '../../ui/LogsSearchLink';
import { sortIp } from '../../../helpers/helpers';

const COLUMN_MIN_WIDTH = 200;

class AutoClients extends Component {
    columns = [
        {
            Header: this.props.t('table_client'),
            accessor: 'ip',
            minWidth: COLUMN_MIN_WIDTH,
            Cell: CellWrap,
            sortMethod: sortIp,
        },
        {
            Header: this.props.t('table_name'),
            accessor: 'name',
            minWidth: COLUMN_MIN_WIDTH,
            Cell: CellWrap,
        },
        {
            Header: this.props.t('source_label'),
            accessor: 'source',
            minWidth: COLUMN_MIN_WIDTH,
            Cell: CellWrap,
        },
        {
            Header: this.props.t('whois'),
            accessor: 'whois_info',
            minWidth: COLUMN_MIN_WIDTH,
            Cell: whoisCell(this.props.t),
        },
        {
            Header: this.props.t('requests_count'),
            accessor: (row) => this.props.normalizedTopClients.auto[row.ip] || 0,
            sortMethod: (a, b) => b - a,
            id: 'statistics',
            minWidth: COLUMN_MIN_WIDTH,
            Cell: (row) => {
                const { value: clientStats } = row;

                if (clientStats) {
                    return (
                        <div className="logs__row">
                            <div className="logs__text" title={clientStats}>
                                <LogsSearchLink search={row.original.ip}>
                                    {clientStats}
                                </LogsSearchLink>
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
                    defaultSorted={[
                        {
                            id: 'statistics',
                            asc: true,
                        },
                    ]}
                    className="-striped -highlight card-table-overflow"
                    showPagination
                    defaultPageSize={10}
                    minRows={5}
                    ofText="/"
                    previousText={t('previous_btn')}
                    nextText={t('next_btn')}
                    pageText={t('page_table_footer_text')}
                    rowsText={t('rows_table_footer_text')}
                    loadingText={t('loading_table_status')}
                    noDataText={t('clients_not_found')}
                />
            </Card>
        );
    }
}

AutoClients.propTypes = {
    t: PropTypes.func.isRequired,
    autoClients: PropTypes.array.isRequired,
    normalizedTopClients: PropTypes.object.isRequired,
};

export default withTranslation()(AutoClients);
