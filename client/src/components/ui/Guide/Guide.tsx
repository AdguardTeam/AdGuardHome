import React, { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import i18next from 'i18next';
import { useSelector } from 'react-redux';

import { MOBILE_CONFIG_LINKS } from '../../../helpers/constants';

import Tabs from '../Tabs';

import MobileConfigForm from './MobileConfigForm';
import { RootState } from '../../../initialState';

interface renderLiProps {
    label?: string;
    components?: JSX.Element[];
}

const renderLi = ({ label, components }: renderLiProps) => (
    <li key={label}>
        <Trans
            components={components?.map((props: any) => {
                if (React.isValidElement(props)) {
                    return props;
                }
                const {
                    // eslint-disable-next-line react/prop-types
                    href,
                    target = '_blank',
                    rel = 'noopener noreferrer',
                    key = '0',
                } = props;

                return (
                    <a href={href} target={target} rel={rel} key={key}>
                        link
                    </a>
                );
            })}>
            {label}
        </Trans>
    </li>
);

const getDnsPrivacyList = () => [
    {
        title: 'Android',
        list: [
            {
                label: 'setup_dns_privacy_android_1',
            },
            {
                label: 'setup_dns_privacy_android_2',
                components: [
                    {
                        key: 0,
                        href: 'https://link.adtidy.org/forward.html?action=android&from=ui&app=home',
                    },

                    <code key="1">text</code>,
                ],
            },
            {
                label: 'setup_dns_privacy_android_3',
                components: [
                    {
                        key: 0,
                        href: 'https://getintra.org/',
                    },

                    <code key="1">text</code>,
                ],
            },
        ],
    },
    {
        title: 'iOS',
        list: [
            {
                label: 'setup_dns_privacy_ios_2',
                components: [
                    {
                        key: 0,
                        href: 'https://link.adtidy.org/forward.html?action=ios&from=ui&app=home',
                    },

                    <code key="1">text</code>,
                ],
            },
            {
                label: 'setup_dns_privacy_ios_1',
                components: [
                    {
                        key: 0,
                        href: 'https://itunes.apple.com/app/id1452162351',
                    },

                    <code key="1">text</code>,
                    {
                        key: 2,
                        href: 'https://dnscrypt.info/stamps',
                    },
                ],
            },
        ],
    },
    {
        title: 'setup_dns_privacy_other_title',
        list: [
            {
                label: 'setup_dns_privacy_other_1',
            },
            {
                label: 'setup_dns_privacy_other_2',
                components: [
                    {
                        key: 0,
                        href: 'https://github.com/AdguardTeam/dnsproxy',
                    },
                ],
            },
            {
                href: 'https://github.com/jedisct1/dnscrypt-proxy',
                label: 'setup_dns_privacy_other_3',
                components: [
                    {
                        key: 0,
                        href: 'https://github.com/jedisct1/dnscrypt-proxy',
                    },

                    <code key="1">text</code>,
                ],
            },
            {
                label: 'setup_dns_privacy_other_4',
                components: [
                    {
                        key: 0,
                        href: 'https://support.mozilla.org/kb/firefox-dns-over-https',
                    },

                    <code key="1">text</code>,
                ],
            },
            {
                label: 'setup_dns_privacy_other_5',
                components: [
                    {
                        key: 0,
                        href: 'https://dnscrypt.info/implementations',
                    },
                    {
                        key: 1,
                        href: 'https://dnsprivacy.org/wiki/display/DP/DNS+Privacy+Clients',
                    },
                ],
            },
        ],
    },
];

interface renderDnsPrivacyListProps {
    title: string;
    list: unknown[];
    renderList?: (...args: unknown[]) => string;
}

const renderDnsPrivacyList = ({ title, list }: renderDnsPrivacyListProps) => (
    <div className="tab__paragraph" key={title}>
        <strong>
            <Trans>{title}</Trans>
        </strong>

        <ul>
            {list.map(({ label, components, renderComponent = renderLi }: any) =>
                renderComponent({ label, components }),
            )}
        </ul>
    </div>
);

const getTabs = ({ tlsAddress, httpsAddress, showDnsPrivacyNotice, serverName, portHttps, t }: any) => ({
    Router: {
        // eslint-disable-next-line react/display-name
        getTitle: () => (
            <p>
                <Trans>install_devices_router_desc</Trans>
            </p>
        ),
        title: 'Router',
        list: [
            'install_devices_router_list_1',
            'install_devices_router_list_2',
            'install_devices_router_list_3',

            // eslint-disable-next-line react/jsx-key
            <Trans
                components={[
                    <a href="#dhcp" key="0">
                        link
                    </a>,
                ]}>
                install_devices_router_list_4
            </Trans>,
        ],
    },
    Windows: {
        title: 'Windows',
        list: [
            'install_devices_windows_list_1',
            'install_devices_windows_list_2',
            'install_devices_windows_list_3',
            'install_devices_windows_list_4',
            'install_devices_windows_list_5',
            'install_devices_windows_list_6',
        ],
    },
    macOS: {
        title: 'macOS',
        list: [
            'install_devices_macos_list_1',
            'install_devices_macos_list_2',
            'install_devices_macos_list_3',
            'install_devices_macos_list_4',
        ],
    },
    Android: {
        title: 'Android',
        list: [
            'install_devices_android_list_1',
            'install_devices_android_list_2',
            'install_devices_android_list_3',
            'install_devices_android_list_4',
            'install_devices_android_list_5',
        ],
    },
    iOS: {
        title: 'iOS',
        list: [
            'install_devices_ios_list_1',
            'install_devices_ios_list_2',
            'install_devices_ios_list_3',
            'install_devices_ios_list_4',
        ],
    },
    dns_privacy: {
        title: 'dns_privacy',
        getTitle: function Title() {
            return (
                <div title={t('dns_privacy')}>
                    <div className="tab__text">
                        {tlsAddress?.length > 0 && (
                            <div className="tab__paragraph">
                                <Trans
                                    values={{ address: tlsAddress[0] }}
                                    components={[<strong key="0">text</strong>, <code key="1">text</code>]}>
                                    setup_dns_privacy_1
                                </Trans>
                            </div>
                        )}
                        {httpsAddress?.length > 0 && (
                            <div className="tab__paragraph">
                                <Trans
                                    values={{ address: httpsAddress[0] }}
                                    components={[<strong key="0">text</strong>, <code key="1">text</code>]}>
                                    setup_dns_privacy_2
                                </Trans>
                            </div>
                        )}
                        {showDnsPrivacyNotice ? (
                            <div className="tab__paragraph">
                                <Trans
                                    components={[
                                        <a
                                            href="https://github.com/AdguardTeam/AdguardHome/wiki/Encryption"
                                            target="_blank"
                                            rel="noopener noreferrer"
                                            key="0">
                                            link
                                        </a>,

                                        <code key="1">text</code>,
                                    ]}>
                                    setup_dns_notice
                                </Trans>
                            </div>
                        ) : (
                            <>
                                <div className="tab__paragraph">
                                    <Trans components={[<p key="0">text</p>]}>setup_dns_privacy_3</Trans>
                                </div>
                                {getDnsPrivacyList().map(renderDnsPrivacyList)}

                                <div>
                                    <strong>
                                        <Trans>setup_dns_privacy_ioc_mac</Trans>
                                    </strong>
                                </div>

                                <div className="mb-3">
                                    <Trans components={{ highlight: <code /> }}>setup_dns_privacy_4</Trans>
                                </div>

                                <MobileConfigForm
                                    initialValues={{
                                        host: serverName,
                                        clientId: '',
                                        protocol: MOBILE_CONFIG_LINKS.DOH,
                                        port: portHttps,
                                    }}
                                />
                            </>
                        )}
                    </div>
                </div>
            );
        },
    },
});

interface renderContentProps {
    title: string;
    list: unknown[];
    getTitle?: (...args: unknown[]) => unknown;
}

const renderContent = ({ title, list, getTitle }: renderContentProps) => (
    <div title={i18next.t(title)}>
        <div className="tab__title">{i18next.t(title)}</div>

        <div className="tab__text">
            {getTitle?.()}
            {list && (
                <ol>
                    {list.map((item: any) => (
                        <li key={item}>
                            <Trans>{item}</Trans>
                        </li>
                    ))}
                </ol>
            )}
        </div>
    </div>
);

interface GuideProps {
    dnsAddresses?: unknown[];
}

const Guide = ({ dnsAddresses }: GuideProps) => {
    const { t } = useTranslation();

    const serverName = useSelector((state: RootState) => state.encryption?.server_name);

    const portHttps = useSelector((state: RootState) => state.encryption?.port_https);
    const tlsAddress = dnsAddresses?.filter((item: any) => item.includes('tls://')) ?? '';
    const httpsAddress = dnsAddresses?.filter((item: any) => item.includes('https://')) ?? '';
    const showDnsPrivacyNotice = httpsAddress.length < 1 && tlsAddress.length < 1;

    const [activeTabLabel, setActiveTabLabel] = useState('Router');

    const tabs = getTabs({
        tlsAddress,
        httpsAddress,
        showDnsPrivacyNotice,
        serverName,
        portHttps,
        t,
    });

    const activeTab = renderContent(tabs[activeTabLabel]);

    return (
        <div>
            <Tabs tabs={tabs} activeTabLabel={activeTabLabel} setActiveTabLabel={setActiveTabLabel}>
                {activeTab}
            </Tabs>
        </div>
    );
};

Guide.defaultProps = {
    dnsAddresses: [],
};

export default Guide;
