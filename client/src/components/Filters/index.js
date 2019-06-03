import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import Modal from '../ui/Modal';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import UserRules from './UserRules';

class Filters extends Component {
    componentDidMount() {
        this.props.getFilteringStatus();
    }

    handleRulesChange = (value) => {
        this.props.handleRulesChange({ userRules: value });
    };

    handleRulesSubmit = () => {
        this.props.setRules(this.props.filtering.userRules);
    };

    renderCheckbox = (row) => {
        const { url } = row.original;
        const { filters } = this.props.filtering;
        const filter = filters.filter(filter => filter.url === url)[0];
        return (
            <label className="checkbox">
                <input type="checkbox" className="checkbox__input" onChange={() => this.props.toggleFilterStatus(filter.url)} checked={filter.enabled}/>
                <span className="checkbox__label"/>
            </label>
        );
    };

    handleDelete = (url) => {
        // eslint-disable-next-line no-alert
        if (window.confirm(this.props.t('filter_confirm_delete'))) {
            this.props.removeFilter({ url });
        }
    }

    columns = [{
        Header: <Trans>enabled_table_header</Trans>,
        accessor: 'enabled',
        Cell: this.renderCheckbox,
        width: 90,
        className: 'text-center',
    }, {
        Header: <Trans>name_table_header</Trans>,
        accessor: 'name',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><span className="logs__text" title={value}>{value}</span></div>),
    }, {
        Header: <Trans>filter_url_table_header</Trans>,
        accessor: 'url',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><a href={value} target='_blank' rel='noopener noreferrer' className="link logs__text">{value}</a></div>),
    }, {
        Header: <Trans>rules_count_table_header</Trans>,
        accessor: 'rulesCount',
        className: 'text-center',
        Cell: props => props.value.toLocaleString(),
    }, {
        Header: <Trans>last_time_updated_table_header</Trans>,
        accessor: 'lastUpdated',
        className: 'text-center',
    }, {
        Header: <Trans>actions_table_header</Trans>,
        accessor: 'url',
        Cell: ({ value }) => (
            <button
                type="button"
                className="btn btn-icon btn-outline-secondary btn-sm"
                onClick={() => this.handleDelete(value)}
                title={this.props.t('delete_table_action')}
            >
                <svg className="icons">
                    <use xlinkHref="#delete" />
                </svg>
            </button>
        ),
        className: 'text-center',
        width: 80,
        sortable: false,
    },
    ];

    render() {
        const { t } = this.props;
        const { filters, userRules, processingRefreshFilters } = this.props.filtering;
        return (
            <div>
                <PageTitle title={ t('filters') } />
                <div className="content">
                    <div className="row">
                        <div className="col-md-12">
                            <Card
                                title={ t('filters_and_hosts') }
                                subtitle={ t('filters_and_hosts_hint') }
                            >
                                <ReactTable
                                    data={filters}
                                    columns={this.columns}
                                    showPagination={true}
                                    defaultPageSize={10}
                                    minRows={4}
                                    // Text
                                    previousText={ t('previous_btn') }
                                    nextText={ t('next_btn') }
                                    loadingText={ t('loading_table_status') }
                                    pageText={ t('page_table_footer_text') }
                                    ofText={ t('of_table_footer_text') }
                                    rowsText={ t('rows_table_footer_text') }
                                    noDataText={ t('no_filters_added') }
                                />
                                <div className="card-actions">
                                    <button
                                        className="btn btn-success btn-standard mr-2"
                                        type="submit"
                                        onClick={this.props.toggleFilteringModal}
                                    >
                                        <Trans>add_filter_btn</Trans>
                                    </button>
                                    <button
                                        className="btn btn-primary btn-standard"
                                        type="submit"
                                        onClick={this.props.refreshFilters}
                                        disabled={processingRefreshFilters}
                                    >
                                        <Trans>check_updates_btn</Trans>
                                    </button>
                                </div>
                            </Card>
                        </div>
                        <div className="col-md-12">
                            <UserRules
                                userRules={userRules}
                                handleRulesChange={this.handleRulesChange}
                                handleRulesSubmit={this.handleRulesSubmit}
                            />
                        </div>
                    </div>
                </div>
                <Modal
                    isOpen={this.props.filtering.isFilteringModalOpen}
                    toggleModal={this.props.toggleFilteringModal}
                    addFilter={this.props.addFilter}
                    isFilterAdded={this.props.filtering.isFilterAdded}
                    processingAddFilter={this.props.filtering.processingAddFilter}
                    title={ t('new_filter_btn') }
                    inputDescription={ t('enter_valid_filter_url') }
                />
            </div>
        );
    }
}

Filters.propTypes = {
    setRules: PropTypes.func,
    getFilteringStatus: PropTypes.func.isRequired,
    filtering: PropTypes.shape({
        userRules: PropTypes.string,
        filters: PropTypes.array,
        isFilteringModalOpen: PropTypes.bool.isRequired,
        isFilterAdded: PropTypes.bool,
        processingAddFilter: PropTypes.bool,
        processingRefreshFilters: PropTypes.bool,
    }),
    removeFilter: PropTypes.func.isRequired,
    toggleFilterStatus: PropTypes.func.isRequired,
    addFilter: PropTypes.func.isRequired,
    toggleFilteringModal: PropTypes.func.isRequired,
    handleRulesChange: PropTypes.func.isRequired,
    refreshFilters: PropTypes.func.isRequired,
    t: PropTypes.func,
};


export default withNamespaces()(Filters);
