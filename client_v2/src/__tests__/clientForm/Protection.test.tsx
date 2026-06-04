import React from 'react';
import { render, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MemoryRouter } from 'react-router-dom';

import { Protection } from 'panel/components/Clients/AddClient/blocks/Protection/Protection';
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

describe('Protection Page', () => {
    beforeEach(() => {
        mocks.state = JSON.parse(JSON.stringify(initialState)) as RootState;
        mocks.state.clientForm = getInitialClientFormState();
        vi.clearAllMocks();
    });

    it('renders protection toggles', () => {
        render(
            <MemoryRouter initialEntries={['/clients/add/protection']}>
                <Protection />
            </MemoryRouter>,
        );
        expect(screen.getByText('Filter requests')).toBeInTheDocument();
        expect(screen.getByText('Browsing security')).toBeInTheDocument();
        expect(screen.getByText('Parental control')).toBeInTheDocument();
        expect(screen.getByText('Safe search')).toBeInTheDocument();
    });

    it('renders logs and statistics section', () => {
        render(
            <MemoryRouter initialEntries={['/clients/add/protection']}>
                <Protection />
            </MemoryRouter>,
        );
        expect(screen.getByText('Logs and statistics')).toBeInTheDocument();
        expect(screen.getByText("Don't log this client")).toBeInTheDocument();
        expect(screen.getByText("Don't collect stats for this client")).toBeInTheDocument();
    });
});
