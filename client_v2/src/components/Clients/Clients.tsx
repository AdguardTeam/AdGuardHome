import { createSignal, createMemo, Show, onMount } from 'solid-js';
import { useNavigate, useSearchParams } from '@solidjs/router';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { Tabs } from 'panel/common/ui/Tabs';
import { dashboardState, getClients } from 'panel/stores/dashboard';
import { statsState, getStats } from 'panel/stores/stats';
import { clientsState, deleteClient } from 'panel/stores/clients';
import { servicesState, getAllBlockedServices } from 'panel/stores/services';
import { initClientForm, buildFormPayload } from 'panel/stores/clientForm';
import type { Client } from 'panel/initialState';
import { linkPathBuilder, RoutePath, Paths } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';
import type { WebService } from './blocks/PersistentClientsTable/ServiceIcons';

import { PersistentClientsTable } from './blocks/PersistentClientsTable';
import { RuntimeClientsTable } from './blocks/RuntimeClientsTable';
import s from './Clients.module.pcss';
import { PlusButton } from 'panel/common/ui/PlusButton';

const CLIENT_TABS = {
    PERSISTENT: 'persistent',
    RUNTIME: 'runtime',
} as const;

export const Clients = () => {
    const navigate = useNavigate();
    const [clientNameToDelete, setClientNameToDelete] = createSignal('');

    const [searchParams, setSearchParams] = useSearchParams<{ tab?: string }>();

    const activeTab = createMemo(() =>
        searchParams.tab === CLIENT_TABS.RUNTIME ? CLIENT_TABS.RUNTIME : CLIENT_TABS.PERSISTENT,
    );

    const handleTabChange = (tabId: string) => {
        setSearchParams({ tab: tabId }, { replace: true });
    };

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

    const isLoading = createMemo(
        () => dashboardState.processingClients || statsState.processingStats,
    );

    const serviceMap = createMemo(() => {
        const map = new Map<string, WebService>();
        (servicesState.allServices || []).forEach((svc) => {
            map.set(svc.id, svc);
        });
        return map;
    });

    return (
        <div class={theme.layout.container}>
            <div class={theme.layout.containerIn}>
                <h1
                    class={cn(theme.layout.title, theme.title.h4, theme.title.h3_tablet)}
                    data-testid="clients-title"
                >
                    {intl.getMessage('clients')}
                </h1>

                <Tabs
                    activeTab={activeTab()}
                    onTabChange={handleTabChange}
                    class={s.tabs}
                    variant="filled"
                    fullWidth
                    contentClass={s.tabContent}
                    tabs={[
                        {
                            id: CLIENT_TABS.PERSISTENT,
                            label: intl.getMessage('clients_title'),
                            content: (
                                <>
                                    <div class={s.desc}>{intl.getMessage('clients_desc')}</div>

                                    <PlusButton
                                        onClick={handleAddClient}
                                        data-testid="clients-add-button"
                                    >
                                        {intl.getMessage('clients_add')}
                                    </PlusButton>

                                    {dashboardState.clients?.length > 0 && (
                                        <div class={s.tableSection}>
                                            <PersistentClientsTable
                                                clients={dashboardState.clients || []}
                                                normalizedTopClients={
                                                    statsState.normalizedTopClients
                                                }
                                                loading={isLoading()}
                                                onEdit={handleEditClient}
                                                onDelete={handleDeleteClient}
                                                deleteDisabled={clientsState.processingDeleting}
                                                serviceMap={serviceMap()}
                                            />
                                        </div>
                                    )}
                                </>
                            ),
                        },
                        {
                            id: CLIENT_TABS.RUNTIME,
                            label: intl.getMessage('auto_clients_title'),
                            content: (
                                <>
                                    <div class={s.desc}>{intl.getMessage('auto_clients_desc')}</div>

                                    {dashboardState.autoClients?.length > 0 && (
                                        <div class={s.tableSection}>
                                            <RuntimeClientsTable
                                                autoClients={dashboardState.autoClients || []}
                                                normalizedTopClients={
                                                    statsState.normalizedTopClients
                                                }
                                                loading={isLoading()}
                                            />
                                        </div>
                                    )}
                                </>
                            ),
                        },
                    ]}
                />

                <Show when={clientNameToDelete()}>
                    <ConfirmDialog
                        onClose={handleDeleteClose}
                        onConfirm={handleDeleteConfirm}
                        submitDisabled={clientsState.processingDeleting}
                        buttonText={intl.getMessage('yes_remove')}
                        cancelText={intl.getMessage('cancel')}
                        title={intl.getMessage('clients_remove_title')}
                        text={intl.getMessage('clients_remove_desc', {
                            value: clientNameToDelete(),
                        })}
                        buttonVariant="danger"
                    />
                </Show>
            </div>
        </div>
    );
};
