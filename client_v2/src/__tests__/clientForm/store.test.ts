import { describe, expect, it, vi, beforeEach } from 'vitest';

// vi.hoisted ensures the mock is available before module evaluation.
const dashboardMocks = vi.hoisted(() => ({
    getClients: vi.fn(),
    dashboardState: { clients: [] as any[] },
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

import {
    initClientForm,
    updateClientFormField,
    clearClientForm,
    clientFormState,
    saveClient,
} from 'panel/stores/clientForm';

describe('clientForm store', () => {
    it('returns initial state by default', () => {
        clearClientForm();
        expect(clientFormState.mode).toBe('add');
        expect(clientFormState.name).toBe('');
        expect(clientFormState.ids).toEqual(['']);
    });

    it('handles initClientForm with null payload (add mode)', () => {
        initClientForm(null);
        expect(clientFormState.mode).toBe('add');
        expect(clientFormState.name).toBe('');
    });

    it('handles initClientForm with payload (edit mode)', () => {
        const payload = {
            name: 'My Laptop',
            ids: ['192.168.1.10'],
            filtering_enabled: true,
        };
        initClientForm(payload as any);
        expect(clientFormState.mode).toBe('edit');
        expect(clientFormState.name).toBe('My Laptop');
        expect(clientFormState.ids).toEqual(['192.168.1.10']);
        expect(clientFormState.filtering_enabled).toBe(true);
        expect(clientFormState.originalName).toBe('My Laptop');
    });

    it('handles updateClientFormField', () => {
        clearClientForm();
        updateClientFormField({ field: 'name', value: 'Test' });
        expect(clientFormState.name).toBe('Test');
    });

    it('handles nested field update via updateClientFormField', () => {
        clearClientForm();
        updateClientFormField({
            field: 'safe_search',
            value: {
                enabled: true,
                google: true,
                youtube: false,
                bing: false,
                duckduckgo: false,
                yandex: false,
                pixabay: false,
            },
        });
        expect(clientFormState.safe_search.enabled).toBe(true);
        expect(clientFormState.safe_search.google).toBe(true);
    });

    it('handles clearClientForm', () => {
        initClientForm({ name: 'Test' } as any);
        expect(clientFormState.name).toBe('Test');
        clearClientForm();
        expect(clientFormState.name).toBe('');
        expect(clientFormState.mode).toBe('add');
    });
});

describe('saveClient cross-client validation', () => {
    beforeEach(() => {
        clearClientForm();
        dashboardMocks.dashboardState.clients = [];
    });

    it('rejects an identifier that already belongs to another client', async () => {
        // Set up the mocked dashboard store to include a client with a known ID.
        dashboardMocks.dashboardState.clients = [{ name: 'Other', ids: ['192.168.1.50'] } as any];

        updateClientFormField({ field: 'name', value: 'New Client' });
        updateClientFormField({ field: 'ids', value: ['192.168.1.50'] });

        const result = await saveClient();
        expect(result).toBe(false);
        expect(clientFormState.formErrors.ids).toBeDefined();
    });

    it('rejects cache size of 0 when cache is enabled', async () => {
        clearClientForm();
        updateClientFormField({ field: 'name', value: 'Test' });
        updateClientFormField({ field: 'ids', value: ['192.168.1.1'] });
        updateClientFormField({ field: 'upstreams_cache_enabled', value: true });
        updateClientFormField({ field: 'upstreams_cache_size', value: 0 });

        const result = await saveClient();
        expect(result).toBe(false);
        expect(clientFormState.formErrors.upstreams_cache_size).toBeDefined();
    });
});
