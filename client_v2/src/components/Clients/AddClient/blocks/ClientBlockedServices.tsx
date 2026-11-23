import { createMemo, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import intl from 'panel/common/intl';
import { clientFormState, clearClientForm } from 'panel/stores/clientForm';
import { useInitClientFormFromRoute } from 'panel/hooks/useInitClientFormFromRoute';
import { Paths, RoutePath, linkPathBuilder } from 'panel/components/Routes/Paths';

import { BlockedServices } from 'panel/components/BlockedServices/BlockedServices';

import { ClientFormActions } from './ClientFormActions';

import s from './ClientBlockedServices.module.pcss';

export const ClientBlockedServices = () => {
    useInitClientFormFromRoute();

    const navigate = useNavigate();

    const isEdit = createMemo(() => clientFormState.mode === 'edit');

    const clientPageLink = createMemo(() =>
        isEdit()
            ? {
                  path: RoutePath.ClientsEdit,
                  title: clientFormState.name || intl.getMessage('clients_edit'),
                  props: { clientName: encodeURIComponent(clientFormState.originalName) },
              }
            : {
                  path: RoutePath.ClientsAdd,
                  title: intl.getMessage('clients_add'),
              },
    );

    const breadcrumbs = createMemo(() => ({
        parentLinks: [
            { path: RoutePath.Clients, title: intl.getMessage('client_settings') },
            clientPageLink(),
        ],
        currentTitle: intl.getMessage('blocked_services'),
    }));

    const handleCancel = () => {
        const clientName = clientFormState.originalName;
        clearClientForm();
        navigate(
            linkPathBuilder(RoutePath.ClientsEdit, {
                clientName: encodeURIComponent(clientName),
            }),
        );
    };

    const handleSaved = () => {
        navigate(Paths.Clients);
    };

    return (
        <BlockedServices
            clientScope
            breadcrumbs={breadcrumbs()}
            class={s.containerOverride}
            actions={
                <Show when={isEdit()}>
                    <ClientFormActions onCancel={handleCancel} onSaved={handleSaved} />
                </Show>
            }
        />
    );
};
