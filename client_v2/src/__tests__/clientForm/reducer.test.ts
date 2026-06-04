import { describe, expect, it } from 'vitest';
import { getInitialClientFormState } from 'panel/initialState';
import { initClientForm, updateClientFormField, clearClientForm } from 'panel/actions/clientForm';
import clientForm from 'panel/reducers/clientForm';

describe('clientForm reducer', () => {
    it('returns initial state by default', () => {
        const state = clientForm(undefined, { type: '@@INIT' } as any);
        expect(state.mode).toBe('add');
        expect(state.name).toBe('');
        expect(state.ids).toEqual(['']);
    });

    it('handles initClientForm with null payload (add mode)', () => {
        const state = clientForm(getInitialClientFormState(), initClientForm(null));
        expect(state.mode).toBe('add');
        expect(state.name).toBe('');
    });

    it('handles initClientForm with payload (edit mode)', () => {
        const payload = {
            name: 'My Laptop',
            ids: ['192.168.1.10'],
            filtering_enabled: true,
        };
        const state = clientForm(getInitialClientFormState(), initClientForm(payload));
        expect(state.mode).toBe('edit');
        expect(state.name).toBe('My Laptop');
        expect(state.ids).toEqual(['192.168.1.10']);
        expect(state.filtering_enabled).toBe(true);
        expect(state.originalName).toBe('My Laptop');
    });

    it('handles updateClientFormField', () => {
        const state = clientForm(
            getInitialClientFormState(),
            updateClientFormField({ field: 'name', value: 'Test' }),
        );
        expect(state.name).toBe('Test');
    });

    it('handles nested field update via updateClientFormField', () => {
        const state = clientForm(
            getInitialClientFormState(),
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
            }),
        );
        expect(state.safe_search.enabled).toBe(true);
        expect(state.safe_search.google).toBe(true);
    });

    it('handles clearClientForm', () => {
        const modified = clientForm(
            getInitialClientFormState(),
            updateClientFormField({ field: 'name', value: 'Test' }),
        );
        const cleared = clientForm(modified, clearClientForm());
        expect(cleared.name).toBe('');
        expect(cleared.mode).toBe('add');
    });
});
