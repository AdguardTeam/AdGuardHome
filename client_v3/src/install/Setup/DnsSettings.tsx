import { createMemo, untrack } from 'solid-js';

import intl from 'panel/common/intl';
import styles from 'panel/install/Setup/styles.module.pcss';
import cn from 'clsx';
import { Controls } from './Controls';

import { AddressList } from './blocks';
import { DnsBanner } from './blocks/Banner';
import { buildInterfaceOptions } from './helpers/InterfaceOptions';
import { createHandleAutofix, useInstallSettingsForm } from './helpers/useInstallSettingsForm';

import { InstallInterface } from '../../initialState';

import type { ConfigType, DnsConfig, SettingsFormValues, WebConfig } from './types';

type Props = {
    handleSubmit: (data: SettingsFormValues) => void;
    handleChange?: (data: SettingsFormValues) => void;
    handleFix: (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => void;
    validateForm: (data: SettingsFormValues) => void;
    config: ConfigType;
    interfaces: InstallInterface[];
    initialValues?: object;
};

export const DnsSettings = (props: Props) => {
    const form = useInstallSettingsForm(
        untrack(() => props.config),
        untrack(() => props.validateForm),
    );

    const dnsStatus = () => props.config.dns.status;
    const isDnsFixAvailable = () => props.config.dns.can_autofix;

    const dnsIpOptions = createMemo(() => buildInterfaceOptions(props.interfaces));

    const handleAutofix = createHandleAutofix(
        form.watchFields,
        untrack(() => props.handleFix),
    );

    const onSubmit = (data: SettingsFormValues) => {
        props.validateForm(data);
        props.handleSubmit(data);
    };

    return (
        <div class={styles.configSetting}>
            <form class={styles.step} onSubmit={(e) => form.handleSubmit(onSubmit)(e)}>
                <div class={styles.info}>
                    <div>
                        <div class={styles.titleStep}>{intl.getMessage('setup_dns_title')}</div>

                        <p class={styles.descAdresses}>{intl.getMessage('setup_dns_desc')}</p>

                        <DnsBanner
                            class={cn(styles.banner, styles.bannerMobile)}
                            dnsIp={form.dnsIp}
                            dnsPort={form.dnsPort}
                            setDnsIp={form.setDnsIp}
                            setDnsPort={form.setDnsPort}
                            dnsIpOptions={dnsIpOptions()}
                            dnsStatus={dnsStatus()}
                            isDnsFixAvailable={isDnsFixAvailable()}
                            onAutofix={() => handleAutofix('dns')}
                        />
                    </div>
                    <div class={styles.addressListWrapper}>
                        <AddressList
                            interfaces={props.interfaces}
                            address={form.dnsIp()}
                            port={form.dnsPort()}
                            isDns={true}
                        />
                    </div>
                    <div class={styles.quote}>
                        <div class={styles.quoteTitle}>
                            {intl.getMessage('setup_dns_quote_title')}
                        </div>
                        <div class={styles.quoteDesc}>
                            {intl.getMessage('setup_dns_quote_desc')}
                        </div>
                    </div>

                    <Controls invalid={!form.isValid()} />
                </div>
                <div class={styles.content}>
                    <DnsBanner
                        class={styles.banner}
                        dnsIp={form.dnsIp}
                        dnsPort={form.dnsPort}
                        setDnsIp={form.setDnsIp}
                        setDnsPort={form.setDnsPort}
                        dnsIpOptions={dnsIpOptions()}
                        dnsStatus={dnsStatus()}
                        isDnsFixAvailable={isDnsFixAvailable()}
                        onAutofix={() => handleAutofix('dns')}
                    />
                </div>
            </form>
        </div>
    );
};
