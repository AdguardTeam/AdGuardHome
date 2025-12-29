import React, { useEffect, useCallback } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import i18n from 'i18next';

import { Input } from 'panel/common/controls/Input';
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
import { toNumber } from '../../helpers/form';
import intl from 'panel/common/intl';

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

export const InterfaceSettings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {
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
        <div className="setup__config-setting">
            <form className="setup__step" onSubmit={reactHookFormSubmit(onSubmit)}>

                <div className="setup__left-side">
                    <div className="setup__subtitle">
                        <div className="setup__title">{intl.getMessage('setup_ui_title')}</div>

                        <p className="setup__desc">{intl.getMessage('setup_ui_desc')}</p>
                    </div>

                    <div className="setup__group">
                        {getStaticIpMessage(staticIp)}
                    </div>

                    <Controls invalid={!isValid} />
                </div>

                <div className="setup__right-side">
                    <div className="setup__banner">
                        <h3 className="setup__banner-title">{intl.getMessage('setup_ui_title_banner')}</h3>
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
                                            <select {...field} id="install_web_ip">
                                                <option value={ALL_INTERFACES_IP}>
                                                    {t('install_settings_all_interfaces')}
                                                </option>
                                                {renderInterfaces(interfaces)}
                                            </select>
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
                            </div>

                            <div className="col-12">
                                {webStatus && (
                                    <div className="setup__error text-danger">
                                        {webStatus}
                                        {isWebFixAvailable && (
                                            <button
                                                type="button"
                                                id="install_web_fix"
                                                className="btn btn-secondary btn-sm ml-2"
                                                onClick={() => handleAutofix('web')}>
                                                <Trans>fix</Trans>
                                            </button>
                                        )}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            </form>
        </div>
    );
};
