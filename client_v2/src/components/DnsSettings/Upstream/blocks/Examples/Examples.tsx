import React from 'react';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { COMMENT_LINE_DEFAULT_TOKEN } from 'panel/helpers/constants';
import { Accordion } from 'panel/common/ui/Accordion';

import theme from 'panel/lib/theme';
import { Icon } from 'panel/common/ui/Icon';
import s from './Examples.module.pcss';

const examples = [
    intl.getMessage('upstream_example_udp', {
        ipv4: '94.140.14.140',
        ipv6: '2a10:50c0::1:ff',
    }),
    intl.getMessage('upstream_example_udp_port', {
        ipv4: '94.140.14.140:53',
        ipv6: '[2a10:50c0::1:ff]:53',
    }),
    intl.getMessage('upstream_example_udp_hostname', {
        value: 'udp://unfiltered.adguard-dns.com',
    }),
    intl.getMessage('upstream_example_tcp', {
        ipv4: 'tcp://94.140.14.140',
        ipv6: 'tcp://[2a10:50c0::1:ff]',
    }),
    intl.getMessage('upstream_example_tcp_port', {
        ipv4: 'tcp://94.140.14.140:53',
        ipv6: 'tcp://[2a10:50c0::1:ff]:53',
    }),
    intl.getMessage('upstream_example_tcp_hostname', {
        value: 'tcp://unfiltered.adguard-dns.com',
    }),
    intl.getMessage('upstream_example_upstream_dot', {
        value: 'tls://unfiltered.adguard-dns.com',
        a: (text: string) => (
            <a href="https://en.wikipedia.org/wiki/DNS_over_TLS" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_upstream_doh', {
        value: 'https://unfiltered.adguard-dns.com/dns-query',
        a: (text: string) => (
            <a href="https://en.wikipedia.org/wiki/DNS_over_HTTPS" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_upstream_doh3', {
        value: 'h3://unfiltered.adguard-dns.com/dns-query',
        a: (text: string) => (
            <a href="https://en.wikipedia.org/wiki/HTTP/3" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_upstream_doq', {
        value: 'quic://unfiltered.adguard-dns.com',
        a: (text: string) => (
            <a href="https://datatracker.ietf.org/doc/html/rfc9250" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_upstream_sdns', {
        value: 'sdns://...',
        a: (text: string) => (
            <a href="https://dnscrypt.info/stamps/" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
        b: (text: string) => (
            <a href="https://dnscrypt.info/" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
        c: (text: string) => (
            <a href="https://en.wikipedia.org/wiki/DNS_over_HTTPS" target="_blank" rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_upstream_reserved', {
        value: '[/example.local/]94.140.14.140',
        a: (text: string) => (
            <a
                href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams-for-domains"
                target="_blank"
                rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_multiple_upstreams_reserved', {
        value: '[/example.local/]94.140.14.140 2a10:50c0::1:ff',
        a: (text: string) => (
            <a
                href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams-for-domains"
                target="_blank"
                rel="noopener noreferrer">
                {text}
            </a>
        ),
    }),
    intl.getMessage('upstream_example_upstream_comment', {
        value: `${COMMENT_LINE_DEFAULT_TOKEN} comment`,
    }),
];

export const Examples = () => (
    <Accordion title={intl.getMessage('upstream_examples_title')} defaultOpen>
        <div className={s.list}>
            {examples.map((example, index) => (
                <div key={index} className={cn(theme.text.t3, s.listItem)}>
                    <Icon icon="label" className={s.icon} />

                    {example}
                </div>
            ))}
        </div>
    </Accordion>
);
