import { beforeEach, describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen, waitFor } from '@solidjs/testing-library';
import { MemoryRouter, Route, createMemoryHistory } from '@solidjs/router';

const mocks = vi.hoisted(() => ({
    clientsUpdate: vi.fn(),
    clientsAdd: vi.fn(),
    getClients: vi.fn(),
    addSuccessToast: vi.fn(),
    addErrorToast: vi.fn(),
    getAllBlockedServices: vi.fn(),
    servicesState: {
        allServices: [] as unknown[],
        allGroups: [] as unknown[],
        processingAll: false,
        processing: false,
        processingSet: false,
    },
}));

vi.mock('panel/api/generated', () => ({
    clientsUpdate: mocks.clientsUpdate,
    clientsAdd: mocks.clientsAdd,
}));

vi.mock('panel/stores/dashboard', () => ({
    dashboardState: { clients: [], supportedTags: [] },
    getClients: mocks.getClients,
}));

vi.mock('panel/stores/toasts', () => ({
    addSuccessToast: mocks.addSuccessToast,
    addErrorToast: mocks.addErrorToast,
}));

vi.mock('panel/stores/services', () => ({
    getBlockedServices: vi.fn(),
    getAllBlockedServices: mocks.getAllBlockedServices,
    updateBlockedServices: vi.fn(),
    servicesState: mocks.servicesState,
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { Protection } from 'panel/components/Clients/AddClient/blocks/Protection/Protection';
import { ClientBlockedServices } from 'panel/components/Clients/AddClient/blocks/ClientBlockedServices';
import {
    clearClientForm,
    clientFormState,
    initClientForm,
    updateClientFormField,
} from 'panel/stores/clientForm';
import { copy } from 'panel/__tests__/helpers/copy';

const renderProtection = (edit = true) => {
    const history = createMemoryHistory();
    const path = edit ? '/clients/edit/:clientName/protection' : '/clients/add/protection';
    history.set({ value: edit ? '/clients/edit/Test/protection' : path });

    return render(() => (
        <MemoryRouter history={history}>
            <Route path={path} component={() => <Protection />} />
        </MemoryRouter>
    ));
};

const renderClientBlockedServices = () => {
    const history = createMemoryHistory();
    history.set({ value: '/clients/edit/Test/blocked_services' });

    return render(() => (
        <MemoryRouter history={history}>
            <Route
                path="/clients/edit/:clientName/blocked_services"
                component={() => <ClientBlockedServices />}
            />
        </MemoryRouter>
    ));
};

const seedEditForm = () => {
    initClientForm({
        name: 'Test',
        ids: ['1.2.3.4'],
        blocked_services: ['youtube'],
        use_global_blocked_services: false,
    });
};

describe('client sub-page saves', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.clientsUpdate.mockReset();
        clearClientForm();
    });

    it('saves the client draft from the Protection sub-page', async () => {
        seedEditForm();
        mocks.clientsUpdate.mockResolvedValue({});

        renderProtection();
        fireEvent.click(screen.getByTestId('client-form-save'));

        await waitFor(() => expect(mocks.clientsUpdate).toHaveBeenCalledTimes(1));
        expect(mocks.clientsUpdate).toHaveBeenCalledWith({
            name: 'Test',
            data: expect.objectContaining({
                blocked_services: ['youtube'],
                ids: ['1.2.3.4'],
            }),
        });
    });

    it('saves the blocked services toggled on the client sub-page', async () => {
        seedEditForm();
        mocks.clientsUpdate.mockResolvedValue({});

        renderClientBlockedServices();
        fireEvent.click(document.getElementById('use-global-blocked-services') as HTMLElement);
        fireEvent.click(screen.getByTestId('client-form-save'));

        await waitFor(() => expect(mocks.clientsUpdate).toHaveBeenCalledTimes(1));
        expect(mocks.clientsUpdate).toHaveBeenCalledWith({
            name: 'Test',
            data: expect.objectContaining({
                blocked_services: ['youtube'],
                use_global_blocked_services: true,
            }),
        });
    });

    it('keeps the draft and the footer when the save fails', async () => {
        seedEditForm();
        mocks.clientsUpdate.mockRejectedValue(new Error('network'));

        renderProtection();
        fireEvent.click(screen.getByTestId('client-form-save'));

        await waitFor(() => expect(mocks.addErrorToast).toHaveBeenCalledTimes(1));
        expect(clientFormState.mode).toBe('edit');
        expect(clientFormState.name).toBe('Test');
        expect(clientFormState.blocked_services).toEqual(['youtube']);
        expect(screen.getByTestId('client-form-save')).toBeInTheDocument();
    });

    it('surfaces a validation failure from the shared draft', async () => {
        seedEditForm();
        updateClientFormField({ field: 'name', value: '' });

        renderProtection();
        fireEvent.click(screen.getByTestId('client-form-save'));

        await waitFor(() => expect(mocks.addErrorToast).toHaveBeenCalledTimes(1));
        const [payload] = mocks.addErrorToast.mock.calls[0] as [{ error: Error }];
        expect(payload.error.message).toBe(copy('form_error_required'));
        expect(mocks.clientsUpdate).not.toHaveBeenCalled();
    });

    it('renders no save footer in add mode and creates nothing', () => {
        renderProtection(false);

        expect(screen.queryByTestId('client-form-save')).toBeNull();
        expect(screen.queryByTestId('client-form-cancel')).toBeNull();
        expect(mocks.clientsUpdate).not.toHaveBeenCalled();
        expect(mocks.clientsAdd).not.toHaveBeenCalled();
    });
});
