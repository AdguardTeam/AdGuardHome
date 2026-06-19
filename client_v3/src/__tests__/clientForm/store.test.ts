import { describe, expect, it } from 'vitest';
import {
    initClientForm,
    updateClientFormField,
    clearClientForm,
    clientFormState,
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
