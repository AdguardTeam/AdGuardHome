import React, { useEffect, useCallback } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import i18n from 'i18next';

import Controls from './Controls';
import AddressList from './AddressList';

import { getInterfaceIp } from '../../helpers/helpers';
import {
    ALL_INTERFACES_IP,
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STATUS_RESPONSE,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
    MAX_PORT,
    MIN_PORT,
} from '../../helpers/constants';

import { validateRequiredValue } from '../../helpers/validators';
import { InstallInterface } from '../../initialState';
import { Input } from '../../components/ui/Controls/Input';
import { Select } from '../../components/ui/Controls/Select';
import { toNumber } from '../../helpers/form';

const validateInstallPort = (value: number) => {
    if (value < MIN_PORT || value > MAX_PORT) {
        return i18n.t('form_error_port');
    }
    return undefined;
};

export type WebConfig = {
    ip: string;
    port: number;
};

export type DnsConfig = {
    ip: string;
    port: number;
};

export type SettingsFormValues = {
    web: WebConfig;
    dns: DnsConfig;
};

type StaticIpType = {
    ip: string;
    static: string;
};

export type ConfigType = {
    web: {
        ip: string;
        port?: number;
        status: string;
        can_autofix: boolean;
    };
    dns: {
        ip: string;
        port?: number;
        status: string;
        can_autofix: boolean;
    };
    staticIp: StaticIpType;
};

type Props = {
    handleSubmit: (data: SettingsFormValues) => void;
    handleChange?: (data: SettingsFormValues) => unknown;
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
                    {name} - {ip} {!isUp && `(${i18n.t('down')})`}
                </option>
            );
        }

        return null;
    });

