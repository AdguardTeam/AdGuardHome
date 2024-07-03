import React from 'react';
import { Trans, withTranslation } from 'react-i18next';
import { COMMENT_LINE_DEFAULT_TOKEN } from '../../../../helpers/constants';

interface ExamplesProps {
    t: (...args: unknown[]) => string;
}

const Examples = (props: ExamplesProps) => (
    <div className="list leading-loose">
        <Trans>examples_title</Trans>:
        <ol className="leading-loose">
            <li>
                <code>94.140.14.140</code>, <code>2a10:50c0::1:ff</code>: {props.t('example_upstream_regular')}
            </li>

            <li>
                <code>94.140.14.140:53</code>, <code>[2a10:50c0::1:ff]:53</code>:{' '}
                {props.t('example_upstream_regular_port')}
            </li>

            <li>
                <code>udp://unfiltered.adguard-dns.com</code>: <Trans>example_upstream_udp</Trans>
            </li>

            <li>
                <code>tcp://94.140.14.140</code>, <code>tcp://[2a10:50c0::1:ff]</code>:{' '}
                <Trans>example_upstream_tcp</Trans>
            </li>

            <li>
                <code>tcp://94.140.14.140:53</code>, <code>tcp://[2a10:50c0::1:ff]:53</code>:{' '}
                <Trans>example_upstream_tcp_port</Trans>
            </li>

            <li>
                <code>tcp://unfiltered.adguard-dns.com</code>: <Trans>example_upstream_tcp_hostname</Trans>
            </li>

            <li>
                <code>tls://unfiltered.adguard-dns.com</code>:{' '}
                <Trans
                    components={[
                        <a
                            href="https://en.wikipedia.org/wiki/DNS_over_TLS"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="0">
                            DNS-over-TLS
                        </a>,
                    ]}>
                    example_upstream_dot
                </Trans>
            </li>

            <li>
                <code>https://unfiltered.adguard-dns.com/dns-query</code>:{' '}
                <Trans
                    components={[
                        <a
                            href="https://en.wikipedia.org/wiki/DNS_over_HTTPS"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="0">
                            DNS-over-HTTPS
                        </a>,
                    ]}>
                    example_upstream_doh
                </Trans>
            </li>

            <li>
                <code>h3://unfiltered.adguard-dns.com/dns-query</code>:{' '}
                <Trans
                    components={[
                        <a
                            href="https://en.wikipedia.org/wiki/HTTP/3"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="0">
                            HTTP/3
                        </a>,
                    ]}>
                    example_upstream_doh3
                </Trans>
            </li>

            <li>
                <code>quic://unfiltered.adguard-dns.com</code>:{' '}
                <Trans
                    components={[
                        <a
                            href="https://datatracker.ietf.org/doc/html/rfc9250"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="0">
                            DNS-over-QUIC
                        </a>,
                    ]}>
                    example_upstream_doq
                </Trans>
            </li>

            <li>
                <code>sdns://...</code>:{' '}
                <Trans
                    components={[
                        <a href="https://dnscrypt.info/stamps/" target="_blank" rel="noopener noreferrer" key="0">
                            DNS Stamps
                        </a>,

                        <a href="https://dnscrypt.info/" target="_blank" rel="noopener noreferrer" key="1">
                            DNSCrypt
                        </a>,

                        <a
                            href="https://en.wikipedia.org/wiki/DNS_over_HTTPS"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="2">
                            DNS-over-HTTPS
                        </a>,
                    ]}>
                    example_upstream_sdns
                </Trans>
            </li>

            <li>
                <code>[/example.local/]94.140.14.140</code>:{' '}
                <Trans
                    components={[
                        <a
                            href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams-for-domains"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="0">
                            Link
                        </a>,
                    ]}>
                    example_upstream_reserved
                </Trans>
            </li>

            <li>
                <code>[/example.local/]94.140.14.140 2a10:50c0::1:ff</code>:{' '}
                <Trans
                    components={[
                        <a
                            href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams-for-domains"
                            target="_blank"
                            rel="noopener noreferrer"
                            key="0">
                            Link
                        </a>,
                    ]}>
                    example_multiple_upstreams_reserved
                </Trans>
            </li>

            <li>
                <code>{COMMENT_LINE_DEFAULT_TOKEN} comment</code>: <Trans>example_upstream_comment</Trans>
            </li>
        </ol>
    </div>
);

export default withTranslation()(Examples);
