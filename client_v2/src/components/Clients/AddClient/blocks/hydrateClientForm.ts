import { createEffect, createSignal, onMount } from 'solid-js';
import { useNavigate, useParams } from '@solidjs/router';

import type { Client } from 'panel/initialState';
import {
    buildFormPayload,
    clearClientForm,
    clientFormState,
    initClientForm,
} from 'panel/stores/clientForm';
import { dashboardState, getClients } from 'panel/stores/dashboard';
import { Paths } from 'panel/components/Routes/Paths';

/**
 * hydrateClientForm fills the client form from the clientName route parameter,
 * so that opening or reloading any page of the form works on its own rather than
 * only after going through the client list.  It also reloads the form when the
 * URL starts naming another client, so that a stale entry is never edited.
 *
 * It does nothing on the add route, which has no such parameter.
 *
 * It returns whether the form describes the client named by the URL, so that a
 * page can keep its controls inert until then.  Acting sooner would be lost, as
 * the hydration replaces the whole form.
 */
export const hydrateClientForm = (): (() => boolean) => {
    const navigate = useNavigate();
    const params = useParams<{ clientName?: string }>();

    // The list held by the store may predate the client named by the URL, so
    // leaving the page is only safe once a fresh list has arrived.
    const [fetched, setFetched] = createSignal(false);

    onMount(async () => {
        await getClients();
        setFetched(true);
    });

    createEffect(() => {
        const urlClientName = params.clientName;
        const clients = dashboardState.clients || [];

        if (!urlClientName) {
            return;
        }

        const decodedName = decodeURIComponent(urlClientName);
        if (clientFormState.mode === 'edit' && clientFormState.originalName === decodedName) {
            // Already editing this client, so keep the unsaved changes.
            return;
        }

        const client = clients.find((c: Client) => c.name === decodedName);
        if (client) {
            initClientForm(buildFormPayload(client));

            return;
        }

        if (fetched()) {
            clearClientForm();
            navigate(Paths.Clients, { replace: true });
        }
    });

    return () => {
        const urlClientName = params.clientName;
        if (!urlClientName) {
            return true;
        }

        return (
            clientFormState.mode === 'edit' &&
            clientFormState.originalName === decodeURIComponent(urlClientName)
        );
    };
};
