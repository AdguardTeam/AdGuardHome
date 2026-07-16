import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen, fireEvent } from '@solidjs/testing-library';
import { createSignal } from 'solid-js';

const mocks = vi.hoisted(() => ({
    setAccessList: vi.fn(),
}));

vi.mock('panel/stores/access', () => ({
    accessState: { blocked_hosts: '*$dnstype=ANY' },
    setAccessList: mocks.setAccessList,
}));

const { themeMock } = vi.hoisted(() => {
    const proxy = new Proxy(
        {},
        {
            get: (_t, p) => (p === Symbol.toPrimitive || p === 'toString' ? () => '' : proxy),
        },
    );
    return { themeMock: proxy };
});

vi.mock('panel/lib/theme', () => ({ default: themeMock }));

vi.mock('panel/common/intl', () => ({
    default: {
        getMessage: (key: string, values?: Record<string, unknown>) => {
            if (key === 'form_error_format_line') return `Invalid format on line ${values?.line}`;
            if (key === 'form_error_format') return 'Invalid format';
            return key;
        },
    },
}));

import { DisallowedDomainsDialog } from 'panel/components/DnsSettings/Access/blocks/DisallowedDomainsDialog';

describe('DisallowedDomainsDialog', () => {
    beforeEach(() => vi.clearAllMocks());

    it('accepts $dnstype=ANY rule without showing a validation error after blur', () => {
        const [open] = createSignal(true);
        render(() => <DisallowedDomainsDialog open={open} onClose={() => {}} processing={false} />);

        const textarea = screen.getByRole('textbox') as HTMLTextAreaElement;
        expect(textarea.value).toBe('*$dnstype=ANY');

        // Trigger the onBlur validator
        fireEvent.blur(textarea);

        expect(screen.queryByText(/Invalid format/)).not.toBeInTheDocument();
    });
});
