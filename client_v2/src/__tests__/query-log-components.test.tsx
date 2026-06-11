import React from 'react';
import { afterEach, beforeEach, describe, expect, test, vi } from 'vitest';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';

import intl from 'panel/common/intl';
import { FILTERED_STATUS } from 'panel/helpers/constants';
import { RequestCell } from 'panel/components/QueryLog/blocks/LogTable/blocks/RequestCell';
import { ClientCell } from 'panel/components/QueryLog/blocks/LogTable/blocks/ClientCell';
import { StatusCell } from 'panel/components/QueryLog/blocks/LogTable/blocks/StatusCell';
import { ReasonCell } from 'panel/components/QueryLog/blocks/LogTable/blocks/ReasonCell';
import { QueryDetailsTooltipContent } from 'panel/components/QueryLog/blocks/LogTable/blocks/QueryDetailsTooltipContent';
import { LogCard } from 'panel/components/QueryLog/blocks/LogCard';
import { DetailModal } from 'panel/components/QueryLog/blocks/DetailModal';
import { EmptyState } from 'panel/components/QueryLog/blocks/EmptyState/EmptyState';
import { InfiniteScrollTrigger } from 'panel/components/QueryLog/blocks/InfiniteScrollTrigger';
import type { LogEntry, Service } from 'panel/components/QueryLog/types';
import type { Filter } from 'panel/helpers/helpers';

const FILTERS: Filter[] = [
    {
        enabled: true,
        id: 1,
        lastUpdated: '',
        name: 'Primary blocklist',
        rulesCount: 1234,
        url: 'https://filters.example/blocklist.txt',
    },
];

const makeLogEntry = (overrides: Partial<LogEntry> = {}): LogEntry => ({
    answer_dnssec: true,
    cached: false,
    client: '192.168.0.10',
    client_id: '',
    client_info: {
        name: 'Office Mac',
        ids: [],
        tags: [],
        whois: { city: 'London', country: 'United Kingdom', orgname: 'Example ISP' },
    },
    client_proto: 'udp',
    domain: 'example.org',
    ecs: '',
    elapsedMs: '17.2',
    filterId: 1,
    originalResponse: [],
    reason: FILTERED_STATUS.FILTERED_BLACK_LIST,
    response: [{ type: 'A', value: '93.184.216.34', ttl: 300 }],
    rule: '||example.org^',
    rules: [{ filter_list_id: 1, text: '||example.org^' }],
    service_name: '',
    serviceName: '',
    status: 'NOERROR',
    time: '2026-05-12T10:00:00.000Z',
    tracker: null,
    type: 'A',
    unicodeName: 'example.org',
    upstream: 'tls://1.1.1.1',
    ...overrides,
});

beforeEach(() => {
    const originalGetComputedStyle = window.getComputedStyle.bind(window);
    const originalGetMessage = intl.getMessage.bind(intl);

    vi.spyOn(window, 'getComputedStyle').mockImplementation((element) =>
        originalGetComputedStyle(element),
    );
    vi.spyOn(intl, 'getMessage').mockImplementation((key, params) => {
        if (key === 'type_value') {
            return `type_value:${String(params?.value ?? '')}`;
        }

        if (key.startsWith('query_log_detail_')) {
            return originalGetMessage(key, params);
        }

        if (key === 'query_log_blocked_services') {
            return 'Blocked services';
        }

        if (key === 'query_log_nothing_available_rotation' && typeof params?.a === 'function') {
            return params.a('open settings');
        }

        if (key === 'query_log_nothing_available') {
            return 'nothing available';
        }

        return String(key);
    });
});

afterEach(() => {
    vi.restoreAllMocks();
});

