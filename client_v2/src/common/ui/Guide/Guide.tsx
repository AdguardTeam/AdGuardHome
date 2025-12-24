import React, { useState } from 'react';
import intl from 'panel/common/intl';
import { useSelector } from 'react-redux';
import cn from 'clsx';

import { RootState } from 'panel/initialState';
import { MOBILE_CONFIG_LINKS } from 'panel/helpers/constants';
import { MobileConfigForm } from 'panel/components/SetupGuide/MobileConfigForm';
import { Select } from 'panel/common/controls/Select';
import { IconType } from '../Icons';
import { CopiedText } from '../CopiedText';
import s from './Guide.module.pcss';

type PlatformLayoutProps = {
    serverName?: string;
    portHttps?: number;
    dnsAddresses?: string[];
}

type PlatformLayout = {
    title: string;
    icon: IconType;
    component: React.ReactElement;
}

type PlatformLayouts = Record<string, PlatformLayout>;

const RouterLayout = () => (
    <div className={s.guideContent}>
        <div className={s.title}>
            {intl.getMessage('setup_devices_router_title')}
        </div>
        <div className={s.guideText}>
            <div className={s.guideParagraph}>
                {intl.getMessage('setup_devices_router_desc_1')}
            </div>
            <ol className={cn({ [s.strongNumbers]: true })}>
                <li className={s.guideItem}>
                    <strong className={s.guideItemTitle}>
                        {intl.getMessage('setup_devices_router_list_1_title')}
                    </strong>
                    {intl.getMessage('setup_devices_router_list_1', {
                        code: () => <CopiedText text="http://192.168.0.1" />,
                        code2: () => <CopiedText text="http://192.168.1.1" />,
                    })}
                </li>
                <li className={s.guideItem}>
                    <strong className={s.guideItemTitle}>
                        {intl.getMessage('setup_devices_router_list_2_title')}
                    </strong>
                    {intl.getMessage('setup_devices_router_list_2')}
                </li>
            </ol>
            <div className={s.guideParagraph}>
                {intl.getMessage('setup_devices_router_desc_2', {
                    a: (text: string) => (
                        <a
                            href="#dhcp"
                            target="_blank"
                            className={s.dnsLink}
                        >
                            {text}
                        </a>
                    ),
                })}
            </div>
        </div>
    </div>
);

const WindowsLayout = () => (
    <div title="Windows">
        <div className={s.title}>
            {intl.getMessage('setup_devices_windows_title')}
        </div>
        <div className={s.text}>
            <ol className={s.guideList}>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_windows_list_1')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_windows_list_2')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_windows_list_3')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_windows_list_4')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_windows_list_5')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_windows_list_6')}</li>
            </ol>
        </div>
    </div>
);

const MacOSLayout = () => (
    <div title="macOS">
        <div className={s.title}>macOS</div>
        <div className={s.text}>
            <ol className={s.guideList}>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_macos_list_1')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_macos_list_2')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_macos_list_3')}</li>
            </ol>
        </div>
    </div>
);

const AndroidLayout = () => (
    <div title="Android">
        <div className={s.title}>Android</div>
        <div className={s.text}>
            <ol className={s.guideList}>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_android_list_1')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_android_list_2')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_android_list_3')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_android_list_4')}</li>
            </ol>
        </div>
    </div>
);

const IOSLayout = () => (
    <div title="iOS">
        <div className={s.title}>iOS</div>
        <div className={s.text}>
            <ol className={s.guideList}>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_ios_list_1')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_ios_list_2')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_ios_list_3')}</li>
                <li className={s.guideItem}>{intl.getMessage('setup_devices_ios_list_4')}</li>
            </ol>
        </div>
    </div>
);

