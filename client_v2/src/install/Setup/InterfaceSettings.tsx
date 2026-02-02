import React, { useCallback } from 'react';

import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import styles from 'panel/install/Setup/styles.module.pcss';
import cn from 'clsx';
import Controls from './Controls';
import { WebBanner } from './blocks/Banner';
import { AddressList } from './blocks';
import { buildInterfaceOptions } from './helpers/InterfaceOptions';
import { createHandleAutofix, useInstallSettingsForm } from './helpers/useInstallSettingsForm';

import {
    ALL_INTERFACES_IP,
    STATUS_RESPONSE,
    STANDARD_WEB_PORT,
} from '../../helpers/constants';

import { InstallInterface } from '../../initialState';

import type { ConfigType, DnsConfig, SettingsFormValues, StaticIpType, WebConfig } from './types';

type Props = {
    handleSubmit: (data: SettingsFormValues) => void;
    handleChange?: (data: SettingsFormValues) => void;
    handleFix: (web: WebConfig, dns: DnsConfig, set_static_ip: boolean) => void;
    validateForm: (data: SettingsFormValues) => void;
    config: ConfigType;
    interfaces: InstallInterface[];
    initialValues?: object;
};

export const InterfaceSettings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {

    const {
        control,
        reactHookFormSubmit,
        isValid,
        watchFields,
        webIpVal,
        webPortVal,
    } = useInstallSettingsForm(config, validateForm);

    const { status: webStatus, can_autofix: isWebFixAvailable } = config.web;

    const { staticIp } = config;

    const webIpOptions = buildInterfaceOptions(interfaces);

    const handleAutofix = createHandleAutofix(watchFields, handleFix);

    const handleStaticIp = (ip: string) => {
        const web = {
            ip: watchFields.web?.ip,
            port: watchFields.web?.port,
            autofix: false,
        };
        const dns = {
            ip: watchFields.dns?.ip,
            port: watchFields.dns?.port,
            autofix: false,
        };
        const set_static_ip = true;

        if (window.confirm(intl.getMessage('confirm_static_ip', { ip }))) {
            handleFix(web, dns, set_static_ip);
        }
    };

    const getStaticIpMessage = useCallback(
        (staticIp: StaticIpType) => {
            const { static: status, ip } = staticIp;

            switch (status) {
                case STATUS_RESPONSE.NO:
                    return (
                        <>
                            <div className={styles.spacerBottom}>
                                {intl.getMessage('install_static_configure', { ip }).replace('{ip}', ip)}
                            </div>

                            <Button
                                type="button"
                                size="small"
                                variant="secondary"
                                className={styles.button}
                                onClick={() => handleStaticIp(ip)}>
                                {intl.getMessage('set_static_ip')}
                            </Button>
                        </>
                    );
                case STATUS_RESPONSE.ERROR:
                    return (
                        <div className={styles.errorText}>
                            {intl.getMessage('install_static_error')}
                        </div>
                    );
                case STATUS_RESPONSE.YES:
                    return (
                        <div className={styles.successText}>
                            {intl.getMessage('install_static_ok')}
                        </div>
                    );
                default:
                    return null;
            }
        },
        [handleStaticIp],
    );

    const onSubmit = (data: SettingsFormValues) => {
        validateForm(data);
        handleSubmit(data);
    };

    return (
        <div className={styles.configSetting}>
            <form className={styles.step} onSubmit={reactHookFormSubmit(onSubmit)}>

                <div className={styles.info}>
                    <div>
                        <div className={styles.titleStep}>{intl.getMessage('setup_ui_title')}</div>

                        <p className={styles.descAdresses}>{intl.getMessage('setup_ui_desc')}</p>

                        <WebBanner
                            className={cn(styles.banner, styles.bannerMobile)}
                            control={control}
                            webIpOptions={webIpOptions}
                            webStatus={webStatus}
                            isWebFixAvailable={isWebFixAvailable}
                            onAutofix={() => handleAutofix('web')}
                        />
                    </div>

                    <AddressList
                        interfaces={interfaces}
                        address={webIpVal || ALL_INTERFACES_IP}
                        port={webPortVal || STANDARD_WEB_PORT}
                    />

                    <div className={styles.group}>
                        {getStaticIpMessage(staticIp)}
                    </div>

                    <Controls invalid={!isValid} />
                </div>

                <div className={styles.content}>
                    <WebBanner
                        className={styles.banner}
                        control={control}
                        webIpOptions={webIpOptions}
                        webStatus={webStatus}
                        isWebFixAvailable={isWebFixAvailable}
                        onAutofix={() => handleAutofix('web')}
                    />
                </div>
            </form>
        </div>
    );
};
