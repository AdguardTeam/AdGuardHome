import { createEffect, onMount } from 'solid-js';
import { useNavigate, useParams } from '@solidjs/router';

import type { Client } from 'panel/initialState';
import { Paths } from 'panel/components/Routes/Paths';
import {
    buildFormPayload,
    clearClientForm,
    clientFormState,
    initClientForm,
} from 'panel/stores/clientForm';
import { dashboardState, getClients } from 'panel/stores/dashboard';

/**
 * Populates the client form from the `:clientName` route param.
 *
 * The client sub-pages (Protection, Blocked services, Inactivity schedule)
 * render outside the main form, so a direct page load must initialize the
 * store themselves; otherwise they read the form defaults.
 */
export const useInitClientFormFromRoute = () => {
    const navigate = useNavigate();
    const params = useParams<{ clientName?: string }>();

    onMount(() => {
        getClients();
    });

    createEffect(() => {
        const urlClientName = params.clientName;
        const clients = dashboardState.clients || [];

        if (!urlClientName || !clients.length) {
            return;
        }

        const decodedName = decodeURIComponent(urlClientName);

        // Don't clobber a save in flight, and skip when the form already holds
        // this client.  Otherwise re-initialize so Back/Forward between two
        // clients cannot show — and save — the previous client's settings.
        if (clientFormState.processingSave) {
            return;
        }
        if (clientFormState.mode === 'edit' && clientFormState.originalName === decodedName) {
            return;
        }

        const client = clients.find((c: Client) => c.name === decodedName);

        if (!client) {
            clearClientForm();
            navigate(Paths.Clients, { replace: true });
            return;
        }

        initClientForm(buildFormPayload(client));
    });
};
