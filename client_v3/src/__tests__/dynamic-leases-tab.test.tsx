import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen } from '@solidjs/testing-library';

const MOCK_LEASE = { mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.101', hostname: 'dyn1' };

const mocks = vi.hoisted(() => {
    const lease = { mac: 'AA:BB:CC:DD:EE:FF', ip: '192.168.1.101', hostname: 'dyn1' };
    return {
        dhcpState: {
            leases: [lease],
            processingUpdating: false,
            processingDeleting: false,
            processingConfig: false,
        },
        removeStaticLease: vi.fn(),
        toggleLeaseModal: vi.fn(),
        getDhcpStatus: vi.fn(),
    };
});

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
    removeStaticLease: mocks.removeStaticLease,
    toggleLeaseModal: mocks.toggleLeaseModal,
    getDhcpStatus: mocks.getDhcpStatus,
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string, values?: Record<string, unknown>) =>
            values?.ip !== undefined ? `${key}:${values.ip}` : key,
    },
}));

import { DynamicLeasesTab } from 'panel/components/Dhcp/LeasesPage/DynamicLeasesTab';

describe('DynamicLeasesTab', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        // Reset state for each test
        mocks.dhcpState = {
            leases: [MOCK_LEASE],
            processingUpdating: false,
            processingDeleting: false,
            processingConfig: false,
        };
    });

    it('renders DynamicLeasesTable when leases exist', () => {
        render(() => <DynamicLeasesTab />);

        expect(screen.getByTestId('dynamic-lease-edit-button')).toBeInTheDocument();
        expect(screen.getByTestId('dynamic-lease-make-static-button')).toBeInTheDocument();
        expect(screen.getByTestId('dynamic-lease-refresh-button')).toBeInTheDocument();
        expect(screen.getByTestId('dynamic-lease-delete-button')).toBeInTheDocument();
    });

    it('does not render DynamicLeasesTable when leases array is empty', () => {
        mocks.dhcpState = { ...mocks.dhcpState, leases: [] };
        render(() => <DynamicLeasesTab />);

        expect(screen.queryByTestId('dynamic-lease-edit-button')).not.toBeInTheDocument();
        expect(screen.queryByTestId('dynamic-lease-delete-button')).not.toBeInTheDocument();
    });

    it('does not render DynamicLeasesTable when leases is undefined', () => {
        mocks.dhcpState = {
            ...mocks.dhcpState,
            leases: undefined as unknown as typeof mocks.dhcpState.leases,
        };
        render(() => <DynamicLeasesTab />);

        expect(screen.queryByTestId('dynamic-lease-edit-button')).not.toBeInTheDocument();
    });

    it('calls toggleLeaseModal with ADD_LEASE when edit button is clicked', () => {
        render(() => <DynamicLeasesTab />);

        fireEvent.click(screen.getByTestId('dynamic-lease-edit-button'));
        expect(mocks.toggleLeaseModal).toHaveBeenCalledWith('ADD_LEASE', MOCK_LEASE);
    });

    it('calls toggleLeaseModal with MAKE_STATIC when make-static button is clicked', () => {
        render(() => <DynamicLeasesTab />);

        fireEvent.click(screen.getByTestId('dynamic-lease-make-static-button'));
        expect(mocks.toggleLeaseModal).toHaveBeenCalledWith('MAKE_STATIC', MOCK_LEASE);
    });

    it('calls getDhcpStatus when refresh button is clicked', () => {
        render(() => <DynamicLeasesTab />);

        fireEvent.click(screen.getByTestId('dynamic-lease-refresh-button'));
        expect(mocks.getDhcpStatus).toHaveBeenCalledTimes(1);
    });

    it('shows ConfirmDialog when delete button is clicked', () => {
        render(() => <DynamicLeasesTab />);

        fireEvent.click(screen.getByTestId('dynamic-lease-delete-button'));

        // ConfirmDialog should appear with the title and description
        expect(screen.getByText('delete_confirm')).toBeInTheDocument();
        expect(screen.getByText('delete_confirm_desc:192.168.1.101')).toBeInTheDocument();
    });

    it('calls removeStaticLease when delete is confirmed', () => {
        render(() => <DynamicLeasesTab />);

        // Click delete to open the confirmation dialog
        fireEvent.click(screen.getByTestId('dynamic-lease-delete-button'));

        // Click the confirm button
        const confirmButton = screen.getByText('delete_table_action_confirm');
        fireEvent.click(confirmButton);

        expect(mocks.removeStaticLease).toHaveBeenCalledWith(MOCK_LEASE);
    });

    it('closes ConfirmDialog when cancel is clicked', () => {
        render(() => <DynamicLeasesTab />);

        // Click delete to open the confirmation dialog
        fireEvent.click(screen.getByTestId('dynamic-lease-delete-button'));

        // Verify dialog is visible
        expect(screen.getByText('delete_confirm')).toBeInTheDocument();

        // Click cancel
        const cancelButton = screen.getByText('cancel');
        fireEvent.click(cancelButton);

        // Dialog should close — confirm that removeStaticLease was NOT called
        expect(mocks.removeStaticLease).not.toHaveBeenCalled();
        expect(screen.queryByText('delete_confirm')).not.toBeInTheDocument();
    });
});
