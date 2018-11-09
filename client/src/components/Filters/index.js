import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';
import Modal from '../ui/Modal';
import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import UserRules from './UserRules';
import './Filters.css';

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

    columns = [{
        Header: this.props.t('enabled_table_header'),
        accessor: 'enabled',
        Cell: this.renderCheckbox,
        width: 90,
        className: 'text-center',
    }, {
        Header: this.props.t('name_table_header'),
        accessor: 'name',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><span className="logs__text" title={value}>{value}</span></div>),
    }, {
        Header: this.props.t('filter_url_table_header'),
        accessor: 'url',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><a href={value} target='_blank' rel='noopener noreferrer' className="link logs__text">{value}</a></div>),
    }, {
        Header: this.props.t('rules_count_table_header'),
        accessor: 'rulesCount',
        className: 'text-center',
        Cell: props => props.value.toLocaleString(),
    }, {
        Header: this.props.t('last_time_updated_table_header'),
        accessor: 'lastUpdated',
        className: 'text-center',
    }, {
        Header: this.props.t('actions_table_header'),
        accessor: 'url',
        Cell: ({ value }) => (<span title={ this.props.t('delete_table_action') } className='remove-icon fe fe-trash-2' onClick={() => this.props.removeFilter(value)}/>),
        className: 'text-center',
        width: 75,
        sortable: false,
    },
    ];

    render() {
        const { t } = this.props;
        const { filters, userRules } = this.props.filtering;
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
                                    showPagination={false}
                                    noDataText={ t('no_filters_added') }
                                    minRows={4} // TODO find out what to show if rules.length is 0
                                />
                                <div className="card-actions">
                                    <button className="btn btn-success btn-standart mr-2" type="submit" onClick={this.props.toggleFilteringModal}><Trans>add_filter_btn</Trans></button>
                                    <button className="btn btn-primary btn-standart" type="submit" onClick={this.props.refreshFilters}><Trans>cancel_btn</Trans></button>
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
