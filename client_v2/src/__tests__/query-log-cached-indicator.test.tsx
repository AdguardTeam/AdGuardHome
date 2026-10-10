import { render, screen } from '@solidjs/testing-library';
import { describe, expect, it, vi } from 'vitest';

import type { NormalizedQueryLogItem } from 'panel/helpers/helpers';
import { LogCard } from 'panel/components/QueryLog/blocks/LogCard/LogCard';
import { StatusCell } from 'panel/components/QueryLog/blocks/LogTable/blocks/StatusCell';

const makeEntry = (cached: boolean): NormalizedQueryLogItem => ({
    time: '2026-08-02T12:00:00Z',
    domain: 'example.org',
    unicodeName: '',
    type: 'A',
    response: [],
    reason: 'NotFilteredNotFound',
    client: '192.0.2.1',
    client_info: null,
    rules: [],
    originalResponse: [],
    tracker: null,
    elapsedMs: '1',
    cached,
});

describe('query-log cached indicator', () => {
    it('shows cached responses in the desktop status cell', () => {
        render(() => <StatusCell row={makeEntry(true)} />);

        expect(screen.getByText('Cached')).toBeInTheDocument();
    });

    it('does not label uncached desktop responses', () => {
        render(() => <StatusCell row={makeEntry(false)} />);

        expect(screen.queryByText('Cached')).not.toBeInTheDocument();
    });

    it('shows cached responses in the mobile card', () => {
        render(() => (
            <LogCard
                entry={makeEntry(true)}
                filters={[]}
                services={[]}
                whitelistFilters={[]}
                onRowClick={vi.fn()}
                onBlock={vi.fn()}
                onUnblock={vi.fn()}
                onBlockClient={vi.fn()}
                onDisallowClient={vi.fn()}
                onAddPersistentClient={vi.fn()}
                persistentClientIds={[]}
                persistentClientsLoaded
            />
        ));

        expect(screen.getByText('Cached')).toBeInTheDocument();
    });

    it('does not label uncached mobile responses', () => {
        render(() => (
            <LogCard
                entry={makeEntry(false)}
                filters={[]}
                services={[]}
                whitelistFilters={[]}
                onRowClick={vi.fn()}
                onBlock={vi.fn()}
                onUnblock={vi.fn()}
                onBlockClient={vi.fn()}
                onDisallowClient={vi.fn()}
                onAddPersistentClient={vi.fn()}
                persistentClientIds={[]}
                persistentClientsLoaded
            />
        ));

        expect(screen.queryByText('Cached')).not.toBeInTheDocument();
    });
});
