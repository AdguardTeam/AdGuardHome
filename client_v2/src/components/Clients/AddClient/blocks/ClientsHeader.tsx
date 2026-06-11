import React from 'react';
import { useSelector } from 'react-redux';
import { useLocation, matchPath } from 'react-router-dom';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { Breadcrumbs } from 'panel/common/ui/Breadcrumbs';
import { RootState } from 'panel/initialState';
import { Paths, RoutePath, RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';

import s from './ClientsHeader.module.pcss';

type ClientsHeaderProps = {
    /** The page after "Add client" / "Edit client" in breadcrumbs. Omit for the main form. */
    currentTitle: string;
    /** Additional breadcrumb segments between the client page and the current page. */
    extraLinks?: {
        path: RoutePathKey;
        title: string;
        props?: Partial<Record<string, string | number>>;
    }[];
};

export const ClientsHeader = ({ currentTitle, extraLinks = [] }: ClientsHeaderProps) => {
    const form = useSelector((state: RootState) => state.clientForm);
    const location = useLocation();
    const isEdit = form.mode === 'edit';

    const clientPageLink = isEdit
        ? {
              path: RoutePath.ClientsEdit,
              title: form.originalName || intl.getMessage('clients_edit'),
              props: { clientName: encodeURIComponent(form.originalName) },
          }
        : {
              path: RoutePath.ClientsAdd,
              title: intl.getMessage('clients_add'),
          };

    const isMainFormPage =
        matchPath(Paths.ClientsAdd, location.pathname) !== null ||
        matchPath(Paths.ClientsEdit, location.pathname) !== null;

    const pageTitle =
        isEdit && isMainFormPage ? form.originalName || intl.getMessage('clients_edit') : currentTitle;

    const parentLinks = [
        { path: RoutePath.Clients, title: intl.getMessage('client_settings') },
        ...(pageTitle !== clientPageLink.title ? [clientPageLink] : []),
        ...extraLinks,
    ];

    return (
        <div className={s.headerWrapper}>
            <Breadcrumbs parentLinks={parentLinks} currentTitle={pageTitle} />
            <h1
                className={cn(
                    theme.title.h4,
                    theme.title.h3_tablet,
                    theme.common.textOverflow,
                    s.title,
                )}
                title={pageTitle}
            >
                {pageTitle}
            </h1>
        </div>
    );
};
