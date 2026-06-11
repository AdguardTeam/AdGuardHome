import React from 'react';
import { useSelector } from 'react-redux';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { RootState } from 'panel/initialState';
import { RoutePath, RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';
import { InactivitySchedule } from 'panel/components/BlockedServices/InactivitySchedule/InactivitySchedule';
import s from './ClientSchedule.module.pcss';

import { ClientsHeader } from './ClientsHeader';

export const ClientSchedule = () => {
    const form = useSelector((state: RootState) => state.clientForm);
    const isEdit = form.mode === 'edit';

    const blockedServicesPath = isEdit
        ? RoutePath.ClientsEditBlockedServices
        : RoutePath.ClientsBlockedServices;

    return (
        <div className={cn(theme.layout.container, s.containerOverride)}>
            <ClientsHeader
                currentTitle={intl.getMessage('inactivity_schedule')}
                extraLinks={[
                    {
                        path: blockedServicesPath as RoutePathKey,
                        title: intl.getMessage('blocked_services'),
                    },
                ]}
            />
            <div className={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <InactivitySchedule clientScope />
            </div>
        </div>
    );
};
