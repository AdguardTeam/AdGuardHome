import React from 'react';
import { useSelector } from 'react-redux';
import intl from 'panel/common/intl';
import { RootState } from 'panel/initialState';
import { RoutePath } from 'panel/components/Routes/Paths';

import { BlockedServices } from 'panel/components/BlockedServices/BlockedServices';

import s from './ClientBlockedServices.module.pcss';

export const ClientBlockedServices = () => {
    const form = useSelector((state: RootState) => state.clientForm);
    const isEdit = form.mode === 'edit';

    const clientPageLink = isEdit
        ? {
              path: RoutePath.ClientsEdit,
              title: form.name || intl.getMessage('clients_edit'),
              props: { clientName: encodeURIComponent(form.originalName) },
          }
        : {
              path: RoutePath.ClientsAdd,
              title: intl.getMessage('clients_add'),
          };

    const breadcrumbs = {
        parentLinks: [
            { path: RoutePath.Clients, title: intl.getMessage('client_settings') },
            clientPageLink,
        ],
        currentTitle: intl.getMessage('blocked_services'),
    };

    return (
        <BlockedServices clientScope breadcrumbs={breadcrumbs} className={s.containerOverride} />
    );
};
