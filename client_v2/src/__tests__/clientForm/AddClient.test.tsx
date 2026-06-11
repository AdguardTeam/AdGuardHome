import React from 'react';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi, type Mock } from 'vitest';
import { MemoryRouter } from 'react-router-dom';

import { AddClient } from 'panel/components/Clients/AddClient';
import type { RootState } from 'panel/initialState';
import { initialState, getInitialClientFormState } from 'panel/initialState';

const mocks = vi.hoisted(() => ({
    dispatch: vi.fn((action: unknown) => action),
    state: null as unknown as RootState,
}));

vi.mock('react-redux', () => ({
    batch: (fn: () => void) => fn(),
    useDispatch: () => mocks.dispatch,
    useSelector: (selector: (state: RootState) => unknown) => selector(mocks.state),
    shallowEqual: (a: unknown, b: unknown) => a === b,
}));

vi.mock('panel/actions/clientForm', () => ({
    initClientForm: vi.fn(() => ({ type: 'INIT_CLIENT_FORM' })),
    updateClientFormField: vi.fn((payload: unknown) => ({
        type: 'UPDATE_CLIENT_FORM_FIELD',
        payload,
    })),
    clearClientForm: vi.fn(() => ({ type: 'CLEAR_CLIENT_FORM' })),
    saveClient: vi.fn(() => ({ type: 'SAVE_CLIENT' })),
}));

vi.mock('react-router-dom', async () => {
    const actual = await vi.importActual('react-router-dom');
    return {
        ...actual,
        useHistory: () => ({
            push: vi.fn(),
        }),
    };
});

describe('AddClient Main Form', () => {
    beforeEach(() => {
        mocks.state = JSON.parse(JSON.stringify(initialState)) as RootState;
        mocks.state.clientForm = getInitialClientFormState();
        mocks.state.dashboard.supportedTags = ['tag1', 'tag2'];
        (mocks.dispatch as Mock).mockClear();
    });

    it('renders client name input', () => {
        render(
            <MemoryRouter
                future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
                initialEntries={['/clients/add']}
            >
                <AddClient />
            </MemoryRouter>,
        );
        expect(screen.getByPlaceholderText('My device')).toBeInTheDocument();
    });

    it('renders identifier rows with add button', () => {
        render(
            <MemoryRouter
                future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
                initialEntries={['/clients/add']}
            >
                <AddClient />
            </MemoryRouter>,
        );
        expect(screen.getByText('Add identifier')).toBeInTheDocument();
    });

    it('renders Protection and Blocked services navigation links', () => {
        render(
            <MemoryRouter
                future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
                initialEntries={['/clients/add']}
            >
                <AddClient />
            </MemoryRouter>,
        );
        expect(screen.getByText('Protection')).toBeInTheDocument();
        expect(screen.getByText('Blocked services')).toBeInTheDocument();
    });

    it('renders Save and Cancel buttons', () => {
        render(
            <MemoryRouter
                future={{ v7_startTransition: true, v7_relativeSplatPath: true }}
                initialEntries={['/clients/add']}
            >
                <AddClient />
            </MemoryRouter>,
        );
        expect(screen.getByText('Save')).toBeInTheDocument();
        expect(screen.getByText('Cancel')).toBeInTheDocument();
    });
});
