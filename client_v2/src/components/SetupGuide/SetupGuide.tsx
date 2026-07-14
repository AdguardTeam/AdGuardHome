import { createMemo, Show, For, type JSX } from 'solid-js';

import { dashboardState } from 'panel/stores/dashboard';
import { Guide } from 'panel/common/ui/Guide/Guide';

import theme from 'panel/lib/theme';

import intl from 'panel/common/intl';
import { CopiedText } from 'panel/common/ui/CopiedText/CopiedText';
import s from './SetupGuide.module.pcss';

type Props = {
    dnsAddresses?: string[];
    isStep?: boolean;
    footer?: JSX.Element;
};

export const SetupGuide = (props: Props) => {
    const dnsAddresses = createMemo(() => props.dnsAddresses ?? dashboardState.dnsAddresses ?? []);

    const encryptedAddresses = createMemo(() =>
        dnsAddresses().filter(
            (address: string) =>
                address.includes('https://') ||
                address.includes('tls://') ||
                address.includes('quic://'),
        ),
    );

    const plainAddresses = createMemo(() =>
        dnsAddresses().filter(
            (address: string) =>
                !address.includes('https://') &&
                !address.includes('tls://') &&
                !address.includes('quic://'),
        ),
    );

    return (
        <div class={props.isStep ? s.stepRoot : theme.layout.container}>
            <div class={s.header}>
                <h1 class={s.pageTitle}>
                    {props.isStep
                        ? intl.getMessage('setup_guide_title')
                        : intl.getMessage('setup_guide')}
                </h1>
                <Show when={!props.isStep}>
                    <div class={s.pageDesc}>{intl.getMessage('setup_guide_desc')}</div>
                </Show>
            </div>

            <div class={s.guidePage}>
                <Show when={!props.isStep}>
                    <h1 class={s.guideTitle}>{intl.getMessage('setup_guide_device_type')}</h1>
                </Show>
                <Guide dnsAddresses={dnsAddresses()} />

                <div class={s.guideDesc}>
                    <h1 class={s.dnsTitle}>{intl.getMessage('home_dns_addresses')}</h1>

                    <p>{intl.getMessage('home_dns_addresses_desc')}</p>

                    <Show when={encryptedAddresses().length > 0}>
                        <div class={s.dnsSubtitle}>
                            {intl.getMessage('encrypted_dns_addresses')}
                        </div>

                        <ul class={s.addressList}>
                            <For each={encryptedAddresses()}>
                                {(ip) => (
                                    <li class={s.address}>
                                        <span class={s.bulletIcon} />
                                        <CopiedText text={ip} />
                                    </li>
                                )}
                            </For>
                        </ul>
                    </Show>

                    <Show when={plainAddresses().length > 0}>
                        <div class={s.dnsSubtitle}>{intl.getMessage('plain_dns_addresses')}</div>

                        <ul class={s.addressList}>
                            <For each={plainAddresses()}>
                                {(ip) => (
                                    <li class={s.address}>
                                        <span class={s.bulletIcon} />
                                        <CopiedText text={ip} />
                                    </li>
                                )}
                            </For>
                        </ul>
                    </Show>
                </div>
            </div>

            <div class={s.footer}>{props.footer}</div>
        </div>
    );
};
