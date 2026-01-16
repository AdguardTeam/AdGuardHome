import React, { useEffect, useCallback } from 'react';
import { useForm, Controller } from 'react-hook-form';
import intl from 'panel/common/intl';

import { Input } from 'panel/common/controls/Input';
import { Button } from 'panel/common/ui/Button';
import Controls from './Controls';
import cn from 'clsx';
import AddressList from './AddressList';

import { getInterfaceIp } from '../../helpers/helpers';
import {
    ALL_INTERFACES_IP,
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STATUS_RESPONSE,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
} from '../../helpers/constants';

import { validateRequiredValue, validateInstallPort } from '../../helpers/validators';
import { InstallInterface } from '../../initialState';
import { toNumber } from '../../helpers/form';
import styles from './styles.module.pcss';

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

const renderInterfaces = (interfaces: InstallInterface[]) =>
    Object.values(interfaces).map((option: InstallInterface) => {
        const { name, ip_addresses, flags } = option;

        if (option && ip_addresses?.length > 0) {
            const ip = getInterfaceIp(option);
            const isUp = flags?.includes('up');

            return (
                <option value={ip} key={name} disabled={!isUp}>
                    {name} - {ip} {!isUp && `(${intl.getMessage('down')})`}
                </option>
            );
        }

        return null;
    });

export const Settings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {

    const defaultValues = {
        web: {
            ip: config.web.ip || ALL_INTERFACES_IP,
            port: config.web.port || STANDARD_WEB_PORT,
        },
        dns: {
            ip: config.dns.ip || ALL_INTERFACES_IP,
            port: config.dns.port || STANDARD_DNS_PORT,
        },
    };

    const {
        control,
        watch,
        handleSubmit: reactHookFormSubmit,
        formState: { isValid },
    } = useForm<SettingsFormValues>({
        defaultValues,
        mode: 'onBlur',
    });

    const watchFields = watch();

    const { status: webStatus, can_autofix: isWebFixAvailable } = config.web;
    const { status: dnsStatus, can_autofix: isDnsFixAvailable } = config.dns;
    const { staticIp } = config;

    const webIpVal = watch('web.ip');
    const webPortVal = watch('web.port');
    const dnsIpVal = watch('dns.ip');
    const dnsPortVal = watch('dns.port');

    useEffect(() => {
        const webPortError = validateInstallPort(webPortVal);
        const dnsPortError = validateInstallPort(dnsPortVal);

        if (webPortError || dnsPortError) {
            return;
        }

        validateForm({
            web: {
                ip: webIpVal,
                port: webPortVal,
            },
            dns: {
                ip: dnsIpVal,
                port: dnsPortVal,
            },
        });
    }, [webIpVal, webPortVal, dnsIpVal, dnsPortVal]);

    const handleAutofix = (type: string) => {
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
        const set_static_ip = false;

        if (type === 'web') {
            web.autofix = true;
        } else {
            dns.autofix = true;
        }

        handleFix(web, dns, set_static_ip);
    };

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
        <form className={styles.step} onSubmit={reactHookFormSubmit(onSubmit)}>
            <div className={styles.group}>
                <div className={styles.subtitle}>
                    {intl.getMessage('install_settings_title')}
                </div>

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
                        {intl.getMessage('network_interface')}
                    </label>
                    <Controller
                        name="web.ip"
                        control={control}
                        render={({ field }) => (
                            <select {...field} id="install_web_ip">
                                <option value={ALL_INTERFACES_IP}>
                                    {intl.getMessage('install_settings_all_interfaces')}
                                </option>
                                {renderInterfaces(interfaces)}
                            </select>
                        )}
                    />
                </div>

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
                        {intl.getMessage('install_settings_port')}
                    </label>
                    <Controller
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
                        <div className={cn(styles.setup__error, styles.errorRow, styles.errorText)}>
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

                <div className={styles.desc}>
                    {intl.getMessage('install_settings_interface_link')}

                    <div>
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.web?.ip}
                            port={watchFields.web?.port}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.group}>
                <div className={styles.subtitle}>
                    {intl.getMessage('install_settings_dns')}
                </div>

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
                        {intl.getMessage('network_interface')}
                    </label>
                    <Controller
                        name="dns.ip"
                        control={control}
                        render={({ field }) => (
                            <select {...field} id="install_dns_ip">
                                <option value={ALL_INTERFACES_IP}>
                                    {intl.getMessage('install_settings_all_interfaces')}
                                </option>
                                {renderInterfaces(interfaces)}
                            </select>
                        )}
                    />
                </div>

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
                        {intl.getMessage('install_settings_port')}
                    </label>
                    <Controller
                        name="dns.port"
                        control={control}
                        rules={{
                            required: intl.getMessage('form_error_required'),
                            validate: {
                                required: validateRequiredValue,
                                installPort: validateInstallPort,
                            },
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="number"
                                id="install_dns_port"
                                errorMessage={fieldState.error?.message}
                                placeholder={STANDARD_WEB_PORT.toString()}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>

                <div>
                    {dnsStatus && (
                        <>
                            <div className={cn(styles.setup__error, styles.errorRow, styles.errorText)}>
                                {dnsStatus}
                                {isDnsFixAvailable && (
                                    <Button
                                        type="button"
                                        id="install_dns_fix"
                                        size="small"
                                        variant="secondary"
                                        className={styles.inlineButton}
                                        onClick={() => handleAutofix('dns')}>
                                        {intl.getMessage('fix')}
                                    </Button>
                                )}
                            </div>
                            {isDnsFixAvailable && (
                                <div className={cn(styles.mutedText, styles.spacerBottom)}>
                                    <p className={styles.compactParagraph}>
                                        {intl.getMessage('autofix_warning_text')}
                                    </p>
                                    {intl.getMessage('autofix_warning_list')}
                                    <p className={styles.compactParagraph}>
                                        {intl.getMessage('autofix_warning_result')}
                                    </p>
                                </div>
                            )}
                        </>
                    )}
                    {watchFields.dns?.port === STANDARD_DNS_PORT &&
                        !isDnsFixAvailable &&
                        dnsStatus?.includes(ADDRESS_IN_USE_TEXT) && (
                            <a href={PORT_53_FAQ_LINK} target="_blank" rel="noopener noreferrer">
                                {intl.getMessage('port_53_faq_link')}
                            </a>
                        )}
                </div>

                <div className={styles.desc}>
                    {intl.getMessage('install_settings_dns_desc')}

                    <div>
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.dns?.ip}
                            port={watchFields.dns?.port}
                            isDns={true}
                        />
                    </div>
                </div>
            </div>

            <div className={styles.group}>
                <div className={styles.subtitle}>
                    {intl.getMessage('static_ip')}
                </div>

                <div className={styles.spacerBottom}>
                    {intl.getMessage('static_ip_desc')}
                </div>

                {getStaticIpMessage(staticIp)}
            </div>

            <Controls invalid={!isValid} />
        </form>
    );
};
