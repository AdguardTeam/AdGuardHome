import { type JSX, Show } from 'solid-js';
import { describe, expect, it, vi } from 'vitest';
import { fireEvent, render, screen } from '@solidjs/testing-library';

type ConfigDialogProps = {
    children?: JSX.Element;
    description?: JSX.Element;
    onSubmit: () => void;
    open: boolean;
};

const mocks = vi.hoisted(() => ({
    dhcpState: {
        processingConfig: false,
        v4: {
            gateway_ip: '192.168.1.1',
            subnet_mask: '255.255.255.0',
            range_start: '192.168.1.100',
            range_end: '192.168.1.200',
            lease_duration: 86400,
            options: ['66 text old-pxe.example.org', '67 text old.efi'],
        },
        interfaces: {
            eth0: {
                gateway_ip: '192.168.1.1',
                ipv4_addresses: ['192.168.1.1'],
            },
        },
    },
}));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string) => key,
    },
}));

vi.mock('panel/common/ui/ConfigDialog', () => ({
    ConfigDialog: (props: ConfigDialogProps) => (
        <Show when={props.open}>
            <section>
                {props.description}
                {props.children}
                <button
                    data-testid="config-dialog-save"
                    onClick={() => props.onSubmit()}
                    type="button"
                >
                    save
                </button>
            </section>
        </Show>
    ),
}));

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
}));

import { DhcpV4Modal } from 'panel/components/Dhcp/blocks/DhcpV4Modal';

describe('DhcpV4Modal', () => {
    it('round-trips custom DHCP options and shows the DNS override warning', () => {
        const onSave = vi.fn();
        render(() => (
            <DhcpV4Modal
                open
                selectedInterface={() => 'eth0'}
                onClose={vi.fn()}
                onSave={onSave}
            />
        ));

        expect(screen.getByText('dhcp_form_options_warning')).toBeInTheDocument();

        const options = screen.getByLabelText('dhcp_form_options') as HTMLTextAreaElement;
        expect(options.value).toBe('66 text old-pxe.example.org\n67 text old.efi');

        fireEvent.change(options, {
            target: { value: '66 text pxe.example.org\n\n67 text bootx64.efi' },
        });
        fireEvent.click(screen.getByTestId('config-dialog-save'));

        expect(onSave).toHaveBeenCalledWith(
            expect.objectContaining({
                options: ['66 text pxe.example.org', '67 text bootx64.efi'],
            }),
        );
    });
});
