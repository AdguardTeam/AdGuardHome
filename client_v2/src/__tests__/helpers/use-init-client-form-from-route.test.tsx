import { render, screen, waitFor } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const dashboardMocks = vi.hoisted(() => ({
    getClients: vi.fn(),
    dashboardState: {
        clients: [] as unknown[],
        clientsInitialized: true,
        supportedTags: [] as string[],
    },
}));

vi.mock('panel/stores/dashboard', async () => {
    const actual =
        await vi.importActual<typeof import('panel/stores/dashboard')>('panel/stores/dashboard');

    return {
        ...actual,
        getClients: dashboardMocks.getClients,
        get dashboardState() {
            return dashboardMocks.dashboardState;
        },
    };
});

import { useInitClientFormFromRoute } from 'panel/hooks/useInitClientFormFromRoute';
import { clearClientForm, clientFormState, initClientForm } from 'panel/stores/clientForm';

const Probe = () => {
    useInitClientFormFromRoute();

    return <div data-testid="probe" />;
};

const renderAt = (path: string) => {
    const history = createMemoryHistory();
    history.set({ value: path });

    return render(() => (
        <MemoryRouter history={history}>
            <Route path="/clients/edit/:clientName" component={Probe} />
        </MemoryRouter>
    ));
};

const client = {
    name: 'Laptop',
    ids: ['192.168.1.1'],
    use_global_blocked_services: true,
    blocked_services: ['youtube'],
    blocked_services_schedule: {
        time_zone: 'UTC',
        mon: { start: 0, end: 60 },
    },
};

describe('useInitClientFormFromRoute', () => {
    beforeEach(() => {
        clearClientForm();
        dashboardMocks.getClients.mockClear();
        dashboardMocks.dashboardState.clients = [client];
    });

    it('initializes the form from the client in the route', async () => {
        renderAt('/clients/edit/Laptop');

        await waitFor(() => {
            expect(clientFormState.mode).toBe('edit');
        });

        expect(clientFormState.name).toBe('Laptop');
        expect(clientFormState.use_global_blocked_services).toBe(true);
        expect(clientFormState.blocked_services).toEqual(['youtube']);
        expect(clientFormState.blocked_services_schedule.mon).toEqual({ start: 0, end: 60 });
    });

    it('keeps an already initialized form untouched', async () => {
        initClientForm({ name: 'Desktop', use_global_blocked_services: false });
        renderAt('/clients/edit/Laptop');

        await waitFor(() => {
            expect(screen.getByTestId('probe')).toBeInTheDocument();
        });

        expect(clientFormState.name).toBe('Desktop');
    });

    it('loads clients when a client page mounts', () => {
        renderAt('/clients/edit/Laptop');

        expect(dashboardMocks.getClients).toHaveBeenCalled();
    });
});
