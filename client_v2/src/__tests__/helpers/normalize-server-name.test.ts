import { describe, expect, it } from 'vitest';
import { normalizeServerName } from 'panel/helpers/form';

describe('normalizeServerName', () => {
    it('returns plain hostname unchanged', () => {
        expect(normalizeServerName('example.com')).toBe('example.com');
    });

    it('returns subdomain unchanged', () => {
        expect(normalizeServerName('dns.example.com')).toBe('dns.example.com');
    });

    it('strips trailing dot (FQDN notation)', () => {
        expect(normalizeServerName('example.com.')).toBe('example.com');
    });

    it('trims leading and trailing whitespace', () => {
        expect(normalizeServerName('  example.com  ')).toBe('example.com');
    });

    it('strips https:// prefix', () => {
        expect(normalizeServerName('https://example.com')).toBe('example.com');
    });

    it('strips http:// prefix', () => {
        expect(normalizeServerName('http://example.com')).toBe('example.com');
    });

    it('strips trailing slash', () => {
        expect(normalizeServerName('https://example.com/')).toBe('example.com');
    });

    it('strips all three: protocol, trailing slash, and trailing dot', () => {
        expect(normalizeServerName('  https://example.com/.  ')).toBe('example.com');
    });

    it('returns empty string for empty input', () => {
        expect(normalizeServerName('')).toBe('');
    });

    it('returns empty string for whitespace-only input', () => {
        expect(normalizeServerName('   ')).toBe('');
    });

    it('returns empty string for a bare dot', () => {
        expect(normalizeServerName('.')).toBe('');
    });

    // ── No silent rewriting beyond protocol and trailing slash ─────

    it('does NOT strip port', () => {
        expect(normalizeServerName('example.com:443')).toBe('example.com:443');
    });

    it('does NOT strip path', () => {
        expect(normalizeServerName('example.com/dns-query')).toBe('example.com/dns-query');
    });

    it('does NOT strip query string', () => {
        expect(normalizeServerName('example.com?param=1')).toBe('example.com?param=1');
    });

    it('does NOT strip fragment', () => {
        expect(normalizeServerName('example.com#section')).toBe('example.com#section');
    });

    it('preserves single-label hostname (localhost)', () => {
        expect(normalizeServerName('localhost')).toBe('localhost');
    });

    it('preserves IDN hostname', () => {
        expect(normalizeServerName('münchen.example.com')).toBe('münchen.example.com');
    });

    it('preserves IPv4 address', () => {
        expect(normalizeServerName('192.168.1.1')).toBe('192.168.1.1');
    });

    it('preserves hostname with hyphens', () => {
        expect(normalizeServerName('my-dns-host.example.com')).toBe('my-dns-host.example.com');
    });

    it('preserves hostname with digits', () => {
        expect(normalizeServerName('dns01.example.com')).toBe('dns01.example.com');
    });

    it('preserves hostname containing "https" text', () => {
        expect(normalizeServerName('https-only.example.com')).toBe('https-only.example.com');
    });
});
