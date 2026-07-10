import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@solidjs/testing-library';
import { StaticLeasesTable } from 'panel/components/Dhcp/LeasesPage/StaticLeasesTable';

const LEASES = [{ mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.100', hostname: 'host1' }];

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
});
