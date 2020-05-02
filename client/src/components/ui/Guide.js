import React, { Fragment } from 'react';
import PropTypes from 'prop-types';
import { Trans, withNamespaces } from 'react-i18next';

import Tabs from '../ui/Tabs';
import Icons from '../ui/Icons';

const Guide = (props) => {
    const { dnsAddresses } = props;
    const tlsAddress = (dnsAddresses && dnsAddresses.filter(item => item.includes('tls://'))) || '';
    const httpsAddress =
        (dnsAddresses && dnsAddresses.filter(item => item.includes('https://'))) || '';
    const showDnsPrivacyNotice = httpsAddress.length < 1 && tlsAddress.length < 1;

    return (
        <div>
            <Icons />
            <Tabs>
                <div label="Router">
                    <div className="tab__title">
                        <Trans>install_devices_router</Trans>
                    </div>
                    <div className="tab__text">
                        <p>
                            <Trans>install_devices_router_desc</Trans>
                        </p>
                        <ol>
                            <li>
                                <Trans>install_devices_router_list_1</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_router_list_2</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_router_list_3</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_router_list_4</Trans>
                            </li>
                        </ol>
                    </div>
                </div>
                <div label="Windows">
                    <div className="tab__title">Windows</div>
                    <div className="tab__text">
                        <ol>
                            <li>
                                <Trans>install_devices_windows_list_1</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_windows_list_2</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_windows_list_3</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_windows_list_4</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_windows_list_5</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_windows_list_6</Trans>
                            </li>
                        </ol>
                    </div>
                </div>
                <div label="macOS">
                    <div className="tab__title">macOS</div>
                    <div className="tab__text">
                        <ol>
                            <li>
                                <Trans>install_devices_macos_list_1</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_macos_list_2</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_macos_list_3</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_macos_list_4</Trans>
                            </li>
                        </ol>
                    </div>
                </div>
                <div label="Android">
                    <div className="tab__title">Android</div>
                    <div className="tab__text">
                        <ol>
                            <li>
                                <Trans>install_devices_android_list_1</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_android_list_2</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_android_list_3</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_android_list_4</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_android_list_5</Trans>
                            </li>
                        </ol>
                    </div>
                </div>
                <div label="iOS">
                    <div className="tab__title">iOS</div>
                    <div className="tab__text">
                        <ol>
                            <li>
                                <Trans>install_devices_ios_list_1</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_ios_list_2</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_ios_list_3</Trans>
                            </li>
                            <li>
                                <Trans>install_devices_ios_list_4</Trans>
                            </li>
                        </ol>
                    </div>
                </div>
                <div label="dns_privacy" title={props.t('dns_privacy')}>
                    <div className="tab__title">
                        <Trans>dns_privacy</Trans>
                    </div>
                    <div className="tab__text">
                        {tlsAddress && tlsAddress.length > 0 && (
                            <div className="tab__paragraph">
                                <Trans
                                    values={{ address: tlsAddress[0] }}
                                    components={[
                                        <strong key="0">text</strong>,
                                        <code key="1">text</code>,
                                    ]}
                                >
                                    setup_dns_privacy_1
                                </Trans>
                            </div>
                        )}
                        {httpsAddress && httpsAddress.length > 0 && (
                            <div className="tab__paragraph">
                                <Trans
                                    values={{ address: httpsAddress[0] }}
                                    components={[
                                        <strong key="0">text</strong>,
                                        <code key="1">text</code>,
                                    ]}
                                >
                                    setup_dns_privacy_2
                                </Trans>
                            </div>
                        )}
                        {showDnsPrivacyNotice && (
                            <div className="tab__paragraph">
                                <Trans
                                    components={[
                                        <a
                                            href="https://github.com/AdguardTeam/AdguardHome/wiki/Encryption"
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            key="0"
                                        >
                                            link
                                        </a>,
                                        <code key="1">text</code>,
                                    ]}
                                >
                                    setup_dns_notice
                                </Trans>
                            </div>
                        )}
                        {!showDnsPrivacyNotice && (
                            <Fragment>
                                <div className="tab__paragraph">
                                    <Trans components={[<p key="0">text</p>]}>
                                        setup_dns_privacy_3
                                    </Trans>
                                </div>
                                <div className="tab__paragraph">
                                    <strong>Android</strong>
                                    <ul>
                                        <li>
                                            <Trans>setup_dns_privacy_android_1</Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://adguard.com/adguard-android/overview.html"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <code key="1">text</code>,
                                                ]}
                                            >
                                                setup_dns_privacy_android_2
                                            </Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://getintra.org/"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <code key="1">text</code>,
                                                ]}
                                            >
                                                setup_dns_privacy_android_3
                                            </Trans>
                                        </li>
                                    </ul>
                                </div>
                                <div className="tab__paragraph">
                                    <strong>iOS</strong>
                                    <ul>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://itunes.apple.com/app/id1452162351"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <code key="1">text</code>,
                                                    <a
                                                        href="https://dnscrypt.info/stamps"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="2"
                                                    >
                                                        link
                                                    </a>,
                                                ]}
                                            >
                                                setup_dns_privacy_ios_1
                                            </Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://adguard.com/adguard-ios/overview.html"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <code key="1">text</code>,
                                                ]}
                                            >
                                                setup_dns_privacy_ios_2
                                            </Trans>
                                        </li>
                                    </ul>
                                </div>
                                <div className="tab__paragraph">
                                    <strong>
                                        <Trans>setup_dns_privacy_other_title</Trans>
                                    </strong>
                                    <ul>
                                        <li>
                                            <Trans>setup_dns_privacy_other_1</Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://github.com/AdguardTeam/dnsproxy"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                ]}
                                            >
                                                setup_dns_privacy_other_2
                                            </Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://github.com/jedisct1/dnscrypt-proxy"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <code key="1">text</code>,
                                                ]}
                                            >
                                                setup_dns_privacy_other_3
                                            </Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://www.mozilla.org/firefox/"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <code key="1">text</code>,
                                                ]}
                                            >
                                                setup_dns_privacy_other_4
                                            </Trans>
                                        </li>
                                        <li>
                                            <Trans
                                                components={[
                                                    <a
                                                        href="https://dnscrypt.info/implementations"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="0"
                                                    >
                                                        link
                                                    </a>,
                                                    <a
                                                        href="https://dnsprivacy.org/wiki/display/DP/DNS+Privacy+Clients"
                                                        target="_blank"
                                                        rel="noopener noreferrer"
                                                        key="1"
                                                    >
                                                        link
                                                    </a>,
                                                ]}
                                            >
                                                setup_dns_privacy_other_5
                                            </Trans>
                                        </li>
                                    </ul>
                                </div>
                            </Fragment>
                        )}
                    </div>
                </div>
            </Tabs>
        </div>
    );
};

Guide.defaultProps = {
    dnsAddresses: [],
};

Guide.propTypes = {
    dnsAddresses: PropTypes.array,
    t: PropTypes.func.isRequired,
};

export default withNamespaces()(Guide);
