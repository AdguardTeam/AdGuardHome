import { createMemo } from 'solid-js';
import intl from 'panel/common/intl';
import { clientFormState } from 'panel/stores/clientForm';
import { RoutePath } from 'panel/components/Routes/Paths';

import { BlockedServices } from 'panel/components/BlockedServices/BlockedServices';

import s from './ClientBlockedServices.module.pcss';

export const ClientBlockedServices = () => {
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

    return <BlockedServices clientScope breadcrumbs={breadcrumbs()} class={s.containerOverride} />;
};
