import React from 'react';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, it, expect, vi } from 'vitest';

import { DhcpInterfaces } from 'panel/initialState';
import { Ipv4Settings } from 'panel/components/Dhcp/blocks/Ipv4Settings';

const defaultProps = {
    v4: {
        gateway_ip: '',
        subnet_mask: '',
        range_start: '',
        range_end: '',
        lease_duration: 0,
    },
    interfaces: {
        eth0: {
            name: 'eth0',
            flags: 'up',
            gateway_ip: '192.168.1.1',
            ip_addresses: ['192.168.1.1'],
            ipv4_addresses: ['192.168.1.1'],
            ipv6_addresses: [],
            hardware_address: '00:00:00:00:00:00',
        },
    } as DhcpInterfaces,
    selectedInterface: 'eth0',
    processingConfig: false,
    onSave: vi.fn(),
};

describe('Ipv4Settings', () => {
    it('renders all fields', () => {
        render(<Ipv4Settings {...defaultProps} />);
        expect(screen.getByLabelText('Gateway IP address')).toBeInTheDocument();
        expect(screen.getByPlaceholderText('192.168.1.2')).toBeInTheDocument();
        expect(screen.getByPlaceholderText('192.168.1.254')).toBeInTheDocument();
        expect(screen.getByLabelText('Subnet mask')).toBeInTheDocument();
    });

    it('shows error for gateway IP that is within the DHCP range on blur', async () => {
        render(<Ipv4Settings {...defaultProps} />);

        const gatewayInput = screen.getByLabelText('Gateway IP address');
        const rangeStartInput = screen.getByPlaceholderText('192.168.1.2');
        const rangeEndInput = screen.getByPlaceholderText('192.168.1.254');
        const subnetInput = screen.getByLabelText('Subnet mask');

        // Fill range fields first
        fireEvent.change(rangeStartInput, { target: { value: '192.168.1.1' } });
        fireEvent.change(rangeEndInput, { target: { value: '192.168.1.100' } });
        fireEvent.change(subnetInput, { target: { value: '255.255.255.0' } });

        // Fill gateway inside the range, then blur to trigger validation
        fireEvent.change(gatewayInput, { target: { value: '192.168.1.50' } });
        fireEvent.blur(gatewayInput);

        // Should show out-of-range error with the range endpoints
        await waitFor(() => {
            expect(screen.getByText(/Must be out of range/)).toBeInTheDocument();
        });
    });

    it('shows error for range end <= range start on blur', async () => {
        render(<Ipv4Settings {...defaultProps} />);

        const rangeStartInput = screen.getByPlaceholderText('192.168.1.2');
        const rangeEndInput = screen.getByPlaceholderText('192.168.1.254');
        const subnetInput = screen.getByLabelText('Subnet mask');

        fireEvent.change(rangeStartInput, { target: { value: '192.168.1.100' } });
        fireEvent.change(subnetInput, { target: { value: '255.255.255.0' } });
        fireEvent.change(rangeEndInput, { target: { value: '192.168.1.50' } });
        fireEvent.blur(rangeEndInput);

        await waitFor(() => {
            expect(screen.getByText('Must be greater than range start')).toBeInTheDocument();
        });
    });

    it('shows error for invalid subnet mask on blur', async () => {
        render(<Ipv4Settings {...defaultProps} />);

        const gatewayInput = screen.getByLabelText('Gateway IP address');
        const subnetInput = screen.getByLabelText('Subnet mask');

        fireEvent.change(gatewayInput, { target: { value: '192.168.1.1' } });
        fireEvent.change(subnetInput, { target: { value: '999.999.999.999' } });
        fireEvent.blur(subnetInput);

        await waitFor(() => {
            expect(screen.getByText('Invalid subnet mask')).toBeInTheDocument();
        });
    });

    it('shows error for range start outside subnet on blur', async () => {
        render(<Ipv4Settings {...defaultProps} />);

        const gatewayInput = screen.getByLabelText('Gateway IP address');
        const subnetInput = screen.getByLabelText('Subnet mask');
        const rangeStartInput = screen.getByPlaceholderText('192.168.1.2');

        fireEvent.change(gatewayInput, { target: { value: '192.168.1.1' } });
        fireEvent.change(subnetInput, { target: { value: '255.255.255.0' } });
        fireEvent.change(rangeStartInput, { target: { value: '10.0.0.1' } });
        fireEvent.blur(rangeStartInput);

        await waitFor(() => {
            expect(screen.getByText('Addresses must be in one subnet')).toBeInTheDocument();
        });
    });

    it('calls onSave with valid data', async () => {
        const onSave = vi.fn();
        const user = userEvent.setup();
        render(<Ipv4Settings {...defaultProps} onSave={onSave} />);

        fireEvent.change(screen.getByLabelText('Gateway IP address'), {
            target: { value: '192.168.1.1' },
        });
        fireEvent.change(screen.getByLabelText('Subnet mask'), {
            target: { value: '255.255.255.0' },
        });
        fireEvent.change(screen.getByPlaceholderText('192.168.1.2'), {
            target: { value: '192.168.1.2' },
        });
        fireEvent.change(screen.getByPlaceholderText('192.168.1.254'), {
            target: { value: '192.168.1.254' },
        });
        fireEvent.change(screen.getByLabelText('DHCP lease time (in seconds)'), {
            target: { value: '86400' },
        });

        await user.click(screen.getByRole('button', { name: 'Save' }));

        expect(onSave).toHaveBeenCalledWith({
            gateway_ip: '192.168.1.1',
            subnet_mask: '255.255.255.0',
            range_start: '192.168.1.2',
            range_end: '192.168.1.254',
            lease_duration: 86400,
        });
    });

    it('clears error on blur after fixing invalid value', async () => {
        render(<Ipv4Settings {...defaultProps} />);

        const rangeStartInput = screen.getByPlaceholderText('192.168.1.2');
        const rangeEndInput = screen.getByPlaceholderText('192.168.1.254');
        const subnetInput = screen.getByLabelText('Subnet mask');

        // Trigger an invalid range end error via blur
        fireEvent.change(rangeStartInput, { target: { value: '192.168.1.100' } });
        fireEvent.change(subnetInput, { target: { value: '255.255.255.0' } });
        fireEvent.change(rangeEndInput, { target: { value: '192.168.1.50' } });
        fireEvent.blur(rangeEndInput);

        await waitFor(() => {
            expect(screen.getByText('Must be greater than range start')).toBeInTheDocument();
        });

        // Fix the value and blur again — error should clear via re-validation
        fireEvent.change(rangeEndInput, { target: { value: '192.168.1.200' } });
        fireEvent.blur(rangeEndInput);

        await waitFor(() => {
            expect(screen.queryByText('Must be greater than range start')).not.toBeInTheDocument();
        });
    });
});
