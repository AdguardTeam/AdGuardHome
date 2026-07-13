import { createMemo } from 'solid-js';
import { useMatch } from '@solidjs/router';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { clientFormState } from 'panel/stores/clientForm';
import { Paths, RoutePath, type RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './ClientsHeader.module.pcss';

type ClientsHeaderProps = {
    currentTitle: string;
    extraLinks?: {
        path: RoutePathKey;
        title: string;
        props?: Partial<Record<string, string | number>>;
    }[];
};

export const ClientsHeader = (props: ClientsHeaderProps) => {
    const extraLinks = () => props.extraLinks || [];
    const isEdit = createMemo(() => clientFormState.mode === 'edit');

    const clientPageLink = createMemo(() =>
        isEdit()
            ? {
                  path: RoutePath.ClientsEdit,
                  title: clientFormState.originalName || intl.getMessage('clients_edit'),
                  props: { clientName: encodeURIComponent(clientFormState.originalName) },
              }
            : {
                  path: RoutePath.ClientsAdd,
                  title: intl.getMessage('clients_add'),
              },
    );

    const isAddMatch = useMatch(() => Paths.ClientsAdd);
    const isEditMatch = useMatch(() => Paths.ClientsEdit);
    const isMainFormPage = createMemo(
        () => isAddMatch() !== undefined || isEditMatch() !== undefined,
    );

    const pageTitle = createMemo(() =>
        isEdit() ? intl.getMessage('clients_edit') : props.currentTitle,
    );

    const breadcrumbCurrent = createMemo(() =>
        isEdit() && isMainFormPage() ? clientFormState.originalName : pageTitle(),
    );

    const parentLinks = createMemo(() => [
        { path: RoutePath.Clients, title: intl.getMessage('client_settings') },
        ...(!isMainFormPage() ? [clientPageLink()] : []),
        ...extraLinks(),
    ]);

    return (
        <div class={s.headerWrapper}>
            <Breadcrumbs parentLinks={parentLinks()} currentTitle={breadcrumbCurrent()} />
            <h1
                class={cn(
                    theme.title.h4,
                    theme.title.h3_tablet,
                    theme.common.textOverflow,
                    s.title,
                )}
                title={pageTitle()}
            >
                {pageTitle()}
            </h1>
        </div>
    );
};
