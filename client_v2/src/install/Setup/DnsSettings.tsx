import React from 'react';

import intl from 'panel/common/intl';
import styles from 'panel/install/Setup/styles.module.pcss';
import cn from 'clsx';
import Controls from './Controls';

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

export const DnsSettings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {

    const {
        control,
        reactHookFormSubmit,
        isValid,
        watchFields,
    } = useInstallSettingsForm(config, validateForm);

    const { status: dnsStatus, can_autofix: isDnsFixAvailable } = config.dns;

    const dnsIpOptions = buildInterfaceOptions(interfaces);

    const handleAutofix = createHandleAutofix(watchFields, handleFix);

    const onSubmit = (data: SettingsFormValues) => {
        validateForm(data);
        handleSubmit(data);
    };

    return (
        <div className={styles.configSetting}>
            <form className={styles.step} onSubmit={reactHookFormSubmit(onSubmit)}>
                <div className={styles.info}>
                    <div>
                        <div className={styles.titleStep}>{intl.getMessage('setup_dns_title')}</div>

                        <p className={styles.descAdresses}>{intl.getMessage('setup_dns_desc')}</p>

                        <DnsBanner
                            className={cn(styles.banner, styles.bannerMobile)}
                            control={control}
                            dnsIpOptions={dnsIpOptions}
                            dnsStatus={dnsStatus}
                            isDnsFixAvailable={isDnsFixAvailable}
                            dnsPortVal={watchFields.dns?.port}
                            onAutofix={() => handleAutofix('dns')}
                        />
                    </div>
                    <div className={styles.addressListWrapper}>
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.dns?.ip}
                            port={watchFields.dns?.port}
                            isDns={true}
                        />
                    </div>
                    <div className={styles.quote}>
                        <div className={styles.quoteTitle}>
                            {intl.getMessage('setup_dns_quote_title')}
                        </div>
                        <div className={styles.quoteDesc}>
                            {intl.getMessage(("setup_dns_quote_desc"))}
                        </div>
                    </div>

                    <Controls invalid={!isValid} />
                </div>
                <div className={styles.content}>
                    <DnsBanner
                        className={styles.banner}
                        control={control}
                        dnsIpOptions={dnsIpOptions}
                        dnsStatus={dnsStatus}
                        isDnsFixAvailable={isDnsFixAvailable}
                        dnsPortVal={watchFields.dns?.port}
                        onAutofix={() => handleAutofix('dns')}
                    />
                </div>
            </form>
        </div>
    );
};
