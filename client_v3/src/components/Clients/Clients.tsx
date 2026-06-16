import { createSignal, createMemo, Show, onMount } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Icon } from 'panel/common/ui/Icon';
import { dashboardState, getClients } from 'panel/stores/dashboard';
import { statsState, getStats } from 'panel/stores/stats';
import { clientsState, deleteClient } from 'panel/stores/clients';
import { servicesState, getAllBlockedServices } from 'panel/stores/services';
import { initClientForm, buildFormPayload } from 'panel/stores/clientForm';
import type { Client } from 'panel/initialState';
import { linkPathBuilder, RoutePath, Paths } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import { PersistentClientsTable } from './blocks/PersistentClientsTable';
import { RuntimeClientsTable } from './blocks/RuntimeClientsTable';
import s from './Clients.module.pcss';

export const Clients = () => {
    const navigate = useNavigate();
    const [clientNameToDelete, setClientNameToDelete] = createSignal('');

    onMount(() => {
        getClients();
        getStats();
        getAllBlockedServices();
    });

    const handleAddClient = () => {
        initClientForm(null);
        navigate(Paths.ClientsAdd);
    };

    const handleEditClient = (client: Client) => {
        initClientForm(buildFormPayload(client));
        navigate(
            linkPathBuilder(RoutePath.ClientsEdit, {
                clientName: encodeURIComponent(client.name),
            }),
        );
    };

    const handleDeleteClient = (name: string) => {
        setClientNameToDelete(name);
    };

    const handleDeleteClose = () => {
        setClientNameToDelete('');
    };

    const handleDeleteConfirm = () => {
        deleteClient(clientNameToDelete());
        handleDeleteClose();
    };

    const isLoading = createMemo(() =>
        dashboardState.processingClients || statsState.processingStats,
    );

    return (
        <div class={theme.layout.container}>
            <div class={theme.layout.containerIn}>
                <div class={s.header}>
                    <h1
                        class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}
                        data-testid="clients-title"
                    >
                        {intl.getMessage('clients')}
                    </h1>

                    <button
                        type="button"
                        onClick={handleAddClient}
                        class={cn(s.button, s.button_add)}
                        data-testid="clients-add-button"
                    >
                        <Icon icon="plus" color="green" />
                        {intl.getMessage('client_add')}
                    </button>
                </div>

                <div class={s.section}>
                    <h2 class={cn(theme.title.h5, theme.title.h4_tablet, s.sectionTitle)}>
                        {intl.getMessage('clients_title')}
                    </h2>
                    <div class={s.desc}>{intl.getMessage('clients_desc')}</div>
                </div>

                <Show when={(dashboardState.clients || []).length > 0}>
                    <div class={cn(s.section, s.section_table)}>
                        <PersistentClientsTable
                            clients={dashboardState.clients || []}
                            normalizedTopClients={(statsState as any).normalizedTopClients}
                            loading={isLoading()}
                            onEdit={handleEditClient}
                            onDelete={handleDeleteClient}
                            deleteDisabled={clientsState.processingDeleting}
                            allServices={servicesState.allServices || []}
                        />
                    </div>
                </Show>

                <div class={s.section}>
                    <h2 class={cn(theme.title.h5, theme.title.h4_tablet, s.sectionTitle)}>
                        {intl.getMessage('auto_clients_title')}
                    </h2>
                    <div class={s.desc}>{intl.getMessage('auto_clients_desc')}</div>
                </div>

                <Show when={(dashboardState.autoClients || []).length > 0}>
                    <div class={cn(s.section, s.section_table)}>
                        <RuntimeClientsTable
                            autoClients={dashboardState.autoClients || []}
                            normalizedTopClients={(statsState as any).normalizedTopClients}
                            loading={isLoading()}
                        />
                    </div>
                </Show>

                <Show when={clientNameToDelete()}>
                    <ConfirmDialog
                        onClose={handleDeleteClose}
                        onConfirm={handleDeleteConfirm}
                        submitDisabled={clientsState.processingDeleting}
                        buttonText={intl.getMessage('remove')}
                        cancelText={intl.getMessage('cancel')}
                        title={intl.getMessage('clients_remove_title')}
                        text={intl.getMessage('clients_remove_desc', { value: clientNameToDelete() })}
                        buttonVariant="danger"
                    />
                </Show>
            </div>
        </div>
    );
};