const renderDnsDevicesList = () => (
    <div className={s.deviceDnsList}>
        <div className={s.guideParagraph}>
            <div className={s.guideTitle}>
                <strong>Android</strong>
            </div>
            <ul className={s.guideList}>
                <li className={s.guideBulletItem}>{intl.getMessage('setup_devices_dns_android_list_1')}</li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_android_list_2', {
                        a: (text: string) => (
                            <a
                                href="https://link.adtidy.org/forward.html?action=android&from=ui&app=home"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_android_list_3', {
                        a: (text: string) => (
                            <a
                                href="https://getintra.org/"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
            </ul>
        </div>

        <div className={s.guideParagraph}>
            <div className={s.guideTitle}>
                <strong>iOS</strong>
            </div>
            <ul className={s.guideList}>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_ios_list_1', {
                        a: (text: string) => (
                            <a
                                href="https://link.adtidy.org/forward.html?action=ios&from=ui&app=home"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_ios_list_2', {
                        a: (text: string) => (
                            <a
                                href="https://itunes.apple.com/app/id1452162351"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                        b: (text: string) => (
                            <a
                                href="https://dnscrypt.info/stamps"
                                target="_blank"
                                rel="noopener noreferrer"
                                className={s.dnsLink}
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
            </ul>
        </div>

        <div className={s.guideParagraph}>
            <div className={s.guideTitle}>
                <strong>{intl.getMessage('setup_dns_privacy_other_title')}</strong>
            </div>
            <ul className={s.guideList}>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_1')}
                </li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_2', {
                        a: (text: string) => (
                            <a
                                href="https://github.com/AdguardTeam/dnsproxy"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_3', {
                        a: (text: string) => (
                            <a
                                href="https://github.com/jedisct1/dnscrypt-proxy"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_4', {
                        a: (text: string) => (
                            <a
                                href="https://support.mozilla.org/kb/firefox-dns-over-https"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li className={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_5', {
                        a: (text: string) => (
                            <a
                                href="https://dnscrypt.info/implementations"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                        b: (text: string) => (
                            <a
                                href="https://dnsprivacy.org/wiki/display/DP/DNS+Privacy+Clients"
                                target="_blank"
                                className={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
            </ul>
        </div>
    </div>
);

const getDnsSettingsContent = (dnsAddresses: string[] | undefined, serverName?: string, portHttps?: number) => {
    const tlsAddress = dnsAddresses?.filter((addr: string) => addr.includes('tls://')) ?? [];
    const httpsAddress = dnsAddresses?.filter((addr: string) => addr.includes('https://')) ?? [];
    const quicAddress = dnsAddresses?.find((addr: string) => addr.includes('quic://'));

    const showDnsPrivacyNotice = httpsAddress.length < 1 && tlsAddress.length < 1;

    return (
        showDnsPrivacyNotice ? (
            <div className={s.guideParagraph}>
                {intl.getMessage('setup_dns_notice_new', {
                    a: (text: string) => (
                        <a
                            href="#encryption"
                            rel="noopener noreferrer"
                            key="a"
                            className={s.dnsLink}
                        >
                            {text}
                        </a>
                    ),
                })}
            </div>
        ) : (
            <div className={s.dnsSettingsContent}>
                <ul className={s.deviceDnsList}>
                    {tlsAddress.length > 0 && (
                        <li className={s.deviceDnsListItem}>
                            {intl.getMessage('setup_devices_dns_list_1', {
                                code: <CopiedText text={tlsAddress[0]} />
                            })}
                        </li>
                    )}
                    {httpsAddress.length > 0 && (
                        <li className={s.deviceDnsListItem}>
                            {intl.getMessage('setup_devices_dns_list_2', {
                                code: () => <CopiedText text={httpsAddress[0]} />
                            })}
                        </li>
                    )}
                    {quicAddress && (
                        <li className={s.deviceDnsListItem}>
                            {intl.getMessage('setup_devices_dns_list_3', {
                                code: () => <CopiedText text={quicAddress} />
                            })}
                        </li>
                    )}
                </ul>

                <div className={s.guideParagraph}>
                    {intl.getMessage('setup_devices_dns_list_devices')}
                </div>

                {renderDnsDevicesList()}

                <div className={s.guideParagraph}>
                    <div className={s.guideTitle}>
                        <strong>{intl.getMessage('setup_dns_privacy_ioc_mac')}</strong>
                    </div>
                    <div>
                        {intl.getMessage('setup_devices_dns_macos_desc')}
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
            </div>
        )
    );
};

const DnsPrivacyLayout = ({ serverName, portHttps, dnsAddresses }: PlatformLayoutProps) => (
    <div title={intl.getMessage('dns_privacy')}>
        <div className={s.title}>{intl.getMessage('dns_privacy')}</div>
        <div className={s.text}>
            {getDnsSettingsContent(dnsAddresses, serverName, portHttps)}
        </div>
    </div>
);

const getPlatformLayouts = ({ serverName, portHttps, dnsAddresses }: PlatformLayoutProps): PlatformLayouts => ({
    Router: {
        title: intl.getMessage('setup_devices_router_title'),
        icon: 'router',
        component: <RouterLayout />,
    },
    Windows: {
        title: intl.getMessage('setup_devices_windows_title'),
        icon: 'windows',
        component: <WindowsLayout />,
    },
    macOS: {
        title: intl.getMessage('setup_devices_macos_title'),
        icon: 'mac',
        component: <MacOSLayout />,
    },
    Android: {
        title: intl.getMessage('setup_devices_android_title'),
        icon: 'android',
        component: <AndroidLayout />,
    },
    iOS: {
        title: intl.getMessage('setup_devices_ios_title'),
        icon: 'ios',
        component: <IOSLayout />,
    },
    dns_privacy: {
        title: intl.getMessage('dns_privacy'),
        icon: 'dns_privacy',
        component: (
            <DnsPrivacyLayout
                serverName={serverName}
                portHttps={portHttps}
                dnsAddresses={dnsAddresses}
            />
        ),
    },
});


interface GuideProps {
    dnsAddresses?: string[];
}

export const Guide = ({ dnsAddresses }: GuideProps) => {

    const serverName = useSelector((state: RootState) => state.encryption?.server_name);
    const portHttps = useSelector((state: RootState) => state.encryption?.port_https);

    const [activeTabLabel, setActiveTabLabel] = useState('Router');

    const platformLayouts = getPlatformLayouts({
        serverName,
        portHttps,
        dnsAddresses,
    });

    const selectOptions = Object.entries(platformLayouts).map(([key, value]: [string, PlatformLayout]) => ({
        value: key,
        label: value.title,
        icon: value.icon,
    }));

    const selectedOption = selectOptions.find(option => option.value === activeTabLabel);
    const activeLayout = platformLayouts[activeTabLabel as keyof typeof platformLayouts];

    return (
        <div className={s.deviceSelectorContainer}>
            <p className={s.selectorDesc}>{intl.getMessage('device_type')}</p>
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
            {activeLayout && (
                <div className={s.deviceContent}>
                    {activeLayout.component}
                </div>
            )}
        </div>
    );
};
