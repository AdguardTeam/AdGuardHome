import { type JSX, createSignal, createMemo, Show } from 'solid-js';
import { A } from '@solidjs/router';
import intl from 'panel/common/intl';
import cn from 'clsx';

import { encryptionState } from 'panel/stores/encryption';
import { MOBILE_CONFIG_LINKS } from 'panel/helpers/constants';
import { MobileConfigForm } from 'panel/components/SetupGuide/MobileConfigForm';
import { Select } from 'panel/common/controls/Select';
import { Paths } from 'panel/components/Routes/Paths';
import { type IconType } from '../Icons';
import { CopiedText } from '../CopiedText';
import s from './Guide.module.pcss';

type PlatformLayoutProps = {
    serverName?: string;
    portHttps?: number;
    dnsAddresses?: string[];
};

type PlatformLayout = {
    title: string;
    icon: IconType;
    component: JSX.Element;
};

type PlatformLayouts = Record<string, PlatformLayout>;

const RouterLayout = () => (
    <div class={s.guideContent}>
        <div class={s.title}>{intl.getMessage('setup_devices_router_title')}</div>
        <div class={s.guideText}>
            <div class={s.guideParagraph}>{intl.getMessage('setup_devices_router_desc_1')}</div>
            <ol class={cn({ [s.strongNumbers]: true })}>
                <li class={s.guideItem}>
                    <strong class={s.guideItemTitle}>
                        {intl.getMessage('setup_devices_router_list_1_title')}
                    </strong>
                    {intl.getMessage('setup_devices_router_list_1', {
                        code: () => <CopiedText text="http://192.168.0.1" />,
                        code2: () => <CopiedText text="http://192.168.1.1" />,
                    })}
                </li>
                <li class={s.guideItem}>
                    <strong class={s.guideItemTitle}>
                        {intl.getMessage('setup_devices_router_list_2_title')}
                    </strong>
                    {intl.getMessage('setup_devices_router_list_2')}
                </li>
            </ol>
            <div class={s.guideParagraph}>
                {intl.getMessage('setup_devices_router_desc_2', {
                    a: (text: string) => (
                        <A href={Paths.Dhcp} class={s.dnsLink}>
                            {text}
                        </A>
                    ),
                })}
            </div>
        </div>
    </div>
);

const WindowsLayout = () => (
    <div title="Windows">
        <div class={s.title}>{intl.getMessage('setup_devices_windows_title')}</div>
        <div class={s.text}>
            <ol class={s.guideList}>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_windows_list_1')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_windows_list_2')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_windows_list_3')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_windows_list_4')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_windows_list_5')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_windows_list_6')}</li>
            </ol>
        </div>
    </div>
);

const MacOSLayout = () => (
    <div title="macOS">
        <div class={s.title}>macOS</div>
        <div class={s.text}>
            <ol class={s.guideList}>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_macos_list_1')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_macos_list_2')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_macos_list_3')}</li>
            </ol>
        </div>
    </div>
);

const AndroidLayout = () => (
    <div title="Android">
        <div class={s.title}>Android</div>
        <div class={s.text}>
            <ol class={s.guideList}>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_android_list_1')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_android_list_2')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_android_list_3')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_android_list_4')}</li>
            </ol>
        </div>
    </div>
);

const IOSLayout = () => (
    <div title="iOS">
        <div class={s.title}>iOS</div>
        <div class={s.text}>
            <ol class={s.guideList}>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_ios_list_1')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_ios_list_2')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_ios_list_3')}</li>
                <li class={s.guideItem}>{intl.getMessage('setup_devices_ios_list_4')}</li>
            </ol>
        </div>
    </div>
);

