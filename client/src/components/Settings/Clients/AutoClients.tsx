import React, { Component } from 'react';
import { withTranslation } from 'react-i18next';

// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';

import Card from '../../ui/Card';

import CellWrap from '../../ui/CellWrap';

import whoisCell from './whoisCell';

import LogsSearchLink from '../../ui/LogsSearchLink';

import { sortIp } from '../../../helpers/helpers';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from '../../../helpers/localStorageHelper';
import { TABLES_MIN_ROWS } from '../../../helpers/constants';

const COLUMN_MIN_WIDTH = 200;

interface AutoClientsProps {
    t: (...args: unknown[]) => string;
    autoClients: any[];
    normalizedTopClients: any;
}

class AutoClients extends Component<AutoClientsProps> {
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

            accessor: (row: any) => this.props.normalizedTopClients.auto[row.ip] || 0,
            sortMethod: (a: any, b: any) => b - a,
            id: 'statistics',
            minWidth: COLUMN_MIN_WIDTH,
            Cell: (row: any) => {
                const { value: clientStats } = row;

                if (clientStats) {
                    return (
                        <div className="logs__row">
                            <div className="logs__text" title={clientStats}>
                                <LogsSearchLink search={row.original.ip}>{clientStats}</LogsSearchLink>
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
                bodyType="card-body box-body--settings">
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
                    defaultPageSize={LocalStorageHelper.getItem(LOCAL_STORAGE_KEYS.AUTO_CLIENTS_PAGE_SIZE) || 10}
                    onPageSizeChange={(size: any) =>
                        LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.AUTO_CLIENTS_PAGE_SIZE, size)
                    }
                    minRows={TABLES_MIN_ROWS}
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

export default withTranslation()(AutoClients);
