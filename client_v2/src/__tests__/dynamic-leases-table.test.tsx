import { describe, it, expect, vi } from 'vitest';
import { render, fireEvent } from '@solidjs/testing-library';
import { DynamicLeasesTable } from 'panel/components/Dhcp/LeasesPage/DynamicLeasesTable';

const LEASES = [{ mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.101', hostname: 'dyn1' }];

describe('DynamicLeasesTable', () => {
    it('renders mobile action buttons for all four actions', () => {
        const { getByTestId } = render(() => (
            <DynamicLeasesTable
                leases={LEASES}
                processingUpdating={false}
                processingDeleting={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onMakeStatic={() => {}}
                onRefresh={() => {}}
            />
        ));

        expect(getByTestId('dynamic-lease-edit-button')).toBeInTheDocument();
        expect(getByTestId('dynamic-lease-make-static-button')).toBeInTheDocument();
        expect(getByTestId('dynamic-lease-refresh-button')).toBeInTheDocument();
        expect(getByTestId('dynamic-lease-delete-button')).toBeInTheDocument();
    });

    it('renders desktop Dropdown trigger', () => {
        const { getByTestId } = render(() => (
            <DynamicLeasesTable
                leases={LEASES}
                processingUpdating={false}
                processingDeleting={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onMakeStatic={() => {}}
                onRefresh={() => {}}
            />
        ));

        expect(getByTestId('dynamic-lease-actions-dropdown')).toBeInTheDocument();
    });

    it('fires onEdit when mobile edit button clicked', () => {
        const onEdit = vi.fn();
        const { getByTestId } = render(() => (
            <DynamicLeasesTable
                leases={LEASES}
                processingUpdating={false}
                processingDeleting={false}
                onEdit={onEdit}
                onDelete={() => {}}
                onMakeStatic={() => {}}
                onRefresh={() => {}}
            />
        ));

        fireEvent.click(getByTestId('dynamic-lease-edit-button'));
        expect(onEdit).toHaveBeenCalledWith(LEASES[0]);
    });

    it('fires onMakeStatic when mobile make-static button clicked', () => {
        const onMakeStatic = vi.fn();
        const { getByTestId } = render(() => (
            <DynamicLeasesTable
                leases={LEASES}
                processingUpdating={false}
                processingDeleting={false}
                onEdit={() => {}}
                onDelete={() => {}}
                onMakeStatic={onMakeStatic}
                onRefresh={() => {}}
            />
        ));

        fireEvent.click(getByTestId('dynamic-lease-make-static-button'));
        expect(onMakeStatic).toHaveBeenCalledWith(LEASES[0]);
    });
});
