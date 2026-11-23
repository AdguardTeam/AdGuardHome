import { createMemo, Show } from 'solid-js';
import { useNavigate } from '@solidjs/router';
import cn from 'clsx';
import intl from 'panel/common/intl';
import { clientFormState, clearClientForm } from 'panel/stores/clientForm';
import { useInitClientFormFromRoute } from 'panel/hooks/useInitClientFormFromRoute';
import { Paths, RoutePath, linkPathBuilder, type RoutePathKey } from 'panel/components/Routes/Paths';
import theme from 'panel/lib/theme';
import { InactivitySchedule } from 'panel/components/BlockedServices/InactivitySchedule/InactivitySchedule';
import s from './ClientSchedule.module.pcss';

import { ClientsHeader } from './ClientsHeader';
import { ClientFormActions } from './ClientFormActions';

export const ClientSchedule = () => {
    useInitClientFormFromRoute();

    const navigate = useNavigate();

    const isEdit = createMemo(() => clientFormState.mode === 'edit');

    const blockedServicesPath = createMemo<RoutePathKey>(() =>
        isEdit() ? RoutePath.ClientsEditBlockedServices : RoutePath.ClientsBlockedServices,
    );

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
        <div class={cn(theme.layout.container, s.containerOverride)}>
            <ClientsHeader
                currentTitle={intl.getMessage('inactivity_schedule')}
                extraLinks={[
                    {
                        path: blockedServicesPath(),
                        title: intl.getMessage('blocked_services'),
                        props: isEdit()
                            ? { clientName: encodeURIComponent(clientFormState.originalName) }
                            : undefined,
                    },
                ]}
            />
            <div class={cn(theme.layout.containerIn, theme.layout.containerIn_one_col)}>
                <InactivitySchedule clientScope />
                <Show when={isEdit()}>
                    <ClientFormActions onCancel={handleCancel} onSaved={handleSaved} />
                </Show>
            </div>
        </div>
    );
};
