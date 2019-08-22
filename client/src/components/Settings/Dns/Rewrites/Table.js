import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { withNamespaces } from 'react-i18next';

class Table extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row logs__row--overflow">
            <span className="logs__text" title={value}>
                {value}
            </span>
        </div>
    );

    columns = [
        {
            Header: this.props.t('domain'),
            accessor: 'domain',
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('answer'),
            accessor: 'answer',
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('actions_table_header'),
            accessor: 'actions',
            maxWidth: 100,
            Cell: value => (
                <div className="logs__row logs__row--center">
                    <button
                        type="button"
                        className="btn btn-icon btn-outline-secondary btn-sm"
                        onClick={() =>
                            this.props.handleDelete({
                                answer: value.row.answer,
                                domain: value.row.domain,
                            })
                        }
                        title={this.props.t('delete_table_action')}
                    >
                        <svg className="icons">
                            <use xlinkHref="#delete" />
                        </svg>
                    </button>
                </div>
            ),
        },
    ];

    render() {
        const {
            t, list, processing, processingAdd, processingDelete,
        } = this.props;

        return (
            <ReactTable
                data={list || []}
                columns={this.columns}
                loading={processing || processingAdd || processingDelete}
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
                noDataText={t('rewrite_not_found')}
            />
        );
    }
}

Table.propTypes = {
    t: PropTypes.func.isRequired,
    list: PropTypes.array.isRequired,
    processing: PropTypes.bool.isRequired,
    processingAdd: PropTypes.bool.isRequired,
    processingDelete: PropTypes.bool.isRequired,
    handleDelete: PropTypes.func.isRequired,
};

export default withNamespaces()(Table);
