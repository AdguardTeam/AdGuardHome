import intl from 'panel/common/intl';
import { BreadcrumbLink } from 'panel/common/ui/Breadcrumbs';
import { RoutePath } from 'panel/components/Routes/Paths';

type ClientFormInfo = {
    mode: 'add' | 'edit';
    originalName: string;
} | null;

export const buildClientBreadcrumbs = (
    clientForm: ClientFormInfo,
    extraLinks: BreadcrumbLink[],
): BreadcrumbLink[] => {
    if (!clientForm) {
        return [];
    }

    const isEdit = clientForm.mode === 'edit';
    const clientPageLink: BreadcrumbLink = isEdit
        ? {
              path: RoutePath.ClientsEdit,
              title: intl.getMessage('clients_edit'),
              props: { clientName: encodeURIComponent(clientForm.originalName) },
          }
        : {
              path: RoutePath.ClientsAdd,
              title: intl.getMessage('clients_add'),
          };

    return [
        { path: RoutePath.Clients, title: intl.getMessage('client_settings') },
        clientPageLink,
        ...extraLinks,
    ];
};
