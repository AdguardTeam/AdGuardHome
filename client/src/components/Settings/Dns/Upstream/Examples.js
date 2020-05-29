import React from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

const Examples = (props) => (
    <div className="list leading-loose">
        <p>
            <Trans
                components={[
                    <a
                        href="https://kb.adguard.com/general/dns-providers"
                        target="_blank"
                        rel="noopener noreferrer"
                        key="0"
                    >
                        DNS providers
                    </a>,
                ]}
            >
                dns_providers
            </Trans>
        </p>
        <Trans>examples_title</Trans>:
        <ol className="leading-loose">
            <li>
                <code>9.9.9.9</code> - {props.t('example_upstream_regular')}
            </li>
            <li>
                <code>tls://dns.quad9.net</code> –&nbsp;
                <span>
                    <Trans
                        components={[
                            <a
                                href="https://en.wikipedia.org/wiki/DNS_over_TLS"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="0"
                            >
                                DNS-over-TLS
                            </a>,
                        ]}
                    >
                        example_upstream_dot
                    </Trans>
                </span>
            </li>
            <li>
                <code>https://dns.quad9.net/dns-query</code> –&nbsp;
                <span>
                    <Trans
                        components={[
                            <a
                                href="https://en.wikipedia.org/wiki/DNS_over_HTTPS"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="0"
                            >
                                DNS-over-HTTPS
                            </a>,
                        ]}
                    >
                        example_upstream_doh
                    </Trans>
                </span>
            </li>
            <li>
                <code>tcp://9.9.9.9</code> – <Trans>example_upstream_tcp</Trans>
            </li>
            <li>
                <code>sdns://...</code> –&nbsp;
                <span>
                    <Trans
                        components={[
                            <a
                                href="https://dnscrypt.info/stamps/"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="0"
                            >
                                DNS Stamps
                            </a>,
                            <a
                                href="https://dnscrypt.info/"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="1"
                            >
                                DNSCrypt
                            </a>,
                            <a
                                href="https://en.wikipedia.org/wiki/DNS_over_HTTPS"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="2"
                            >
                                DNS-over-HTTPS
                            </a>,
                        ]}
                    >
                        example_upstream_sdns
                    </Trans>
                </span>
            </li>
            <li>
                <code>[/example.local/]9.9.9.9</code> –&nbsp;
                <span>
                    <Trans
                        components={[
                            <a
                                href="https://github.com/AdguardTeam/AdGuardHome/wiki/Configuration#upstreams-for-domains"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="0"
                            >
                                Link
                            </a>,
                        ]}
                    >
                        example_upstream_reserved
                    </Trans>
                </span>
            </li>
        </ol>
    </div>
);

Examples.propTypes = {
    t: PropTypes.func.isRequired,
};

export default withTranslation()(Examples);
