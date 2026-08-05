import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent, screen, waitFor } from '@solidjs/testing-library';
import { createSignal } from 'solid-js';
import { StaticLeasesTable } from 'panel/components/Dhcp/LeasesPage/StaticLeasesTable';

const LEASES = [{ mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.100', hostname: 'host1' }];
const SEARCH_LEASES = [
    { mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.1', hostname: 'Office Router' },
    { mac: '11:22:33:44:55:66', ip: '192.168.1.25', hostname: 'Printer' },
    { mac: 'DE:AD:BE:EF:00:01', ip: '192.168.1.50', hostname: 'Storage' },
];

const renderSearchableTable = () =>
    render(() => (
        <StaticLeasesTable
            staticLeases={SEARCH_LEASES}
            processingDeleting={false}
            processingUpdating={false}
            onEdit={() => {}}
            onDelete={() => {}}
            onRefresh={() => {}}
        />
    ));

describe('StaticLeasesTable', () => {
    it('renders mobile action buttons for all three actions', () => {
        const { getByTestId } = render(() => (
            <StaticLeasesTable
                staticLeases={LEASES}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onRefresh={() => {}}
            />
        ));

        expect(getByTestId('static-lease-edit-button')).toBeInTheDocument();
        expect(getByTestId('static-lease-refresh-button')).toBeInTheDocument();
        expect(getByTestId('static-lease-delete-button')).toBeInTheDocument();
    });

    it('renders desktop Dropdown trigger', () => {
        const { getByTestId } = render(() => (
            <StaticLeasesTable
                staticLeases={LEASES}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onRefresh={() => {}}
            />
        ));

        expect(getByTestId('static-lease-actions-dropdown')).toBeInTheDocument();
    });

    it('fires onEdit when mobile edit button clicked', () => {
        const onEdit = vi.fn();
        const { getByTestId } = render(() => (
            <StaticLeasesTable
                staticLeases={LEASES}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={onEdit}
                onDelete={() => {}}
                onRefresh={() => {}}
            />
        ));

        fireEvent.click(getByTestId('static-lease-edit-button'));
        expect(onEdit).toHaveBeenCalledWith(LEASES[0]);
    });

    it('fires onRefresh when mobile refresh button clicked', () => {
        const onRefresh = vi.fn();
        const { getByTestId } = render(() => (
            <StaticLeasesTable
                staticLeases={LEASES}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onRefresh={onRefresh}
            />
        ));

        fireEvent.click(getByTestId('static-lease-refresh-button'));
        expect(onRefresh).toHaveBeenCalled();
    });

    it('fires onDelete when mobile delete button clicked', () => {
        const onDelete = vi.fn();
        const { getByTestId } = render(() => (
            <StaticLeasesTable
                staticLeases={LEASES}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={() => {}}
                onDelete={onDelete}
                onRefresh={() => {}}
            />
        ));

        fireEvent.click(getByTestId('static-lease-delete-button'));
        expect(onDelete).toHaveBeenCalledWith(LEASES[0]);
    });

    it.each([
        ['router', 'Office Router', ['Printer', 'Storage']],
        ['192.168.1.25', 'Printer', ['Office Router', 'Storage']],
        ['de:ad:be:ef', 'Storage', ['Office Router', 'Printer']],
    ])('filters leases by hostname, IP, or MAC for %s', (query, expected, hidden) => {
        renderSearchableTable();

        fireEvent.input(screen.getByRole('searchbox', { name: 'Search' }), {
            target: { value: query },
        });

        expect(screen.getByText(expected)).toBeInTheDocument();
        hidden.forEach((hostname) => {
            expect(screen.queryByText(hostname)).not.toBeInTheDocument();
        });
    });

    it('restores all leases when the search is cleared', () => {
        renderSearchableTable();

        const searchInput = screen.getByRole('searchbox', { name: 'Search' });
        fireEvent.input(searchInput, {
            target: { value: 'router' },
        });

        expect(searchInput.className).toContain('hideNativeSearchClear');

        fireEvent.click(screen.getByRole('button', { name: 'Clear input' }));

        SEARCH_LEASES.forEach(({ hostname }) => {
            expect(screen.getByText(hostname)).toBeInTheDocument();
        });
    });

    it('shows an empty state when no static lease matches', () => {
        renderSearchableTable();

        fireEvent.input(screen.getByRole('searchbox', { name: 'Search' }), {
            target: { value: 'missing' },
        });

        expect(screen.getByText('Nothing found')).toBeInTheDocument();
    });

    it('shows matches from the full lease list after changing pages', () => {
        const paginatedLeases = Array.from({ length: 11 }, (_, index) => ({
            mac: `AA:BB:CC:DD:EE:${String(index).padStart(2, '0')}`,
            ip: `192.168.1.${index + 1}`,
            hostname: index === 0 ? 'Target lease' : `Host ${index + 1}`,
        }));

        render(() => (
            <StaticLeasesTable
                staticLeases={paginatedLeases}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onRefresh={() => {}}
            />
        ));

        fireEvent.click(screen.getByRole('button', { name: '2' }));
        expect(screen.queryByText('Target lease')).not.toBeInTheDocument();

        fireEvent.input(screen.getByRole('searchbox', { name: 'Search' }), {
            target: { value: 'target' },
        });

        expect(screen.getByText('Target lease')).toBeInTheDocument();
    });

    it('resets the page when the filtered lease count shrinks', async () => {
        const initialLeases = Array.from({ length: 11 }, (_, index) => ({
            mac: `AA:BB:CC:DD:EE:${String(index).padStart(2, '0')}`,
            ip: `192.168.1.${index + 1}`,
            hostname: `Matching host ${index + 1}`,
        }));
        const [leases, setLeases] = createSignal(initialLeases);

        const TestTable = () => (
            <StaticLeasesTable
                staticLeases={leases()}
                processingDeleting={false}
                processingUpdating={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onRefresh={() => {}}
            />
        );

        render(() => <TestTable />);

        fireEvent.input(screen.getByRole('searchbox', { name: 'Search' }), {
            target: { value: 'matching' },
        });
        fireEvent.click(screen.getByRole('button', { name: '2' }));
        expect(screen.getByText('Matching host 11')).toBeInTheDocument();

        setLeases((current) => current.slice(0, 10));

        await waitFor(() => {
            expect(screen.getByText('Matching host 1')).toBeInTheDocument();
        });
        expect(screen.queryByText('Nothing found')).not.toBeInTheDocument();
    });
});
