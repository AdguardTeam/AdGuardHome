import React, { useEffect, useCallback, useState } from 'react';
import { batch, useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Icon } from 'panel/common/ui/Icon';
import { getClients } from 'panel/actions';
import { deleteClient } from 'panel/actions/clients';
import { initClientForm, buildFormPayload } from 'panel/actions/clientForm';
import { getStats } from 'panel/actions/stats';
import { getAllBlockedServices } from 'panel/actions/services';
import { Client, RootState } from 'panel/initialState';
import { linkPathBuilder, RoutePath, Paths } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import { PersistentClientsTable } from './blocks/PersistentClientsTable';
import { RuntimeClientsTable } from './blocks/RuntimeClientsTable';
import s from './Clients.module.pcss';

export const Clients = () => {
    const dispatch = useDispatch();
    const [clientNameToDelete, setClientNameToDelete] = useState('');

    const dashboard = useSelector((state: RootState) => state.dashboard);
    const stats = useSelector((state: RootState) => state.stats);
    const clientsState = useSelector((state: RootState) => state.clients);
    const services = useSelector((state: RootState) => state.services);

    const clients = dashboard?.clients || [];
    const autoClients = dashboard?.autoClients || [];
    const processingClients = dashboard?.processingClients || false;
    const processingStats = stats?.processingStats || false;
    const normalizedTopClients = stats?.normalizedTopClients;
    const processingDeleting = clientsState?.processingDeleting || false;
    const allServices = services?.allServices || [];

    useEffect(() => {
        batch(() => {
            dispatch(getClients());
            dispatch(getStats());
            dispatch(getAllBlockedServices());
        });
    }, []);

    const navigate = useNavigate();

    const handleAddClient = useCallback(() => {
        dispatch(initClientForm(null));
        navigate(Paths.ClientsAdd);
    }, [dispatch, navigate]);

    const handleEditClient = useCallback(
        (client: Client) => {
            dispatch(initClientForm(buildFormPayload(client)));
            navigate(
                linkPathBuilder(RoutePath.ClientsEdit, {
                    clientName: encodeURIComponent(client.name),
                }),
            );
        },
        [dispatch, navigate],
    );

    const handleDeleteClient = (name: string) => {
        setClientNameToDelete(name);
    };

    const handleDeleteClose = () => {
        setClientNameToDelete('');
    };

    const handleDeleteConfirm = () => {
        dispatch(deleteClient({ name: clientNameToDelete }));
        handleDeleteClose();
    };

    const isLoading = processingClients || processingStats;

    return (
        <div className={theme.layout.container}>
            <div className={theme.layout.containerIn}>
                <div className={s.header}>
                    <h1
                        className={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}
                        data-testid="clients-title"
                    >
                        {intl.getMessage('clients')}
                    </h1>

                    <button
                        type="button"
                        onClick={handleAddClient}
                        className={cn(s.button, s.button_add)}
                        data-testid="clients-add-button"
                    >
                        <Icon icon="plus" color="green" />
                        {intl.getMessage('client_add')}
                    </button>
                </div>

                <div className={s.section}>
                    <h2 className={cn(theme.title.h5, theme.title.h4_tablet, s.sectionTitle)}>
                        {intl.getMessage('clients_title')}
                    </h2>
                    <div className={s.desc}>{intl.getMessage('clients_desc')}</div>
                </div>

                {clients?.length > 0 && (
                    <div className={cn(s.section, s.section_table)}>
                        <PersistentClientsTable
                            clients={clients}
                            normalizedTopClients={normalizedTopClients}
                            loading={isLoading}
                            onEdit={handleEditClient}
                            onDelete={handleDeleteClient}
                            deleteDisabled={processingDeleting}
                            allServices={allServices}
                        />
                    </div>
                )}

                <div className={s.section}>
                    <h2 className={cn(theme.title.h5, theme.title.h4_tablet, s.sectionTitle)}>
                        {intl.getMessage('auto_clients_title')}
                    </h2>
                    <div className={s.desc}>{intl.getMessage('auto_clients_desc')}</div>
                </div>

                {autoClients?.length > 0 && (
                    <div className={cn(s.section, s.section_table)}>
                        <RuntimeClientsTable
                            autoClients={autoClients}
                            normalizedTopClients={normalizedTopClients}
                            loading={isLoading}
                        />
                    </div>
                )}

                {clientNameToDelete && (
                    <ConfirmDialog
                        onClose={handleDeleteClose}
                        onConfirm={handleDeleteConfirm}
                        submitDisabled={processingDeleting}
                        buttonText={intl.getMessage('remove')}
                        cancelText={intl.getMessage('cancel')}
                        title={intl.getMessage('clients_remove_title')}
                        text={intl.getMessage('clients_remove_desc', { value: clientNameToDelete })}
                        buttonVariant="danger"
                    />
                )}
            </div>
        </div>
    );
};
