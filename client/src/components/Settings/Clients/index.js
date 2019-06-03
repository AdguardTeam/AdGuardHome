import React, { Component, Fragment } from 'react';
import { withNamespaces } from 'react-i18next';
import PropTypes from 'prop-types';

import ClientsTable from './ClientsTable';
import AutoClients from './AutoClients';
import PageTitle from '../../ui/PageTitle';
import Loading from '../../ui/Loading';

class Clients extends Component {
    render() {
        const { dashboard, clients, t } = this.props;

        return (
            <Fragment>
                <PageTitle title={t('clients_settings')} />
                {!dashboard.processingTopStats || (!dashboard.processingClients && <Loading />)}
                {!dashboard.processingTopStats && !dashboard.processingClients && (
                    <Fragment>
                        <ClientsTable
                            clients={dashboard.clients}
                            topStats={dashboard.topStats}
                            isModalOpen={clients.isModalOpen}
                            modalClientName={clients.modalClientName}
                            modalType={clients.modalType}
                            addClient={this.props.addClient}
                            updateClient={this.props.updateClient}
                            deleteClient={this.props.deleteClient}
                            toggleClientModal={this.props.toggleClientModal}
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
    clients: PropTypes.array.isRequired,
    topStats: PropTypes.object.isRequired,
    toggleClientModal: PropTypes.func.isRequired,
    deleteClient: PropTypes.func.isRequired,
    addClient: PropTypes.func.isRequired,
    updateClient: PropTypes.func.isRequired,
    isModalOpen: PropTypes.bool.isRequired,
    modalType: PropTypes.string.isRequired,
    modalClientName: PropTypes.string.isRequired,
    processingAdding: PropTypes.bool.isRequired,
    processingDeleting: PropTypes.bool.isRequired,
    processingUpdating: PropTypes.bool.isRequired,
};

export default withNamespaces()(Clients);