describe('Query log cells', () => {
    test('render visible content and client search actions', async () => {
        const row = makeLogEntry();
        const ipHandler = vi.fn();
        const nameHandler = vi.fn();
        const onSearchSelect = vi.fn((value: string) =>
            value === row.client ? ipHandler : nameHandler,
        );

        render(
            <div>
                <RequestCell row={row} />
                <ClientCell row={row} onSearchSelect={onSearchSelect} />
                <StatusCell row={row} />
                <ReasonCell row={row} filters={FILTERS} services={[]} whitelistFilters={[]} />
            </div>,
        );

        expect(screen.getByText('example.org')).toBeVisible();
        expect(screen.getByText('type_value:A, plain_dns')).toBeVisible();
        expect(screen.getByRole('button', { name: '192.168.0.10' })).toBeVisible();
        expect(screen.getByRole('button', { name: 'Office Mac' })).toBeVisible();
        expect(screen.getByText('London, United Kingdom')).toBeVisible();
        expect(screen.getByText('query_log_blocked')).toBeVisible();
        expect(screen.getByText('query_log_blocked_by_filter')).toBeVisible();
        expect(screen.getByText('Primary blocklist')).toBeVisible();

        await userEvent.click(screen.getByRole('button', { name: '192.168.0.10' }));
        await userEvent.click(screen.getByRole('button', { name: 'Office Mac' }));

        expect(ipHandler).toHaveBeenCalledTimes(1);
        expect(nameHandler).toHaveBeenCalledTimes(1);
    });
});

