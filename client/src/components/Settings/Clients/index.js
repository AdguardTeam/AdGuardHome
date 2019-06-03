import React, { Component, Fragment } from 'react';
import { withNamespaces } from 'react-i18next';
import PropTypes from 'prop-types';

import ClientsTable from './ClientsTable';
import AutoClients from './AutoClients';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Clients extends Component {
    render() {
        const {
            t,
            dashboard,
            clients,
            addClient,
            updateClient,
            deleteClient,
            toggleClientModal,
        } = this.props;

        return (
            <Fragment>
                <PageTitle title={t('clients_settings')} />
                {(dashboard.processingTopStats || dashboard.processingClients) && <Loading />}
                {!dashboard.processingTopStats && !dashboard.processingClients && (
                    <Fragment>
                        <ClientsTable
                            clients={dashboard.clients}
                            topStats={dashboard.topStats}
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
                            topStats={dashboard.topStats}
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
    clients: PropTypes.object.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    deleteClient: PropTypes.func.isRequired,
    addClient: PropTypes.func.isRequired,
    updateClient: PropTypes.func.isRequired,
    topStats: PropTypes.object,
};

export default withNamespaces()(Clients);
