import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent, screen, waitFor } from '@solidjs/testing-library';
import { HashRouter, Route } from '@solidjs/router';

type MockDhcpState = {
    processing: boolean;
    processingStatus: boolean;
    processingInterfaces: boolean;
    processingDhcp: boolean;
    processingConfig: boolean;
    processingAdding: boolean;
    processingDeleting: boolean;
    processingUpdating: boolean;
    processingReset: boolean;
    enabled: boolean;
    interface_name: string;
    check: unknown;
    v4: {
        gateway_ip: string;
        subnet_mask: string;
        range_start: string;
        range_end: string;
        lease_duration: number;
    };
    v6: {
        range_start: string;
        lease_duration: number;
    };
    leases: { mac: string; ip: string; hostname: string }[];
    staticLeases: unknown[];
    isModalOpen: boolean;
    leaseModalConfig: { mac: string; ip: string; hostname: string } | undefined;
    modalType: string;
    dhcp_available: boolean;
    staticIpError: boolean;
    interfaces: unknown;
};

const mocks = vi.hoisted(() => {
    const makeMockState = (): MockDhcpState => ({
        processing: false,
        processingStatus: false,
        processingInterfaces: false,
        processingDhcp: false,
        processingConfig: false,
        processingAdding: false,
        processingDeleting: false,
        processingUpdating: false,
        processingReset: false,
        enabled: false,
        interface_name: 'eth0',
        check: null,
        v4: {
            gateway_ip: '',
            subnet_mask: '',
            range_start: '',
            range_end: '',
            lease_duration: 0,
        },
        v6: {
            range_start: '',
            lease_duration: 0,
        },
        leases: [],
        staticLeases: [],
        isModalOpen: false,
        leaseModalConfig: undefined,
        modalType: '',
        dhcp_available: true,
        staticIpError: false,
        interfaces: {},
    });

    return {
        makeMockState,
        dhcpState: makeMockState() as MockDhcpState,
        toggleDhcp: vi.fn(),
        addStaticLease: vi.fn(),
        updateStaticLease: vi.fn(),
        resetDhcpLeases: vi.fn(),
        toggleLeaseModal: vi.fn(),
        getDhcpStatus: vi.fn(),
    };
});

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
    toggleDhcp: mocks.toggleDhcp,
    addStaticLease: mocks.addStaticLease,
    updateStaticLease: mocks.updateStaticLease,
    resetDhcpLeases: mocks.resetDhcpLeases,
    toggleLeaseModal: mocks.toggleLeaseModal,
    getDhcpStatus: mocks.getDhcpStatus,
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
    },
}));

import { LeasesPage } from 'panel/components/Dhcp/LeasesPage/LeasesPage';

const renderPage = () =>
    render(() => (
        <HashRouter>
            <Route path="/" component={() => <LeasesPage />} />
            <Route path="/dhcp" component={() => null} />
        </HashRouter>
    ));

describe('LeasesPage', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.dhcpState = mocks.makeMockState();
    });

    it('shows the disabled-DHCP banner when DHCP is disabled and available', () => {
        renderPage();

        expect(screen.getByTestId('dhcp-disabled-banner')).toBeInTheDocument();
        expect(screen.getByText('setting_not_applied_dhcp')).toBeInTheDocument();
        expect(screen.getByTestId('enable-dhcp-button')).toBeInTheDocument();
    });

    it('hides the banner when DHCP is enabled', () => {
        mocks.dhcpState = { ...mocks.dhcpState, enabled: true };
        renderPage();

        expect(screen.queryByTestId('dhcp-disabled-banner')).not.toBeInTheDocument();
    });

    it('hides the banner when DHCP is unavailable on the OS', () => {
        mocks.dhcpState = { ...mocks.dhcpState, dhcp_available: false };
        renderPage();

        expect(screen.queryByTestId('dhcp-disabled-banner')).not.toBeInTheDocument();
    });

    it('enables DHCP directly when the v4 config is present', () => {
        mocks.dhcpState = {
            ...mocks.dhcpState,
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.50',
                range_end: '192.168.1.100',
                lease_duration: 86400,
            },
        };
        renderPage();

        fireEvent.click(screen.getByTestId('enable-dhcp-button'));

        expect(mocks.toggleDhcp).toHaveBeenCalledWith({
            enabled: false,
            interface_name: 'eth0',
            v4: expect.objectContaining({ gateway_ip: '192.168.1.1' }),
            v6: expect.objectContaining({ range_start: '' }),
        });
    });

    it('navigates to the DHCP settings page when the v4 config is missing', async () => {
        renderPage();

        fireEvent.click(screen.getByTestId('enable-dhcp-button'));

        expect(mocks.toggleDhcp).not.toHaveBeenCalled();
        await waitFor(() => expect(window.location.hash).toContain('/dhcp'));
    });
});
