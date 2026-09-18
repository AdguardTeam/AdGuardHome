import { describe, it, expect } from 'vitest';

import { copy, copyInDom, copyPlural, createIntlMock, en } from './copy';

describe('copy', () => {
    it('returns the base-locale value for a key', () => {
        expect(copy('save')).toBe(en.save);
    });

    it('interpolates every %name% placeholder', () => {
        const text = copy('tls_setup_error_port_busy', {
            port: 8853,
            protocol: 'DNS-over-TLS',
        });

        expect(text).toContain('8853');
        expect(text).toContain('DNS-over-TLS');
        expect(text).not.toContain('%port%');
        expect(text).not.toContain('%protocol%');
    });

    it('leaves placeholders without a value alone', () => {
        // Components occasionally render a message without its values; the
        // real `intl.getMessage` warns and keeps the template, so a test that
        // does the same must not have the text mangled underneath it.
        expect(copy('dns_rate_limit_value')).toBe(en.dns_rate_limit_value);
    });
});

describe('copyInDom', () => {
    it('collapses a non-breaking space so a DOM query can find the element', () => {
        // The base locale keeps `about\u00a0to` together; the queries read the
        // collapsed text, so the matcher has to be collapsed as well.
        expect(copy('tls_certificate_expiring')).not.toBe(copyInDom('tls_certificate_expiring'));
        expect(copyInDom('tls_certificate_expiring')).toContain('about to');
    });

    it('leaves copy without special whitespace untouched', () => {
        expect(copyInDom('save')).toBe(copy('save'));
    });
});

describe('copyPlural', () => {
    it('uses the singular form for one', () => {
        expect(copyPlural('settings_days', 1)).toBe('1 day');
    });

    it('uses the plural form for the other counts', () => {
        expect(copyPlural('settings_days', 5)).toBe('5 days');
    });

    it('renders the zero form — empty in the base locale — for zero', () => {
        // English declares `zero | singular | plural`; the zero form is empty
        // here, so the result is empty rather than the plural wording.
        expect(copyPlural('settings_days', 0)).toBe('');
    });

    it('forces %count% to the given number', () => {
        expect(copyPlural('list_updated', 3)).toBe('3 lists updated');
        expect(copyPlural('list_updated', 1, { count: 99 })).toBe('1 list updated');
    });
});

describe('createIntlMock', () => {
    it('serves the base-locale copy for a known key', () => {
        const { default: intl } = createIntlMock();

        expect(intl.getMessage('save')).toBe(en.save);
        expect(intl.getMessage('dns_ttl_value', { value: 60 })).toContain('60');
    });

    it('throws on an unknown key so a stale key fails loudly', () => {
        const { default: intl } = createIntlMock();

        expect(() => intl.getMessage('not_a_key')).toThrow(/not_a_key/);
    });

    it('pluralises through the same table', () => {
        const { default: intl } = createIntlMock();

        expect(intl.getPlural('settings_hours', 1)).toBe('1 hour');
        expect(intl.getPlural('settings_hours', 2)).toBe('2 hours');
    });
});