describe('Query log composition components', () => {
    test('loads more immediately when the sentinel is already visible', async () => {
        const onLoadMore = vi.fn();

        vi.spyOn(window, 'requestAnimationFrame').mockImplementation(
            (callback: FrameRequestCallback) => {
                return window.setTimeout(() => callback(0), 0);
            },
        );
        vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((frameId: number) => {
            window.clearTimeout(frameId);
        });
        vi.spyOn(HTMLDivElement.prototype, 'getBoundingClientRect').mockReturnValue(
            new DOMRect(0, 100, 300, 20),
        );

        render(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
            />,
        );

        await waitFor(() => {
            expect(onLoadMore).toHaveBeenCalledTimes(1);
        });
    });

    test('loads more again when the reset token changes while the sentinel stays visible', async () => {
        const onLoadMore = vi.fn();

        vi.spyOn(window, 'requestAnimationFrame').mockImplementation(
            (callback: FrameRequestCallback) => {
                return window.setTimeout(() => callback(0), 0);
            },
        );
        vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((frameId: number) => {
            window.clearTimeout(frameId);
        });
        vi.spyOn(HTMLDivElement.prototype, 'getBoundingClientRect').mockReturnValue(
            new DOMRect(0, 100, 300, 20),
        );

        const { rerender } = render(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
                resetToken="few"
            />,
        );

        await waitFor(() => {
            expect(onLoadMore).toHaveBeenCalledTimes(1);
        });

        rerender(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
                resetToken="all"
            />,
        );

        await waitFor(() => {
            expect(onLoadMore).toHaveBeenCalledTimes(2);
        });
    });

    test('does not trigger load when sentinel is hidden (zero-size rect)', async () => {
        const onLoadMore = vi.fn();

        vi.spyOn(window, 'requestAnimationFrame').mockImplementation(
            (callback: FrameRequestCallback) => {
                return window.setTimeout(() => callback(0), 0);
            },
        );
        vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((frameId: number) => {
            window.clearTimeout(frameId);
        });
        // Zero width and height simulates element inside display:none container
        vi.spyOn(HTMLDivElement.prototype, 'getBoundingClientRect').mockReturnValue(
            new DOMRect(0, 0, 0, 0),
        );

        render(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
            />,
        );

        // Wait a tick for the rAF to fire
        await new Promise<void>((resolve) => {
            setTimeout(resolve, 50);
        });

        expect(onLoadMore).not.toHaveBeenCalled();
    });

    test('re-triggers load after disabled cycles back to false (auto-fill)', async () => {
        const onLoadMore = vi.fn();

        vi.spyOn(window, 'requestAnimationFrame').mockImplementation(
            (callback: FrameRequestCallback) => {
                return window.setTimeout(() => callback(0), 0);
            },
        );
        vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((frameId: number) => {
            window.clearTimeout(frameId);
        });
        vi.spyOn(HTMLDivElement.prototype, 'getBoundingClientRect').mockReturnValue(
            new DOMRect(0, 100, 300, 20),
        );

        const { rerender } = render(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
            />,
        );

        await waitFor(() => {
            expect(onLoadMore).toHaveBeenCalledTimes(1);
        });

        // Simulate load in flight
        rerender(<InfiniteScrollTrigger hasMore loading disabled onLoadMore={onLoadMore} />);

        // Load completes, disabled goes back to false
        rerender(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
            />,
        );

        await waitFor(() => {
            expect(onLoadMore).toHaveBeenCalledTimes(2);
        });
    });

    test('stops auto-triggering when sentinel moves out of viewport', async () => {
        const onLoadMore = vi.fn();
        const rectMock = vi.spyOn(HTMLDivElement.prototype, 'getBoundingClientRect');

        vi.spyOn(window, 'requestAnimationFrame').mockImplementation(
            (callback: FrameRequestCallback) => {
                return window.setTimeout(() => callback(0), 0);
            },
        );
        vi.spyOn(window, 'cancelAnimationFrame').mockImplementation((frameId: number) => {
            window.clearTimeout(frameId);
        });
        // Initially in viewport
        rectMock.mockReturnValue(new DOMRect(0, 100, 300, 20));

        const { rerender } = render(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
            />,
        );

        await waitFor(() => {
            expect(onLoadMore).toHaveBeenCalledTimes(1);
        });

        // After load, sentinel has moved far below viewport
        rectMock.mockReturnValue(new DOMRect(0, 2000, 300, 20));

        // Simulate load cycle: disabled → true → false
        rerender(<InfiniteScrollTrigger hasMore loading disabled onLoadMore={onLoadMore} />);
        rerender(
            <InfiniteScrollTrigger
                hasMore
                loading={false}
                disabled={false}
                onLoadMore={onLoadMore}
            />,
        );

        // Wait for rAF to fire
        await new Promise<void>((resolve) => {
            setTimeout(resolve, 50);
        });

        // Should NOT trigger again because sentinel is out of viewport
        expect(onLoadMore).toHaveBeenCalledTimes(1);
    });

    test('render text and actions across composed query log components', () => {
        const row = makeLogEntry({
            answer_dnssec: false,
            client_info: {
                name: 'Living Room TV',
                ids: [],
                tags: [],
                whois: { city: 'Paris', country: 'France', orgname: 'Mobile ISP' },
            },
            client_proto: 'https',
            originalResponse: [{ value: '203.0.113.30', type: 'A', ttl: 300 }],
            reason: FILTERED_STATUS.FILTERED_SAFE_SEARCH,
        });
        const services: Service[] = [];

        render(
            <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <QueryDetailsTooltipContent row={row} />
                <LogCard
                    entry={row}
                    filters={[]}
                    services={services}
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
                <DetailModal
                    entry={row}
                    filters={[]}
                    services={services}
                    whitelistFilters={[]}
                    onClose={vi.fn()}
                    onBlock={vi.fn()}
                    onAddToAllowlist={vi.fn()}
                    onAllowService={vi.fn()}
                />
                <EmptyState mode="rotation-disabled" />
            </MemoryRouter>,
        );

        expect(screen.getAllByText('example.org').length).toBeGreaterThan(0);
        expect(screen.getAllByText(/type_value:A/).length).toBeGreaterThan(0);
        expect(screen.getAllByText('query_log_rewritten').length).toBeGreaterThan(0);
        expect(screen.getAllByText('query_log_safe_search').length).toBeGreaterThan(0);
        expect(screen.getByText('France')).toBeVisible();
        expect(screen.getByRole('button', { name: 'add_to_allowlist' })).toBeVisible();
        expect(screen.getByRole('link', { name: 'open settings' })).toBeVisible();
    });

    test('render blocked-service detail actions and error card summaries', async () => {
        const blockedServiceRow = makeLogEntry({
            answer_dnssec: false,
            client: '192.168.0.60',
            client_info: {
                name: 'Media Console',
                ids: [],
                tags: [],
                whois: { city: 'Prague', country: 'Czechia', orgname: 'Fiber ISP' },
            },
            domain: 'streaming.example',
            reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
            service_name: 'amazon',
            serviceName: '',
            unicodeName: 'streaming.example',
        });
        const errorRow = makeLogEntry({
            answer_dnssec: false,
            client: '192.168.0.40',
            client_info: null,
            domain: 'failed.example',
            originalResponse: [],
            reason: FILTERED_STATUS.NOT_FILTERED_ERROR,
            status: 'SERVFAIL',
            unicodeName: 'failed.example',
        });
        const onClose = vi.fn();
        const onAddToAllowlist = vi.fn();
        const onAllowService = vi.fn();

        render(
            <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <LogCard
                    entry={errorRow}
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
                <DetailModal
                    entry={blockedServiceRow}
                    filters={[]}
                    services={[{ id: 'amazon', name: 'Amazon' }]}
                    whitelistFilters={[]}
                    onClose={onClose}
                    onBlock={vi.fn()}
                    onAddToAllowlist={onAddToAllowlist}
                    onAllowService={onAllowService}
                />
            </MemoryRouter>,
        );

        expect(screen.getByText('failed.example')).toBeVisible();
        expect(screen.getAllByText('error').length).toBeGreaterThan(0);
        expect(screen.getByTestId('query-log-detail-service-name')).toHaveTextContent('Amazon');
        expect(screen.queryByTestId('query-log-detail-action-block')).toBeNull();

        await userEvent.click(screen.getByTestId('query-log-detail-action-allowlist'));
        await userEvent.click(screen.getByTestId('query-log-detail-action-allow-service'));

        expect(onAddToAllowlist).toHaveBeenCalledWith('streaming.example');
        expect(onAllowService).toHaveBeenCalledWith('amazon');
        expect(onClose).toHaveBeenCalledTimes(2);
    });

    test('renders detail rows through tagged translations and composes blocked-service reason details', () => {
        const row = makeLogEntry({
            domain: 'streaming.example',
            reason: FILTERED_STATUS.FILTERED_BLOCKED_SERVICE,
            service_name: 'amazon',
            serviceName: '',
            unicodeName: 'streaming.example',
        });

        render(
            <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <QueryDetailsTooltipContent row={row} />
                <DetailModal
                    entry={row}
                    filters={[]}
                    services={[{ id: 'amazon', name: 'Amazon.com' }]}
                    whitelistFilters={[]}
                    onClose={vi.fn()}
                    onBlock={vi.fn()}
                    onAddToAllowlist={vi.fn()}
                    onAllowService={vi.fn()}
                />
            </MemoryRouter>,
        );

        expect(
            screen.getAllByText(
                (_content, node) => node?.textContent === 'Domain: streaming.example',
            ).length,
        ).toBeGreaterThan(0);
        expect(screen.getByTestId('query-log-detail-reason')).toHaveTextContent(
            'Reason: Blocked services / Amazon.com',
        );
        expect(screen.getByTestId('query-log-detail-response')).toHaveTextContent(
            'Response: A: 93.184.216.34 (ttl=300)',
        );
        expect(screen.queryByText('[object Object]')).not.toBeInTheDocument();
    });

    test('renders the detail modal with direct rich intl detail rows', () => {
        const getMessageSpy = vi.spyOn(intl, 'getMessage');

        render(
            <MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}>
                <DetailModal
                    entry={makeLogEntry({
                        client_info: null,
                        ecs: '',
                        reason: FILTERED_STATUS.REWRITE,
                        rules: [],
                        service_name: '',
                        serviceName: '',
                        tracker: null,
                        upstream: '',
                    })}
                    filters={[]}
                    services={[]}
                    whitelistFilters={[]}
                    onClose={vi.fn()}
                    onBlock={vi.fn()}
                    onAddToAllowlist={vi.fn()}
                    onAllowService={vi.fn()}
                />
            </MemoryRouter>,
        );

        expect(screen.getByTestId('query-log-detail-modal')).toBeVisible();
        expect(screen.getByTestId('query-log-detail-domain')).toHaveTextContent(
            'Domain: example.org',
        );
        expect(getMessageSpy).toHaveBeenCalledWith(
            'query_log_detail_domain',
            expect.objectContaining({
                value: 'example.org',
                span: expect.any(Function),
            }),
        );
    });
});
