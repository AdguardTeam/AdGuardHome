import { createSignal, untrack } from 'solid-js';
import { render, fireEvent } from '@solidjs/testing-library';
import { describe, it, expect, vi } from 'vitest';

import { WebBanner } from '../install/Setup/blocks/Banner/WebBanner';
import { DnsBanner } from '../install/Setup/blocks/Banner/DnsBanner';

const IP_OPTIONS = [{ value: '0.0.0.0', label: 'All interfaces' }];

describe('WebBanner', () => {
    it('commits the port value on each keystroke (input event)', () => {
        const [webPort, setWebPort] = createSignal(80);
        const onPortChange = vi.fn(setWebPort);

        render(() => (
            <WebBanner
                class="banner"
                webIp={() => '0.0.0.0'}
                webPort={webPort}
                setWebIp={() => {}}
                setWebPort={onPortChange}
                webIpOptions={IP_OPTIONS}
            />
        ));

        const input = document.querySelector('#install_web_port') as HTMLInputElement;
        fireEvent.input(input, { target: { value: '800' } });

        expect(onPortChange).toHaveBeenCalledWith(800);
        expect(untrack(webPort)).toBe(800);
    });

    it('hides the raw backend status when the port is already in use', () => {
        const [webPort] = createSignal(80);

        render(() => (
            <WebBanner
                class="banner"
                webIp={() => '0.0.0.0'}
                webPort={webPort}
                setWebIp={() => {}}
                setWebPort={() => {}}
                webIpOptions={IP_OPTIONS}
                webStatus="validating ports: listen tcp 0.0.0.0:80: bind: address already in use"
            />
        ));

        // The localized input hint covers this case; the raw status is redundant.
        expect(document.body.textContent).not.toContain('validating ports');
    });

    it('shows the raw backend status for other errors', () => {
        const [webPort] = createSignal(80);

        render(() => (
            <WebBanner
                class="banner"
                webIp={() => '0.0.0.0'}
                webPort={webPort}
                setWebIp={() => {}}
                setWebPort={() => {}}
                webIpOptions={IP_OPTIONS}
                webStatus="failed to listen"
            />
        ));

        expect(document.body.textContent).toContain('failed to listen');
    });
});

describe('DnsBanner', () => {
    it('commits the port value on each keystroke (input event)', () => {
        const [dnsPort, setDnsPort] = createSignal(53);
        const onPortChange = vi.fn(setDnsPort);

        render(() => (
            <DnsBanner
                class="banner"
                dnsIp={() => '0.0.0.0'}
                dnsPort={dnsPort}
                setDnsIp={() => {}}
                setDnsPort={onPortChange}
                dnsIpOptions={IP_OPTIONS}
                isDnsFixAvailable={false}
                onAutofix={() => {}}
            />
        ));

        const input = document.querySelector('#install_dns_port') as HTMLInputElement;
        fireEvent.input(input, { target: { value: '5353' } });

        expect(onPortChange).toHaveBeenCalledWith(5353);
        expect(untrack(dnsPort)).toBe(5353);
    });

    it('hides the raw backend status when the DNS port is already in use', () => {
        const [dnsPort] = createSignal(53);

        render(() => (
            <DnsBanner
                class="banner"
                dnsIp={() => '0.0.0.0'}
                dnsPort={dnsPort}
                setDnsIp={() => {}}
                setDnsPort={() => {}}
                dnsIpOptions={IP_OPTIONS}
                dnsStatus="validating ports: listen tcp 0.0.0.0:53: bind: address already in use"
                isDnsFixAvailable={false}
                onAutofix={() => {}}
            />
        ));

        expect(document.body.textContent).not.toContain('validating ports');
    });

    it('renders the Fix label on the autofix button', () => {
        const [dnsPort] = createSignal(53);

        render(() => (
            <DnsBanner
                class="banner"
                dnsIp={() => '0.0.0.0'}
                dnsPort={dnsPort}
                setDnsIp={() => {}}
                setDnsPort={() => {}}
                dnsIpOptions={IP_OPTIONS}
                dnsStatus="validating ports: listen tcp 0.0.0.0:53: bind: address already in use"
                isDnsFixAvailable
                onAutofix={() => {}}
            />
        ));

        const button = document.querySelector('#install_dns_fix') as HTMLButtonElement;
        expect(button).toBeTruthy();
        expect(button.textContent).toBe('Fix');
    });
});
