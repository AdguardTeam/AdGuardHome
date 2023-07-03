import React, { Component } from 'react';
import PropTypes from 'prop-types';
import ReactTable from 'react-table';
import { withTranslation } from 'react-i18next';
import { sortIp } from '../../../helpers/helpers';
import { MODAL_TYPE } from '../../../helpers/constants';

class Table extends Component {
    cellWrap = ({ value }) => (
        <div className="logs__row o-hidden">
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
            sortMethod: sortIp,
            Cell: this.cellWrap,
        },
        {
            Header: this.props.t('actions_table_header'),
            accessor: 'actions',
            maxWidth: 100,
            sortable: false,
            resizable: false,
            Cell: (value) => {
                const currentRewrite = {
                    answer: value.row.answer,
                    domain: value.row.domain,
                };

                return (
                    <div className="logs__row logs__row--center">
                        <button
                            type="button"
                            className="btn btn-icon btn-outline-primary btn-sm mr-2"
                            onClick={() => {
                                this.props.toggleRewritesModal({
                                    type: MODAL_TYPE.EDIT_REWRITE,
                                    currentRewrite,
                                });
                            }}
                            disabled={this.props.processingUpdate}
                            title={this.props.t('edit_table_action')}
                        >
                            <svg className="icons icon12">
                                <use xlinkHref="#edit" />
                            </svg>
                        </button>

                        <button
                            type="button"
                            className="btn btn-icon btn-outline-secondary btn-sm"
                            onClick={() => this.props.handleDelete(currentRewrite)}
                            title={this.props.t('delete_table_action')}
                        >
                            <svg className="icons">
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
            t, list, processing, processingAdd, processingDelete,
        } = this.props;

        return (
            <ReactTable
                data={list || []}
                columns={this.columns}
                loading={processing || processingAdd || processingDelete}
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
    processingUpdate: PropTypes.bool.isRequired,
    handleDelete: PropTypes.func.isRequired,
    toggleRewritesModal: PropTypes.func.isRequired,
};

export default withTranslation()(Table);
