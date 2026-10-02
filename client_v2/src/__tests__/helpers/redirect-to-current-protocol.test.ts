import { describe, it, expect, vi } from 'vitest';

import { redirectToCurrentProtocol } from 'panel/helpers/helpers';

/**
 * Replaces the jsdom location with a plain object so `replace()` is a spy.
 * `redirectToCurrentProtocol` reads the location at call time.
 */
const mockLocation = (props: {
    protocol: string;
    hostname: string;
    hash: string;
    port?: string;
}) => {
    const replace = vi.fn();
    Object.defineProperty(window, 'location', {
        value: {
            protocol: props.protocol,
            hostname: props.hostname,
            hash: props.hash,
            port: props.port ?? '',
            replace,
        },
        writable: true,
        configurable: true,
    });

    return replace;
};

describe('redirectToCurrentProtocol', () => {
    it('navigates from https to the HTTP UI when encryption is disabled', () => {
        const replace = mockLocation({
            protocol: 'https:',
            hostname: 'dns.example.com',
            hash: '#/encryption',
            port: '443',
        });

        redirectToCurrentProtocol({ enabled: false, force_https: false, port_https: 443 }, 3001);

        expect(replace).toHaveBeenCalledWith('http://dns.example.com:3001/#/encryption');
    });

    it('stays put when encryption is off on an http origin', () => {
        const replace = mockLocation({
            protocol: 'http:',
            hostname: 'dns.example.com',
            hash: '#/encryption',
        });

        redirectToCurrentProtocol({ enabled: false, force_https: false, port_https: 443 }, 80);

        expect(replace).not.toHaveBeenCalled();
    });
});