export const Settings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {
    const { t } = useTranslation();

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

        if (window.confirm(t('confirm_static_ip', { ip }))) {
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
                            <div className="mb-2">
                                <Trans values={{ ip }} components={[<strong key="0">text</strong>]}>
                                    install_static_configure
                                </Trans>
                            </div>

                            <button
                                type="button"
                                className="btn btn-outline-primary btn-sm"
                                onClick={() => handleStaticIp(ip)}>
                                <Trans>set_static_ip</Trans>
                            </button>
                        </>
                    );
                case STATUS_RESPONSE.ERROR:
                    return (
                        <div className="text-danger">
                            <Trans>install_static_error</Trans>
                        </div>
                    );
                case STATUS_RESPONSE.YES:
                    return (
                        <div className="text-success">
                            <Trans>install_static_ok</Trans>
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
        <form className="setup__step" onSubmit={reactHookFormSubmit(onSubmit)}>
            <div className="setup__group">
                <div className="setup__subtitle">
                    <Trans>install_settings_title</Trans>
                </div>

                <div className="row">
                    <div className="col-8">
                        <div className="form-group">
                            <label>
                                <Trans>install_settings_listen</Trans>
                            </label>
                            <Controller
                                name="web.ip"
                                control={control}
                                render={({ field }) => (
                                    <Select {...field} data-testid="install_web_ip">
                                        <option value={ALL_INTERFACES_IP}>
                                            {t('install_settings_all_interfaces')}
                                        </option>
                                        {renderInterfaces(interfaces)}
                                    </Select>
                                )}
                            />
                        </div>
                    </div>

                    <div className="col-4">
                        <div className="form-group">
                            <label>
                                <Trans>install_settings_port</Trans>
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
                                        data-testid="install_web_port"
                                        placeholder={STANDARD_WEB_PORT.toString()}
                                        error={fieldState.error?.message}
                                        onChange={(e) => {
                                            const { value } = e.target;
                                            field.onChange(toNumber(value));
                                        }}
                                    />
                                )}
                            />
                        </div>
                    </div>

                    <div className="col-12">
                        {webStatus && (
                            <div className="setup__error text-danger">
                                {webStatus}
                                {isWebFixAvailable && (
                                    <button
                                        type="button"
                                        data-testid="install_web_fix"
                                        className="btn btn-secondary btn-sm ml-2"
                                        onClick={() => handleAutofix('web')}>
                                        <Trans>fix</Trans>
                                    </button>
                                )}
                            </div>
                        )}

                        <hr className="divider--small" />
                    </div>
                </div>

                <div className="setup__desc">
                    <Trans>install_settings_interface_link</Trans>

                    <div className="mt-1">
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.web?.ip}
                            port={watchFields.web?.port}
                        />
                    </div>
                </div>
            </div>

            <div className="setup__group">
                <div className="setup__subtitle">
                    <Trans>install_settings_dns</Trans>
                </div>

                <div className="row">
                    <div className="col-8">
                        <div className="form-group">
                            <label>
                                <Trans>install_settings_listen</Trans>
                            </label>
                            <Controller
                                name="dns.ip"
                                control={control}
                                render={({ field }) => (
                                    <Select {...field} data-testid="install_dns_ip">
                                        <option value={ALL_INTERFACES_IP}>
                                            {t('install_settings_all_interfaces')}
                                        </option>
                                        {renderInterfaces(interfaces)}
                                    </Select>
                                )}
                            />
                        </div>
                    </div>

                    <div className="col-4">
                        <div className="form-group">
                            <label>
                                <Trans>install_settings_port</Trans>
                            </label>
                            <Controller
                                name="dns.port"
                                control={control}
                                rules={{
                                    required: t('form_error_required'),
                                    validate: {
                                        required: validateRequiredValue,
                                        installPort: validateInstallPort,
                                    },
                                }}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        type="number"
                                        data-testid="install_dns_port"
                                        error={fieldState.error?.message}
                                        placeholder={STANDARD_WEB_PORT.toString()}
                                        onChange={(e) => {
                                            const { value } = e.target;
                                            field.onChange(toNumber(value));
                                        }}
                                    />
                                )}
                            />
                        </div>
                    </div>

                    <div className="col-12">
                        {dnsStatus && (
                            <>
                                <div className="setup__error text-danger">
                                    {dnsStatus}
                                    {isDnsFixAvailable && (
                                        <button
                                            type="button"
                                            data-testid="install_dns_fix"
                                            className="btn btn-secondary btn-sm ml-2"
                                            onClick={() => handleAutofix('dns')}>
                                            <Trans>fix</Trans>
                                        </button>
                                    )}
                                </div>
                                {isDnsFixAvailable && (
                                    <div className="text-muted mb-2">
                                        <p className="mb-1">
                                            <Trans>autofix_warning_text</Trans>
                                        </p>
                                        <Trans components={[<li key="0">text</li>]}>autofix_warning_list</Trans>
                                        <p className="mb-1">
                                            <Trans>autofix_warning_result</Trans>
                                        </p>
                                    </div>
                                )}
                            </>
                        )}
                        {watchFields.dns?.port === STANDARD_DNS_PORT &&
                            !isDnsFixAvailable &&
                            dnsStatus?.includes(ADDRESS_IN_USE_TEXT) && (
                                <Trans
                                    components={[
                                        <a href={PORT_53_FAQ_LINK} key="0" target="_blank" rel="noopener noreferrer">
                                            link
                                        </a>,
                                    ]}>
                                    port_53_faq_link
                                </Trans>
                            )}

                        <hr className="divider--small" />
                    </div>
                </div>

                <div className="setup__desc">
                    <Trans>install_settings_dns_desc</Trans>

                    <div className="mt-1">
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.dns?.ip}
                            port={watchFields.dns?.port}
                            isDns={true}
                        />
                    </div>
                </div>
            </div>

            <div className="setup__group">
                <div className="setup__subtitle">
                    <Trans>static_ip</Trans>
                </div>

                <div className="mb-2">
                    <Trans>static_ip_desc</Trans>
                </div>

                {getStaticIpMessage(staticIp)}
            </div>

            <Controls invalid={!isValid} />
        </form>
    );
};
