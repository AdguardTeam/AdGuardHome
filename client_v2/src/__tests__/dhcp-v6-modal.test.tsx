import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@solidjs/testing-library';
import userEvent from '@testing-library/user-event';

const mocks = vi.hoisted(() => ({
    dhcpState: {
        processingConfig: false,
        v6: { range_start: '', lease_duration: 0 },
        interfaces: { eth0: { ipv6_addresses: ['fe80::1'] } },
    } as Record<string, unknown>,
}));

vi.mock('panel/stores/dhcp', () => ({
    get dhcpState() {
        return mocks.dhcpState;
    },
}));

vi.mock('panel/common/intl', async () =>
    (await import('panel/__tests__/helpers/copy')).createIntlMock(),
);

import { DhcpV6Modal } from 'panel/components/Dhcp/blocks/DhcpV6Modal';
import { copy } from 'panel/__tests__/helpers/copy';

const rangeStart = () => document.getElementById('v6_range_start') as HTMLInputElement;
const leaseDuration = () => document.getElementById('v6_lease_duration') as HTMLInputElement;
const saveButton = () => screen.getByTestId('config-dialog-save');

const renderModal = () => {
    const onSave = vi.fn();
    render(() => (
        <DhcpV6Modal
            open
            selectedInterface={() => 'eth0'}
            onClose={() => {}}
            onSave={onSave}
        />
    ));
    return onSave;
};

beforeEach(() => vi.clearAllMocks());

describe('DhcpV6Modal — empty section', () => {
    // REGRESSION: AGH-171 — the legacy client shared one react-hook-form
    // between the v4 and v6 forms, so a required v6.range_start blocked saving
    // DHCPv4 on an IPv6-capable interface.  The v6 range has no required rule.
    it('accepts an empty range start when the rest of the section is filled', async () => {
        const user = userEvent.setup();
        const onSave = renderModal();

        expect(rangeStart()).toBeEnabled();
        expect(saveButton()).toBeDisabled();

        await user.type(leaseDuration(), '86400');
        await user.click(saveButton());

        expect(onSave).toHaveBeenCalledWith({ range_start: '', lease_duration: 86400 });
        expect(screen.queryByText(copy('form_error_required'))).toBeNull();
        expect(screen.queryByText(copy('form_error_ip6_format'))).toBeNull();
    });

    it('keeps the save button disabled while nothing is entered', () => {
        renderModal();

        expect(saveButton()).toBeDisabled();
    });
});

describe('DhcpV6Modal — validation', () => {
    it('shows the IPv6 format error for an invalid range start', async () => {
        const user = userEvent.setup();
        const onSave = renderModal();

        await user.type(rangeStart(), 'not-an-ip');
        await user.tab();

        expect(screen.getByText(copy('form_error_ip6_format'))).toBeInTheDocument();

        await user.type(leaseDuration(), '86400');
        await user.click(saveButton());

        expect(onSave).not.toHaveBeenCalled();
    });
});