const renderDnsDevicesList = () => (
    <div class={s.deviceDnsList}>
        <div class={s.guideParagraph}>
            <div class={s.guideTitle}>
                <strong>Android</strong>
            </div>
            <ul class={s.guideList}>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_android_list_1')}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_android_list_2', {
                        a: (text: string) => (
                            <a
                                href="https://link.adtidy.org/forward.html?action=android&from=ui&app=home"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_android_list_3', {
                        a: (text: string) => (
                            <a
                                href="https://getintra.org/"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
            </ul>
        </div>

        <div class={s.guideParagraph}>
            <div class={s.guideTitle}>
                <strong>iOS</strong>
            </div>
            <ul class={s.guideList}>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_ios_list_1', {
                        a: (text: string) => (
                            <a
                                href="https://link.adtidy.org/forward.html?action=ios&from=ui&app=home"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_ios_list_2', {
                        a: (text: string) => (
                            <a
                                href="https://itunes.apple.com/app/id1452162351"
                                target="_blank"
                                class={s.dnsLink}
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
                                class={s.dnsLink}
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
            </ul>
        </div>

        <div class={s.guideParagraph}>
            <div class={s.guideTitle}>
                <strong>{intl.getMessage('setup_dns_privacy_other_title')}</strong>
            </div>
            <ul class={s.guideList}>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_1', {
                        a: (text: string) => (
                            <a
                                href="https://github.com/AdguardTeam/AdGuardHome"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_2', {
                        a: (text: string) => (
                            <a
                                href="https://github.com/AdguardTeam/dnsproxy"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_3', {
                        a: (text: string) => (
                            <a
                                href="https://github.com/jedisct1/dnscrypt-proxy"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_4', {
                        a: (text: string) => (
                            <a
                                href="https://support.mozilla.org/kb/firefox-dns-over-https"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                    })}
                </li>
                <li class={s.guideBulletItem}>
                    {intl.getMessage('setup_devices_dns_other_list_5', {
                        a: (text: string) => (
                            <a
                                href="https://dnscrypt.info/implementations"
                                target="_blank"
                                class={s.dnsLink}
                                rel="noopener noreferrer"
                            >
                                {text}
                            </a>
                        ),
                        b: (text: string) => (
                            <a
                                href="https://dnsprivacy.org/wiki/display/DP/DNS+Privacy+Clients"
                                target="_blank"
                                class={s.dnsLink}
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

const getDnsSettingsContent = (
    dnsAddresses: string[] | undefined,
    serverName?: string,
    portHttps?: number,
) => {
    const tlsAddress = dnsAddresses?.filter((addr: string) => addr.includes('tls://')) ?? [];
    const httpsAddress = dnsAddresses?.filter((addr: string) => addr.includes('https://')) ?? [];
    const quicAddress = dnsAddresses?.find((addr: string) => addr.includes('quic://'));

    const showDnsPrivacyNotice = httpsAddress.length < 1 && tlsAddress.length < 1;

    return showDnsPrivacyNotice ? (
        <div class={s.guideParagraph}>
            {intl.getMessage('setup_dns_notice_new', {
                a: (text: string) => (
                    <A href={Paths.Encryption} class={s.dnsLink}>
                        {text}
                    </A>
                ),
            })}
        </div>
    ) : (
        <div class={s.dnsSettingsContent}>
            <ul class={s.deviceDnsList}>
                <Show when={tlsAddress.length > 0}>
                    <li class={s.deviceDnsListItem}>
                        {intl.getMessage('setup_devices_dns_list_1', {
                            code: () => <CopiedText text={tlsAddress[0]} />,
                        })}
                    </li>
                </Show>
                <Show when={httpsAddress.length > 0}>
                    <li class={s.deviceDnsListItem}>
                        {intl.getMessage('setup_devices_dns_list_2', {
                            code: () => <CopiedText text={httpsAddress[0]} />,
                        })}
                    </li>
                </Show>
                <Show when={quicAddress}>
                    <li class={s.deviceDnsListItem}>
                        {intl.getMessage('setup_devices_dns_list_3', {
                            code: () => <CopiedText text={quicAddress!} />,
                        })}
                    </li>
                </Show>
            </ul>

            <div class={s.guideParagraph}>{intl.getMessage('setup_devices_dns_list_devices')}</div>

            {renderDnsDevicesList()}

            <div class={s.guideParagraph}>
                <div class={s.guideTitle}>
                    <strong>{intl.getMessage('setup_dns_privacy_ioc_mac')}</strong>
                </div>
                <div>{intl.getMessage('setup_devices_dns_macos_desc')}</div>
                <div class={s.mobileConfigContainer}>
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
    );
};

const DnsPrivacyLayout = (props: PlatformLayoutProps) => (
    <div title={intl.getMessage('dns_privacy')}>
        <div class={s.title}>{intl.getMessage('dns_privacy')}</div>
        <div class={s.text}>
            {getDnsSettingsContent(props.dnsAddresses, props.serverName, props.portHttps)}
        </div>
    </div>
);

const getPlatformLayouts = (params: PlatformLayoutProps): PlatformLayouts => ({
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
                serverName={params.serverName}
                portHttps={params.portHttps}
                dnsAddresses={params.dnsAddresses}
            />
        ),
    },
});

type Props = {
    dnsAddresses?: string[];
};

export const Guide = (props: Props) => {
    const serverName = () => encryptionState.server_name;
    const portHttps = () => encryptionState.port_https;

    const [activeTabLabel, setActiveTabLabel] = createSignal('Router');

    const platformLayouts = () =>
        getPlatformLayouts({
            serverName: serverName(),
            portHttps: portHttps(),
            dnsAddresses: props.dnsAddresses,
        });

    const selectOptions = createMemo(() =>
        Object.entries(platformLayouts()).map(([key, value]) => ({
            value: key,
            label: value.title,
            icon: value.icon,
        })),
    );

    const selectedOption = createMemo(() =>
        selectOptions().find((option) => option.value === activeTabLabel()),
    );

    const activeLayout = () => platformLayouts()[activeTabLabel() as keyof typeof platformLayouts];

    return (
        <div class={s.deviceSelectorContainer}>
            <p class={s.selectorDesc}>{intl.getMessage('device_type')}</p>
            <div class={s.deviceSelector}>
                <Select
                    options={selectOptions()}
                    value={selectedOption()}
                    onChange={(option: any) => setActiveTabLabel(option.value)}
                    showIcons={true}
                    size="responsive"
                    height="big"
                />
            </div>
            <Show when={activeLayout()}>
                <div class={s.deviceContent}>{activeLayout().component}</div>
            </Show>
        </div>
    );
};
