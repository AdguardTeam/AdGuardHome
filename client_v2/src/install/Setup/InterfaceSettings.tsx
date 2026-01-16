import React, { useCallback } from 'react';
import { Controller } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import styles from 'panel/install/Setup/styles.module.pcss';
import Controls from './Controls';
import AddressList from './AddressList';
import { buildInterfaceOptions } from './interfaceOptions';
import { createHandleAutofix, useInstallSettingsForm } from './useInstallSettingsForm';

import {
    ALL_INTERFACES_IP,
    STATUS_RESPONSE,
    STANDARD_WEB_PORT,
} from '../../helpers/constants';

import { validateRequiredValue, validateInstallPort } from '../../helpers/validators';
import { InstallInterface } from '../../initialState';
import { toNumber } from '../../helpers/form';

import type { ConfigType, DnsConfig, SettingsFormValues, StaticIpType, WebConfig } from './types';

type Props = {
    handleSubmit: (data: SettingsFormValues) => void;
    handleChange?: (data: SettingsFormValues) => unknown;
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

    const WebBanner = ({ className }: { className: string }) => (
        <div className={className}>
            <h3 className={styles.bannerTitle}>{intl.getMessage('setup_ui_title_banner')}</h3>
            <div className={styles.bannerInputs}>
                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
                        {intl.getMessage('network_interface')}
                    </label>
                    <Controller<SettingsFormValues, 'web.ip'>
                        name="web.ip"
                        control={control}
                        render={({ field }) => (
                            <Select
                                options={webIpOptions}
                                value={webIpOptions.find((option) => option.value === field.value)}
                                onChange={(selectedOption) => field.onChange(selectedOption?.value)}
                                placeholder={intl.getMessage('network_interface')}
                                size="responsive"
                                height="big"
                                id="install_web_ip"
                            />
                        )}
                    />
                </div>

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
                        {intl.getMessage('install_settings_port')}
                    </label>
                    <Controller<SettingsFormValues, 'web.port'>
                        name="web.port"
                        control={control}
                        rules={{
                            validate: {
                                required: validateRequiredValue,
                                installPort: validateInstallPort,
                            },
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="number"
                                id="install_web_port"
                                placeholder={STANDARD_WEB_PORT.toString()}
                                errorMessage={fieldState.error?.message}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>

                <div>
                    {webStatus && (
                        <div className={`${styles.setup__error} ${styles.errorRow} ${styles.errorText}`}>
                            {webStatus}
                            {isWebFixAvailable && (
                                <Button
                                    type="button"
                                    id="install_web_fix"
                                    size="small"
                                    variant="secondary"
                                    className={styles.inlineButton}
                                    onClick={() => handleAutofix('web')}>
                                    {intl.getMessage('fix')}
                                </Button>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );

    return (
        <div className={styles.configSetting}>
            <form className={styles.step} onSubmit={reactHookFormSubmit(onSubmit)}>

                <div className={styles.info}>
                    <div>
                        <div className={styles.titleStep}>{intl.getMessage('setup_ui_title')}</div>

                        <p className={styles.descAdresses}>{intl.getMessage('setup_ui_desc')}</p>

                        <WebBanner className={`${styles.banner} ${styles.bannerMobile}`} />
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
                    <WebBanner className={styles.banner} />
                </div>
            </form>
        </div>
    );
};
