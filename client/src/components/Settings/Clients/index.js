import React, { Component, Fragment } from 'react';
import { withNamespaces } from 'react-i18next';
import PropTypes from 'prop-types';

import ClientsTable from './ClientsTable';
import AutoClients from './AutoClients';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Clients extends Component {
    componentDidMount() {
        this.props.getClients();
        this.props.getStats();
    }

    render() {
        const {
            t,
            dashboard,
            stats,
            clients,
            addClient,
            updateClient,
            deleteClient,
            toggleClientModal,
        } = this.props;

        return (
            <Fragment>
                <PageTitle title={t('client_settings')} />
                {(stats.processingStats || dashboard.processingClients) && <Loading />}
                {!stats.processingStats && !dashboard.processingClients && (
                    <Fragment>
                        <ClientsTable
                            clients={dashboard.clients}
                            topClients={stats.topClients}
                            isModalOpen={clients.isModalOpen}
                            modalClientName={clients.modalClientName}
                            modalType={clients.modalType}
                            addClient={addClient}
                            updateClient={updateClient}
                            deleteClient={deleteClient}
                            toggleClientModal={toggleClientModal}
                            processingAdding={clients.processingAdding}
                            processingDeleting={clients.processingDeleting}
                            processingUpdating={clients.processingUpdating}
                        />
                        <AutoClients
                            autoClients={dashboard.autoClients}
                            topClients={stats.topClients}
                        />
                    </Fragment>
                )}
            </Fragment>
        );
    }
}

Clients.propTypes = {
    t: PropTypes.func.isRequired,
    dashboard: PropTypes.object.isRequired,
    stats: PropTypes.object.isRequired,
    clients: PropTypes.object.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    deleteClient: PropTypes.func.isRequired,
    addClient: PropTypes.func.isRequired,
    updateClient: PropTypes.func.isRequired,
    getClients: PropTypes.func.isRequired,
    getStats: PropTypes.func.isRequired,
};

export default withNamespaces()(Clients);
