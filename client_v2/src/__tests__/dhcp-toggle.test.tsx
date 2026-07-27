import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@solidjs/testing-library';
import { DhcpToggle } from 'panel/components/Dhcp/blocks/DhcpToggle';

const mocks = vi.hoisted(() => ({
    toggleDhcp: vi.fn(),
    dhcpState: {
        enabled: false,
        interface_name: '',
        processingDhcp: false,
        processingConfig: false,
        v4: { gateway_ip: '', subnet_mask: '', range_start: '', range_end: '', lease_duration: 0 },
        v6: { range_start: '', lease_duration: 0 },
    },
}));

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
    toggleDhcp: mocks.toggleDhcp,
}));

describe('DhcpToggle', () => {
    beforeEach(() => vi.clearAllMocks());

    it('renders with current enabled state', () => {
        mocks.dhcpState = {
            enabled: true,
            interface_name: 'eth0',
            processingDhcp: false,
            processingConfig: false,
            v4: { gateway_ip: '', subnet_mask: '', range_start: '', range_end: '', lease_duration: 0 },
            v6: { range_start: '', lease_duration: 0 },
        };
        const { container } = render(() => <DhcpToggle selectedInterface={() => 'eth0'} />);
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        expect(input.checked).toBe(true);
    });

    it('when toggled ON without v4 config, reverts UI and calls onToggleOn without toggleDhcp', () => {
        mocks.dhcpState = {
            enabled: false,
            interface_name: 'eth0',
            processingDhcp: false,
            processingConfig: false,
            v4: { gateway_ip: '', subnet_mask: '', range_start: '', range_end: '', lease_duration: 0 },
            v6: { range_start: '', lease_duration: 0 },
        };
        const onToggleOn = vi.fn();
        const { container } = render(() => (
            <DhcpToggle selectedInterface={() => 'eth0'} onToggleOn={onToggleOn} />
        ));
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        fireEvent.change(input, { target: { checked: true } });
        // Shadow signal reverts — the switch should appear unchecked.
        expect(input.checked).toBe(false);
        // Backend must NOT be called — config is not yet filled.
        expect(mocks.toggleDhcp).not.toHaveBeenCalled();
        // Config modal should open so the user can fill v4 settings.
        expect(onToggleOn).toHaveBeenCalledOnce();
    });

    it('when toggled ON with existing v4 config, calls toggleDhcp without onToggleOn', () => {
        mocks.dhcpState = {
            enabled: false,
            interface_name: 'eth0',
            processingDhcp: false,
            processingConfig: false,
            v4: {
                gateway_ip: '192.168.1.1',
                subnet_mask: '255.255.255.0',
                range_start: '192.168.1.50',
                range_end: '192.168.1.100',
                lease_duration: 86400,
            },
            v6: { range_start: '', lease_duration: 0 },
        };
        const onToggleOn = vi.fn();
        const { container } = render(() => (
            <DhcpToggle selectedInterface={() => 'eth0'} onToggleOn={onToggleOn} />
        ));
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        fireEvent.change(input, { target: { checked: true } });
        expect(mocks.toggleDhcp).toHaveBeenCalledWith(
            expect.objectContaining({
                enabled: false,
                interface_name: 'eth0',
                v4: expect.objectContaining({ gateway_ip: '192.168.1.1' }),
            }),
        );
        // v4 is configured — no need to open the config modal.
        expect(onToggleOn).not.toHaveBeenCalled();
    });

    it('calls toggleDhcp with enabled:true when toggled OFF, does NOT call onToggleOn', () => {
        mocks.dhcpState = {
            enabled: true,
            interface_name: 'eth0',
            processingDhcp: false,
            processingConfig: false,
            v4: { gateway_ip: '', subnet_mask: '', range_start: '', range_end: '', lease_duration: 0 },
            v6: { range_start: '', lease_duration: 0 },
        };
        const onToggleOn = vi.fn();
        const { container } = render(() => (
            <DhcpToggle selectedInterface={() => 'eth0'} onToggleOn={onToggleOn} />
        ));
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        fireEvent.change(input, { target: { checked: false } });
        expect(mocks.toggleDhcp).toHaveBeenCalledWith({ enabled: true });
        expect(onToggleOn).not.toHaveBeenCalled();
    });

    it('is disabled when processing or no interface', () => {
        mocks.dhcpState = {
            enabled: false,
            interface_name: '',
            processingConfig: true,
            processingDhcp: false,
            v4: { gateway_ip: '', subnet_mask: '', range_start: '', range_end: '', lease_duration: 0 },
            v6: { range_start: '', lease_duration: 0 },
        };
        const { container } = render(() => <DhcpToggle selectedInterface={() => ''} />);
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        expect(input.disabled).toBe(true);
    });
});
