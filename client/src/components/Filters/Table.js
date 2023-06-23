import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { withTranslation, Trans } from 'react-i18next';
import CellWrap from '../ui/CellWrap';
import { MODAL_TYPE } from '../../helpers/constants';
import { formatDetailedDateTime } from '../../helpers/helpers';
import { isValidAbsolutePath } from '../../helpers/form';

class Table extends Component {
    getDateCell = (row) => CellWrap(row, formatDetailedDateTime);

    renderCheckbox = ({ original }) => {
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
            Cell: ({ value }) => (
                <div className="logs__row">
                    {isValidAbsolutePath(value) ? value
                        : <a
                            href={value}
                            target="_blank"
                            rel="noopener noreferrer"
                            className="link logs__text"
                        >
                            {value}
                        </a>}
                </div>
            ),
        },
        {
            Header: <Trans>rules_count_table_header</Trans>,
            accessor: 'rulesCount',
            className: 'text-center',
            minWidth: 100,
            Cell: (props) => props.value.toLocaleString(),
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
            Cell: (row) => {
                const { original } = row;
                const { url } = original;
                const { t, toggleFilteringModal, handleDelete } = this.props;

                return (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-primary btn-sm mr-2"
                            title={t('edit_table_action')}
                            onClick={() => toggleFilteringModal({
                                type: MODAL_TYPE.EDIT_FILTERS,
                                url,
                            })
                            }
                        >
                            <svg className="icons icon12">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-secondary btn-sm"
                            onClick={() => handleDelete(url)}
                            title={t('delete_table_action')}
                        >
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
        const {
            loading, filters, t, whitelist,
        } = this.props;

        return (
            <ReactTable
                data={filters}
                columns={this.columns}
                showPagination
                defaultPageSize={10}
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

Table.propTypes = {
    filters: PropTypes.array.isRequired,
    loading: PropTypes.bool.isRequired,
    processingConfigFilter: PropTypes.bool.isRequired,
    toggleFilteringModal: PropTypes.func.isRequired,
    handleDelete: PropTypes.func.isRequired,
    toggleFilter: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
    whitelist: PropTypes.bool,
};

export default withTranslation()(Table);
