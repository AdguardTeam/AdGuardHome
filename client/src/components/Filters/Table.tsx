import React, { Component } from 'react';

// @ts-expect-error FIXME: update react-table
import ReactTable from 'react-table';
import { withTranslation, Trans } from 'react-i18next';

import CellWrap from '../ui/CellWrap';
import { MODAL_TYPE } from '../../helpers/constants';

import { formatDetailedDateTime } from '../../helpers/helpers';

import { isValidAbsolutePath } from '../../helpers/form';
import { LOCAL_STORAGE_KEYS, LocalStorageHelper } from '../../helpers/localStorageHelper';

interface TableProps {
    filters: unknown[];
    loading: boolean;
    processingConfigFilter: boolean;
    toggleFilteringModal: (...args: unknown[]) => unknown;
    handleDelete: (...args: unknown[]) => unknown;
    toggleFilter: (...args: unknown[]) => unknown;
    t: (...args: unknown[]) => string;
    whitelist?: boolean;
}

class Table extends Component<TableProps> {
    getDateCell = (row: any) => CellWrap(row, formatDetailedDateTime);

    renderCheckbox = ({ original }: any) => {
        const { processingConfigFilter, toggleFilter } = this.props;
        const { url, name, enabled } = original;
        const data = { name, url, enabled: !enabled };

        return (
            <label className="checkbox">
                <input
                    type="checkbox"
                    className="checkbox__input"
                    onChange={() => toggleFilter(url, data)}
                    checked={enabled}
                    disabled={processingConfigFilter}
                />

                <span className="checkbox__label" />
            </label>
        );
    };

    columns = [
        {
            Header: <Trans>enabled_table_header</Trans>,
            accessor: 'enabled',
            Cell: this.renderCheckbox,
            width: 90,
            className: 'text-center',
            resizable: false,
        },
        {
            Header: <Trans>name_table_header</Trans>,
            accessor: 'name',
            minWidth: 180,
            Cell: CellWrap,
        },
        {
            Header: <Trans>list_url_table_header</Trans>,
            accessor: 'url',
            minWidth: 180,
            // eslint-disable-next-line react/prop-types
            Cell: ({ value }: any) => (
                <div className="logs__row">
                    {isValidAbsolutePath(value) ? (
                        value
                    ) : (
                        <a href={value} target="_blank" rel="noopener noreferrer" className="link logs__text">
                            {value}
                        </a>
                    )}
                </div>
            ),
        },
        {
            Header: <Trans>rules_count_table_header</Trans>,
            accessor: 'rulesCount',
            className: 'text-center',
            minWidth: 100,
            Cell: (props: any) => props.value.toLocaleString(),
        },
        {
            Header: <Trans>last_time_updated_table_header</Trans>,
            accessor: 'lastUpdated',
            className: 'text-center',
            minWidth: 180,
            Cell: this.getDateCell,
        },
        {
            Header: <Trans>actions_table_header</Trans>,
            accessor: 'actions',
            className: 'text-center',
            width: 100,
            sortable: false,
            resizable: false,
            Cell: (row: any) => {
                const { original } = row;
                const { url } = original;

                const { t, toggleFilteringModal, handleDelete } = this.props;

                return (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-primary btn-sm mr-2"
                            title={t('edit_table_action')}
                            onClick={() =>
                                toggleFilteringModal({
                                    type: MODAL_TYPE.EDIT_FILTERS,
                                    url,
                                })
                            }>
                            <svg className="icons icon12">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>

                        <button
                            type="button"
                            className="btn btn-icon btn-outline-secondary btn-sm"
                            onClick={() => handleDelete(url)}
                            title={t('delete_table_action')}>
                            <svg className="icons icon12">
                                <use xlinkHref="#delete" />
                            </svg>
                        </button>
                    </div>
                );
            },
        },
    ];

    render() {
        const { loading, filters, t, whitelist } = this.props;

        const localStorageKey = whitelist
            ? LOCAL_STORAGE_KEYS.ALLOWLIST_PAGE_SIZE
            : LOCAL_STORAGE_KEYS.BLOCKLIST_PAGE_SIZE;

        return (
            <ReactTable
                data={filters}
                columns={this.columns}
                showPagination
                defaultPageSize={LocalStorageHelper.getItem(localStorageKey) || 10}
                onPageSizeChange={(size: any) => LocalStorageHelper.setItem(localStorageKey, size)}
                loading={loading}
                minRows={6}
                ofText="/"
                previousText={t('previous_btn')}
                nextText={t('next_btn')}
                pageText={t('page_table_footer_text')}
                rowsText={t('rows_table_footer_text')}
                loadingText={t('loading_table_status')}
                noDataText={whitelist ? t('no_whitelist_added') : t('no_blocklist_added')}
            />
        );
    }
}

export default withTranslation()(Table);
