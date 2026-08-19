import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, fireEvent, waitFor } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';

import { TopClients } from 'panel/components/Dashboard/blocks/TopClients';
import { setMatchMedia } from './helpers/matchMedia';

import type { ClientFindSubEntry } from 'panel/api/model/clientFindSubEntry';

const { disallowedClients } = vi.hoisted(() => ({
    disallowedClients: { value: '' },
}));

vi.mock('panel/stores/access', () => ({
    accessState: {
        get disallowed_clients() {
            return disallowedClients.value;
        },
    },
    getAccessList: vi.fn(),
    setAccessList: vi.fn(),
    toggleClientBlock: vi.fn(),
}));

const client = (name: string, count: number, info?: ClientFindSubEntry) => ({
    name,
    count,
    info: info ?? {},
});

const renderTopClients = (topClients: ReturnType<typeof client>[], numDnsQueries: number) =>
    render(() => (
        <HashRouter>
            <Route
                path="/"
                component={() => (
                    <TopClients topClients={topClients} numDnsQueries={numDnsQueries} />
                )}
            />
        </HashRouter>
    ));

const rowsOf = (container: HTMLElement) =>
    Array.from(container.querySelectorAll('[data-testid="top-client-row"]'));

const iconHrefsOf = (row: Element) =>
    Array.from(row.querySelectorAll('use')).map((use) => use.getAttribute('href'));

describe('TopClients', () => {
    afterEach(() => {
        disallowedClients.value = '';
    });

    it('displays only 4 clients on desktop', () => {
        setMatchMedia(true);
        const clients = Array.from({ length: 6 }, (_, i) => client(`192.168.0.${i}`, 100 - i));
        const { container } = renderTopClients(clients, 600);

        expect(rowsOf(container)).toHaveLength(4);
    });

    it('displays only 4 clients on mobile', () => {
        setMatchMedia(false);
        const clients = Array.from({ length: 6 }, (_, i) => client(`192.168.0.${i}`, 100 - i));
        const { container } = renderTopClients(clients, 600);

        expect(rowsOf(container)).toHaveLength(4);
    });

    it('uses the wifi icon for clients and wifi_protect for blocked ones', () => {
        setMatchMedia(true);
        disallowedClients.value = '10.0.0.4';
        const clients = [
            client('10.0.0.1', 100, { name: 'Living Room' }),
            client('10.0.0.2', 90, { name: 'Bedroom' }),
            client('10.0.0.3', 80, { name: 'Kitchen' }),
            client('10.0.0.4', 70, { name: 'Smart TV' }),
        ];
        const { container } = renderTopClients(clients, 400);
        const rows = rowsOf(container);

        const blockedRow = rows.find((row) => row.textContent?.includes('10.0.0.4'));
        expect(blockedRow).toBeDefined();
        expect(iconHrefsOf(blockedRow!)).toContain('#wifi_protect');
        expect(iconHrefsOf(blockedRow!)).not.toContain('#wifi');

        // Blocked state is communicated by the danger icon and the client
        // details tooltip (covered by the tooltip tests below), not by an
        // inline badge.

        const normalRow = rows.find((row) => row.textContent?.includes('10.0.0.1'));
        expect(normalRow).toBeDefined();
        expect(iconHrefsOf(normalRow!)).toContain('#wifi');
        expect(iconHrefsOf(normalRow!)).not.toContain('#wifi_protect');
    });

    it('renders "N/A" for clients without info and the name for known ones', () => {
        setMatchMedia(true);
        const clients = [
            client('192.168.1.10', 100, { name: 'Known client' }),
            client('192.168.1.11', 90),
            client('192.168.1.12', 80),
            client('192.168.1.13', 70),
        ];
        const { container } = renderTopClients(clients, 400);
        const rows = rowsOf(container);

        const unknownRow = rows.find((row) => row.textContent?.includes('192.168.1.11'));
        expect(unknownRow).toBeDefined();
        expect(unknownRow?.querySelector('[data-testid="top-client-name"]')?.textContent).toBe(
            'N/A',
        );

        const knownRow = rows.find((row) => row.textContent?.includes('192.168.1.10'));
        expect(knownRow).toBeDefined();
        expect(knownRow?.querySelector('[data-testid="top-client-name"]')?.textContent).toBe(
            'Known client',
        );
    });

    // The default matchMedia mock reports `matches: false` for every query:
    // `useIsDesktop` is false (mobile layout, no query-count tooltip) while
    // `(hover: none)` is also false, so the tooltip is hover-driven — the same
    // setup as in tooltip.test.tsx.
    const openClientTooltip = async (row: Element) => {
        const trigger = row.querySelector('[data-part="trigger"]') as HTMLElement;
        const content = trigger.parentElement!.querySelector(
            '[data-part="content"]',
        ) as HTMLElement;

        expect(content.hasAttribute('hidden')).toBe(true);

        fireEvent.pointerOver(trigger, { pointerType: 'mouse' });

        await waitFor(
            () => {
                expect(content.hasAttribute('hidden')).toBe(false);
            },
            { timeout: 1000 },
        );

        return content;
    };

    it('shows the client details tooltip with address, country and network on hover', async () => {
        const clients = [
            client('149.3.181.15', 100, {
                name: 'Telecom client',
                whois_info: { country: 'RU', orgname: 'TELECOM ITALIA SPARKLE' },
            }),
        ];
        const { container } = renderTopClients(clients, 100);
        const content = await openClientTooltip(rowsOf(container)[0]);

        expect(content.textContent).toContain('Client details');
        expect(content.textContent).toContain('Address:');
        expect(content.textContent).toContain('149.3.181.15');
        expect(content.textContent).toContain('Country:');
        expect(content.textContent).toContain('RU');
        expect(content.textContent).toContain('Network:');
        expect(content.textContent).toContain('TELECOM ITALIA SPARKLE');

        // Not blocked, so no status row.
        expect(content.textContent).not.toContain('Status:');
    });

    it('shows the blocked status in the tooltip only for blocked clients', async () => {
        disallowedClients.value = '10.0.0.4';
        const clients = [
            client('10.0.0.1', 100, { name: 'Allowed client' }),
            client('10.0.0.4', 90, { name: 'Blocked client' }),
        ];
        const { container } = renderTopClients(clients, 190);
        const rows = rowsOf(container);

        const blockedRow = rows.find((row) => row.textContent?.includes('10.0.0.4'));
        expect(blockedRow).toBeDefined();
        const blockedContent = await openClientTooltip(blockedRow!);
        expect(blockedContent.textContent).toContain('Status:');
        expect(blockedContent.textContent).toContain('Blocked');

        const allowedRow = rows.find((row) => row.textContent?.includes('10.0.0.1'));
        expect(allowedRow).toBeDefined();
        const allowedContent = await openClientTooltip(allowedRow!);
        expect(allowedContent.textContent).not.toContain('Status:');
        expect(allowedContent.textContent).not.toContain('Blocked');
    });

    it('hides the country and network rows when whois info is unavailable', async () => {
        const clients = [client('192.168.1.10', 100, { name: 'No whois client' })];
        const { container } = renderTopClients(clients, 100);
        const content = await openClientTooltip(rowsOf(container)[0]);

        expect(content.textContent).toContain('Address:');
        expect(content.textContent).toContain('192.168.1.10');
        expect(content.textContent).not.toContain('Country:');
        expect(content.textContent).not.toContain('Network:');
    });
});
