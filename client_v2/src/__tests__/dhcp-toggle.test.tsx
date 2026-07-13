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
        };
        const { container } = render(() => <DhcpToggle selectedInterface={() => 'eth0'} />);
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        expect(input.checked).toBe(true);
    });

    it('calls toggleDhcp with enabled:false + interface_name when toggled ON + calls onToggleOn', () => {
        mocks.dhcpState = {
            enabled: false,
            interface_name: 'eth0',
            processingDhcp: false,
            processingConfig: false,
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
            }),
        );
        expect(onToggleOn).toHaveBeenCalledOnce();
    });

    it('calls toggleDhcp with enabled:true when toggled OFF, does NOT call onToggleOn', () => {
        mocks.dhcpState = {
            enabled: true,
            interface_name: 'eth0',
            processingDhcp: false,
            processingConfig: false,
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
        };
        const { container } = render(() => <DhcpToggle selectedInterface={() => ''} />);
        const input = container.querySelector('#dhcp_enabled') as HTMLInputElement;
        expect(input.disabled).toBe(true);
    });
});
