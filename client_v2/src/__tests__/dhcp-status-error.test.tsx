import { render, screen } from '@solidjs/testing-library';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const mocks = vi.hoisted(() => ({
    dhcpState: {
        processing: false,
        processingStatus: false,
        processingInterfaces: false,
        dhcp_available: false,
        statusError: true,
        interface_name: '',
        interfaces: {},
        enabled: false,
        check: null as null,
        v4: {
            gateway_ip: '',
            subnet_mask: '',
            range_start: '',
            range_end: '',
            lease_duration: 0,
        },
        v6: { range_start: '', lease_duration: 0 },
    },
    getDhcpStatus: vi.fn(),
    getDhcpInterfaces: vi.fn(),
    setDhcpConfig: vi.fn(),
    resetDhcp: vi.fn(),
}));

vi.mock('@solidjs/router', () => ({
    useNavigate: () => vi.fn(),
}));

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
    getDhcpStatus: mocks.getDhcpStatus,
    getDhcpInterfaces: mocks.getDhcpInterfaces,
    setDhcpConfig: mocks.setDhcpConfig,
    resetDhcp: mocks.resetDhcp,
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
    },
}));

import { Dhcp } from 'panel/components/Dhcp/Dhcp';

describe('Dhcp status error', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mocks.dhcpState.processing = false;
        mocks.dhcpState.processingInterfaces = false;
        mocks.dhcpState.processingStatus = false;
        mocks.dhcpState.dhcp_available = false;
        mocks.dhcpState.statusError = true;
    });

    it('shows a retryable error instead of reporting DHCP as unavailable', () => {
        render(() => <Dhcp />);

        expect(screen.getByText('error')).toBeInTheDocument();
        expect(screen.getByText('dhcp_error')).toBeInTheDocument();
        expect(screen.getByText('try_again')).toBeInTheDocument();
        expect(screen.queryByText('unavailable_dhcp')).not.toBeInTheDocument();
    });

    it('shows the page loader while a retry is in progress', () => {
        mocks.dhcpState.statusError = false;
        mocks.dhcpState.processingStatus = true;

        const { container } = render(() => <Dhcp />);

        expect(container.querySelector('use[href="#loader"]')).toBeInTheDocument();
        expect(screen.queryByText('unavailable_dhcp')).not.toBeInTheDocument();
    });

    it('does not fetch interfaces after a status error with stale availability', async () => {
        const statusRequest = Promise.resolve();

        mocks.dhcpState.dhcp_available = true;
        mocks.getDhcpStatus.mockReturnValueOnce(statusRequest);

        render(() => <Dhcp />);

        await statusRequest;
        await Promise.resolve();

        expect(mocks.getDhcpStatus).toHaveBeenCalledOnce();
        expect(mocks.getDhcpInterfaces).not.toHaveBeenCalled();
        expect(screen.getByText('error')).toBeInTheDocument();
    });
});
