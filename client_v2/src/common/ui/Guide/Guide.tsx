import React, { useState } from 'react';
import { Trans, useTranslation } from 'react-i18next';
import i18next from 'i18next';
import { useSelector } from 'react-redux';
import cn from 'clsx';

import { RootState } from 'panel/initialState';
import { MOBILE_CONFIG_LINKS } from 'panel/helpers/constants';
import { MobileConfigForm } from 'panel/components/SetupGuide/MobileConfigForm';
import { IconType } from '../Icons';

import { Select } from '../../controls/Select';
import { CopiedText } from '../CopiedText';
import s from './Guide.module.pcss';

const dnsDevicesConfig = [
    {
        title: 'Android',
        list: [
            {
                label: 'setup_devices_dns_android_list_1',
            },
            {
                label: 'setup_devices_dns_android_list_2',
                components: [
                    {
                        key: 0,
                        href: 'https://adguard.com/adguard-android/overview.html',
                    },
                ],
            },
            {
                label: 'setup_devices_dns_android_list_3',
                components: [
                    {
                        key: 0,
                        href: 'https://getintra.org/',
                    },
                ],
            },
        ],
    },
    {
        title: 'iOS',
        list: [
            {
                label: 'setup_devices_dns_ios_list_1',
                components: [
                    {
                        key: 0,
                        href: 'https://adguard.com/adguard-ios/overview.html',
                    },
                ],
            },
            {
                label: 'setup_devices_dns_ios_list_2',
                components: [
                    {
                        key: 0,
                        href: 'https://github.com/s-s/dnscloak',
                    },
                    {
                        key: 1,
                        href: 'https://dnscrypt.info/stamps-specifications',
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
                components: [
                    {
                        key: 0,
                        href: 'https://github.com/AdguardTeam/AdGuardHome',
                    },
                ],
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
                label: 'setup_dns_privacy_other_3',
                components: [
                    {
                        key: 0,
                        href: 'https://github.com/DNSCrypt/dnscrypt-proxy',
                    },
                ],
            },
            {
                label: 'setup_dns_privacy_other_4',
                components: [
                    {
                        key: 0,
                        href: 'https://support.mozilla.org/en-US/kb/firefox-dns-over-https',
                    },
                ],
            },
            {
                label: 'setup_devices_dns_other_list_5',
                components: [
                    {
                        key: 0,
                        href: 'https://dnscrypt.info/',
                    },
                    {
                        key: 1,
                        href: 'https://dnsprivacy.org/',
                    },
                ],
            },
        ],
    },
];

const renderDnsDevicesList = () => (
    <>
        {dnsDevicesConfig.map((section) => (
            <div className={s.paragraph} key={section.title}>
                <div className={s.guideTitle}>
                    <strong>
                        {section.title.startsWith('setup_') ? (
                            <Trans>{section.title}</Trans>
                        ) : (
                            section.title
                        )}
                    </strong>
                </div>
                <ul className={s.guideList}>
                    {section.list.map((item, index) => (
                        <li key={item.label || index}>
                            <Trans
                                i18nKey={item.label}
                                components={item.components?.map((comp) => (
                                    <a
                                        key={comp.key}
                                        href={comp.href}
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        className={s.dnsLink}
                                    />
                                ))}
                            />
                        </li>
                    ))}
                </ul>
            </div>
        ))}
    </>
);

const getDnsSettingsContent = (dnsAddresses: any, serverName?: string, portHttps?: number) => {
    const tlsAddress = dnsAddresses?.find((addr: string) => addr.includes('tls://'));
    const httpsAddress = dnsAddresses?.find((addr: string) => addr.includes('https://'));
    const quicAddress = dnsAddresses?.find((addr: string) => addr.includes('quic://'));

    return (
        <>
            <ul>
                {tlsAddress && (
                    <li>
                        <Trans>setup_devices_dns_list_1</Trans>
                        <div style={{ marginTop: '8px' }}>
                            <CopiedText text={tlsAddress} />
                        </div>
                    </li>
                )}
                {httpsAddress && (
                    <li>
                        <Trans>setup_devices_dns_list_2</Trans>
                        <div style={{ marginTop: '8px' }}>
                            <CopiedText text={httpsAddress} />
                        </div>
                    </li>
                )}
                {quicAddress && (
                    <li>
                        <Trans>setup_devices_dns_list_3</Trans>
                        <div style={{ marginTop: '8px' }}>
                            <CopiedText text={quicAddress} />
                        </div>
                    </li>
                )}
            </ul>

        <div className={s.paragraph}>
            <Trans>setup_devices_dns_list_devices</Trans>
        </div>

        {renderDnsDevicesList()}

        <div className={s.paragraph}>
            <div className={s.guideTitle}>
                <strong>iOS and macOS configuration</strong>
            </div>
            <div className="mb-3">
                <Trans>setup_devices_dns_macos_desc</Trans>
            </div>
            <div className={s.mobileConfigContainer}>
                <MobileConfigForm
                    initialValues={{
                        host: serverName || '',
                        clientId: '',
                        protocol: MOBILE_CONFIG_LINKS.DOH,
                        port: portHttps,
                    }}
                />
            </div>
        </div>
        </>
    );
};

const DnsPrivacyTitle = ({ serverName, portHttps, t, dnsAddresses }: any) => (
    <div title={t('dns_privacy')}>
        <div className={s.text}>
            {getDnsSettingsContent(dnsAddresses, serverName, portHttps)}
        </div>
    </div>
);

const getTabs = ({ tlsAddress, httpsAddress, showDnsPrivacyNotice, serverName, portHttps, t, dnsAddresses }: any) => ({
    Router: {
        title: 'Router',
        icon: 'router',
        subtitle_1: 'setup_devices_router_desc_1',
        list: [
            {
                label: 'setup_devices_router_list_1',
                components: [
                    <CopiedText key="0" text="http://192.168.0.1" />,
                    <CopiedText key="1" text="http://192.168.1.1" />,
                ],
            },
            'setup_devices_router_list_2',
        ],
        subtitle_2: 'setup_devices_router_desc_2',
        subtitle_2_components: [
            {
                key: 0,
                href: '/dhcp',
            },
        ],
        strongNumbers: true,
    },
    Windows: {
        title: 'Windows',
        icon: 'windows',
        list: [
            'setup_devices_windows_list_1',
            'setup_devices_windows_list_2',
            'setup_devices_windows_list_3',
            'setup_devices_windows_list_4',
            'setup_devices_windows_list_5',
            'setup_devices_windows_list_6',
        ],
    },
    macOS: {
        title: 'macOS',
        icon: 'mac',
        list: [
            'setup_devices_macos_list_1',
            'setup_devices_macos_list_2',
            'setup_devices_macos_list_3',
        ],
    },
    Android: {
        title: 'Android',
        icon: 'android',
        list: [
            'setup_devices_android_list_1',
            'setup_devices_android_list_2',
            'setup_devices_android_list_3',
            'setup_devices_android_list_4',
        ],
    },
    iOS: {
        title: 'iOS',
        icon: 'ios',
        list: [
            'setup_devices_ios_list_1',
            'setup_devices_ios_list_2',
            'setup_devices_ios_list_3',
            'setup_devices_ios_list_4',
        ],
    },
    dns_privacy: {
        title: t('dns_privacy'),
        icon: 'dns_privacy',
        getTitle: () => (
            <DnsPrivacyTitle
                tlsAddress={tlsAddress}
                httpsAddress={httpsAddress}
                showDnsPrivacyNotice={showDnsPrivacyNotice}
                serverName={serverName}
                portHttps={portHttps}
                t={t}
                dnsAddresses={dnsAddresses}
            />
        ),
    },
});

interface renderContentProps {
    title: string;
    list?: unknown[];
    getTitle?: (...args: unknown[]) => unknown;
    subtitle_1?: string;
    subtitle_2?: string;
    subtitle_2_components?: unknown[];
    strongNumbers?: boolean;
}

const renderContent = (
    { title, list, getTitle, subtitle_1, subtitle_2, subtitle_2_components, strongNumbers }:
    renderContentProps
) => (
    <div title={i18next.t(title)}>
        <div className={s.title}>{i18next.t(title)}</div>

        <div className={s.text}>
            {getTitle?.()}
            {subtitle_1 && (
                <div className={s.paragraph}>
                    <Trans>{subtitle_1}</Trans>
                </div>
            )}
            {list && (
                <ol className={cn({ [s.strongNumbers]: strongNumbers })}>
                    {list.map((item: any, index: number) => (
                        <li key={typeof item === 'string' ? item : item.label || index}>
                            {typeof item === 'string' ? (
                                <Trans>{item}</Trans>
                            ) : (
                                <Trans
                                    components={item.components?.map((props: any) => {
                                        if (React.isValidElement(props)) {
                                            return props;
                                        }
                                        const {
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
                                    {item.label}
                                </Trans>
                            )}
                        </li>
                    ))}
                </ol>
            )}
            {subtitle_2 && (
                <div className={s.paragraph}>
                    <Trans
                        components={subtitle_2_components?.map((props: any) => {
                            if (React.isValidElement(props)) {
                                return props;
                            }
                            const {
                                href,
                                target = '_self',
                                rel = 'noopener noreferrer',
                                key = '0',
                            } = props;

                            return (
                                <a href={href} target={target} rel={rel} key={key} className={s.dhcpLink}>
                                    link
                                </a>
                            );
                        })}>
                        {subtitle_2}
                    </Trans>
                </div>
            )}
        </div>
    </div>
);

interface GuideProps {
    dnsAddresses?: unknown[];
}

export const Guide = ({ dnsAddresses }: GuideProps) => {
    const { t } = useTranslation();

    const serverName = useSelector((state: RootState) => state.encryption?.server_name);

    const portHttps = useSelector((state: RootState) => state.encryption?.port_https);
    const tlsAddress = dnsAddresses?.filter((item: any) => item.includes('tls://')) ?? '';
    const httpsAddress = dnsAddresses?.filter((item: any) => item.includes('https://')) ?? '';
    const showDnsPrivacyNotice = httpsAddress.length < 1 && tlsAddress.length < 1;

    const [activeTabLabel, setActiveTabLabel] = useState('Router');

    const tabsData = getTabs({
        tlsAddress,
        httpsAddress,
        showDnsPrivacyNotice,
        serverName,
        portHttps,
        t,
        dnsAddresses,
    });

    const selectOptions = Object.entries(tabsData).map(([key, value]: [string, any]) => ({
        value: key,
        label: value.title,
        icon: value.icon as IconType,
    }));


    const tabs = Object.entries(tabsData).map(([key, value]: [string, any]) => ({
        id: key,
        label: value.title,
        content: renderContent({
            title: value.title,
            list: value.list,
            getTitle: value.getTitle,
            subtitle_1: value.subtitle_1,
            subtitle_2: value.subtitle_2,
            subtitle_2_components: value.subtitle_2_components,
            strongNumbers: value.strongNumbers,
        }),
        icon: value.icon as IconType,
    }));

    const selectedOption = selectOptions.find(option => option.value === activeTabLabel);
    const activeTab = tabs.find(tab => tab.id === activeTabLabel);

    return (
        <div className={s.deviceSelectorContainer}>
            <p className={s.selectorDesc}>{t('device_type')}</p>
            <div className={s.deviceSelector}>
                <Select
                    options={selectOptions}
                    value={selectedOption}
                    onChange={(option) => setActiveTabLabel(option.value)}
                    showIcons={true}
                    size="responsive"
                    height="big"
                />
            </div>
            {activeTab && (
                <div className={s.deviceContent}>
                    {activeTab.content}
                </div>
            )}
        </div>
    );
};

Guide.defaultProps = {
    dnsAddresses: [],
};
