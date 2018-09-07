import React, { Component } from 'react';
import ReactTable from 'react-table';
import PropTypes from 'prop-types';
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
        Header: 'Enabled',
        accessor: 'enabled',
        Cell: this.renderCheckbox,
        width: 90,
        className: 'text-center',
    }, {
        Header: 'Filter name',
        accessor: 'name',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><span className="logs__text" title={value}>{value}</span></div>),
    }, {
        Header: 'Host file URL',
        accessor: 'url',
        Cell: ({ value }) => (<div className="logs__row logs__row--overflow"><a href={value} target='_blank' rel='noopener noreferrer' className="link logs__text">{value}</a></div>),
    }, {
        Header: 'Rules count',
        accessor: 'rulesCount',
        className: 'text-center',
    }, {
        Header: 'Last time update',
        accessor: 'lastUpdated',
        className: 'text-center',
    }, {
        Header: 'Actions',
        accessor: 'url',
        Cell: ({ value }) => (<span className='remove-icon fe fe-trash-2' onClick={() => this.props.removeFilter(value)}/>),
        className: 'text-center',
        width: 75,
        sortable: false,
    },
    ];

    render() {
        const { filters, userRules } = this.props.filtering;
        return (
            <div>
                <PageTitle title="Filters" />
                <div className="content">
                    <div className="row">
                        <div className="col-md-12">
                            <Card
                                title="Blocking filters and hosts files"
                                subtitle="AdGuard DNS understands basic adblock rules and hosts files syntax."
                            >
                                <ReactTable
                                    data={filters}
                                    columns={this.columns}
                                    showPagination={false}
                                    noDataText="No filters added"
                                    minRows={4} // TODO find out what to show if rules.length is 0
                                />
                                <div className="card-actions">
                                    <button className="btn btn-success btn-standart mr-2" type="submit" onClick={this.props.toggleFilteringModal}>Add filter</button>
                                    <button className="btn btn-primary btn-standart" type="submit" onClick={this.props.refreshFilters}>Check updates</button>
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
                    title="New filter subscription"
                    inputDescription="Enter valid URL or file path of the filter into field above. You will be subscribed to that filter."
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
};


export default Filters;
