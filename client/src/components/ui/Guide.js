import React, { useState } from 'react';
import PropTypes from 'prop-types';
import { Trans, withTranslation } from 'react-i18next';

import Tabs from './Tabs';
import Icons from './Icons';

const dnsPrivacyList = [{
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
                    href: 'https://adguard.com/adguard-android/overview.html',
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
        {
            label: 'setup_dns_privacy_ios_2',
            components: [
                {
                    key: 0,
                    href: 'https://adguard.com/adguard-ios/overview.html',
                },
                    <code key="1">text</code>,
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
                    href: 'https://github.com/jedisct1/dnscrypt-proxy',
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

const renderDnsPrivacyList = ({ title, list }) => <div className="tab__paragraph">
    <strong><Trans>{title}</Trans></strong>
    <ul>{list.map(({ label, components }) => <li key={label}>
        <Trans
            components={components?.map((props) => {
                if (React.isValidElement(props)) {
                    return props;
                }
                const {
                    // eslint-disable-next-line react/prop-types
                    href, target = '_blank', rel = 'noopener noreferrer', key = '0',
                } = props;

                return <a
                    href={href} target={target}
                    rel={rel} key={key}>link</a>;
            })}>
            {label}
        </Trans>
    </li>)}
    </ul>
</div>;

const getTabs = ({
    tlsAddress,
    httpsAddress,
    showDnsPrivacyNotice,
    t,
}) => ({
    Router: {
        // eslint-disable-next-line react/display-name
        getTitle: () => <p>
            <Trans>install_devices_router_desc</Trans>
        </p>,
        title: 'Router',
        list: ['install_devices_router_list_1',
            'install_devices_router_list_2',
            'install_devices_router_list_3',
            // eslint-disable-next-line react/jsx-key
            <Trans components={[
                <a href="#dhcp" key="0">
                    link
                </a>,
            ]}>install_devices_router_list_4</Trans>,
        ],
    },
    Windows: {
        title: 'Windows',
        list: ['install_devices_windows_list_1',
            'install_devices_windows_list_2',
            'install_devices_windows_list_3',
            'install_devices_windows_list_4',
            'install_devices_windows_list_5',
            'install_devices_windows_list_6'],
    },
    macOS: {
        title: 'macOS',
        list: ['install_devices_macos_list_1',
            'install_devices_macos_list_2',
            'install_devices_macos_list_3',
            'install_devices_macos_list_4'],
    },
    Android: {
        title: 'Android',
        list: ['install_devices_android_list_1',
            'install_devices_android_list_2',
            'install_devices_android_list_3',
            'install_devices_android_list_4',
            'install_devices_android_list_5'],
    },
    iOS: {
        title: 'iOS',
        list: ['install_devices_ios_list_1',
            'install_devices_ios_list_2',
            'install_devices_ios_list_3',
            'install_devices_ios_list_4'],
    },
    dns_privacy: {
        title: 'dns_privacy',
        // eslint-disable-next-line react/display-name
        getTitle: () => <div label="dns_privacy" title={t('dns_privacy')}>
            <div className="tab__text">
                {tlsAddress?.length > 0 && (
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
                {httpsAddress?.length > 0 && (
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
                {showDnsPrivacyNotice
                    ? <div className="tab__paragraph">
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
                    : <>
                        <div className="tab__paragraph">
                            <Trans components={[<p key="0">text</p>]}>
                                setup_dns_privacy_3
                            </Trans>
                        </div>
                        {dnsPrivacyList.map(renderDnsPrivacyList)}
                    </>}
            </div>
        </div>,
    },
});

const renderContent = ({ title, list, getTitle }, t) => <div key={title} label={t(title)}>
    <div className="tab__title">{t(title)}</div>
    <div className="tab__text">
        {typeof getTitle === 'function' && getTitle()}
        {list
        && <ol>{list.map((item) => <li key={item}>
            <Trans>{item}</Trans>
        </li>)}
        </ol>}
    </div>
</div>;

const Guide = ({ dnsAddresses, t }) => {
    const tlsAddress = (dnsAddresses && dnsAddresses.filter((item) => item.includes('tls://'))) || '';
    const httpsAddress = (dnsAddresses && dnsAddresses.filter((item) => item.includes('https://'))) || '';
    const showDnsPrivacyNotice = httpsAddress.length < 1 && tlsAddress.length < 1;

    const [activeTabLabel, setActiveTabLabel] = useState('Router');

    const tabs = getTabs({
        tlsAddress,
        httpsAddress,
        showDnsPrivacyNotice,
        t,
    });

    const activeTab = renderContent(tabs[activeTabLabel], t);

    return (
        <div>
            <Icons />
            <Tabs tabs={tabs} activeTabLabel={activeTabLabel}
                  setActiveTabLabel={setActiveTabLabel}>{activeTab}</Tabs>
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

renderDnsPrivacyList.propTypes = {
    title: PropTypes.string.isRequired,
    list: PropTypes.array.isRequired,
};

renderContent.propTypes = {
    title: PropTypes.string.isRequired,
    list: PropTypes.array.isRequired,
    getTitle: PropTypes.func,
};

export default withTranslation()(Guide);
