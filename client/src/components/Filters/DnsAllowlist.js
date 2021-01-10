import React, { Component } from 'react';
import PropTypes from 'prop-types';
import { withTranslation } from 'react-i18next';

import PageTitle from '../ui/PageTitle';
import Card from '../ui/Card';
import Modal from './Modal';
import Actions from './Actions';
import Table from './Table';

import { MODAL_TYPE } from '../../helpers/constants';
import { getCurrentFilter } from '../../helpers/helpers';

class DnsAllowlist extends Component {
    componentDidMount() {
        this.props.getFilteringStatus();
    }

    handleSubmit = (values) => {
        const { name, url } = values;
        const { filtering } = this.props;
        const whitelist = true;

        if (filtering.modalType === MODAL_TYPE.EDIT_FILTERS) {
            this.props.editFilter(filtering.modalFilterUrl, values, whitelist);
        } else {
            this.props.addFilter(url, name, whitelist);
        }
    };

    handleDelete = (url) => {
        if (window.confirm(this.props.t('list_confirm_delete'))) {
            const whitelist = true;
            this.props.removeFilter(url, whitelist);
        }
    };

    toggleFilter = (url, data) => {
        const whitelist = true;
        this.props.toggleFilterStatus(url, data, whitelist);
    };

    handleRefresh = () => {
        this.props.refreshFilters({ whitelist: true });
    };

    openAddFiltersModal = () => {
        this.props.toggleFilteringModal({ type: MODAL_TYPE.ADD_FILTERS });
    };

    render() {
        const {
            t,
            toggleFilteringModal,
            addFilter,
            toggleFilterStatus,
            filtering: {
                whitelistFilters,
                isModalOpen,
                isFilterAdded,
                processingRefreshFilters,
                processingRemoveFilter,
                processingAddFilter,
                processingConfigFilter,
                processingFilters,
                modalType,
                modalFilterUrl,
            },
        } = this.props;
        const currentFilterData = getCurrentFilter(modalFilterUrl, whitelistFilters);
        const loading = processingConfigFilter
            || processingFilters
            || processingAddFilter
            || processingRemoveFilter
            || processingRefreshFilters;
        const whitelist = true;

        return (
            <>
                <PageTitle
                    title={t('dns_allowlists')}
                    subtitle={t('dns_allowlists_desc')}
                />
                <div className="content">
                    <div className="row">
                        <div className="col-md-12">
                            <Card subtitle={t('filters_and_hosts_hint')}>
                                <Table
                                    filters={whitelistFilters}
                                    loading={loading}
                                    processingConfigFilter={processingConfigFilter}
                                    toggleFilteringModal={toggleFilteringModal}
                                    toggleFilterStatus={toggleFilterStatus}
                                    handleDelete={this.handleDelete}
                                    toggleFilter={this.toggleFilter}
                                    whitelist={whitelist}
                                />
                                <Actions
                                    handleAdd={this.openAddFiltersModal}
                                    handleRefresh={this.handleRefresh}
                                    processingRefreshFilters={processingRefreshFilters}
                                    whitelist={whitelist}
                                />
                            </Card>
                        </div>
                    </div>
                </div>
                <Modal
                    filters={whitelistFilters}
                    isOpen={isModalOpen}
                    toggleFilteringModal={toggleFilteringModal}
                    addFilter={addFilter}
                    isFilterAdded={isFilterAdded}
                    processingAddFilter={processingAddFilter}
                    processingConfigFilter={processingConfigFilter}
                    handleSubmit={this.handleSubmit}
                    modalType={modalType}
                    currentFilterData={currentFilterData}
                    whitelist={whitelist}
                />
            </>
        );
    }
}

DnsAllowlist.propTypes = {
    getFilteringStatus: PropTypes.func.isRequired,
    filtering: PropTypes.object.isRequired,
    removeFilter: PropTypes.func.isRequired,
    toggleFilterStatus: PropTypes.func.isRequired,
    addFilter: PropTypes.func.isRequired,
    toggleFilteringModal: PropTypes.func.isRequired,
    handleRulesChange: PropTypes.func.isRequired,
    refreshFilters: PropTypes.func.isRequired,
    editFilter: PropTypes.func.isRequired,
    t: PropTypes.func.isRequired,
};

export default withTranslation()(DnsAllowlist);
