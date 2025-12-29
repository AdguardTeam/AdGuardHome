import React, { useEffect } from 'react';
import { useForm, Controller } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import i18n from 'i18next';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import Controls from './Controls';

import AddressList from './AddressList';

import {
    ALL_INTERFACES_IP,
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
    MAX_PORT,
    MIN_PORT,
} from '../../helpers/constants';

import { validateRequiredValue } from '../../helpers/validators';
import { InstallInterface } from '../../initialState';
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

export const DnsSettings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {
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

    const { status: dnsStatus, can_autofix: isDnsFixAvailable } = config.dns;

    const webIpVal = watch('web.ip');
    const webPortVal = watch('web.port');
    const dnsIpVal = watch('dns.ip');
    const dnsPortVal = watch('dns.port');

    const dnsIpOptions = [
        { value: ALL_INTERFACES_IP, label: intl.getMessage('install_settings_all_interfaces') },
        ...(Array.isArray(interfaces) ? interfaces.map(iface => ({
            value: iface.ip_addresses[0],
            label: `${iface.name} - ${iface.ip_addresses[0]}`
        })) : []),
    ];

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

    const onSubmit = (data: SettingsFormValues) => {
        validateForm(data);
        handleSubmit(data);
    };

    return (
        <div className="setup__config-setting">
            <form className="setup__step" onSubmit={reactHookFormSubmit(onSubmit)}>
                <div className="setup__left-side">
                    <div className="setup__subtitle">
                        <div className="setup__title">{intl.getMessage('setup_dns_title')}</div>

                        <p className="setup__desc">{intl.getMessage('setup_dns_desc')}</p>
                    </div>
                    <div className="mt-1">
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.dns?.ip}
                            port={watchFields.dns?.port}
                            isDns={true}
                        />
                    </div>
                    <div className="setup__quote">
                        <div className="setup__quote-title">
                            {intl.getMessage('setup_dns_quote_title')}
                        </div>
                        <div className="setup__quote-desc">
                            {intl.getMessage(("setup_dns_quote_desc"))}
                        </div>
                    </div>

                    <Controls invalid={!isValid} />
                </div>
                <div className="setup__right-side">
                    <div className="setup__banner">
                        <div className="setup__group">
                            <div className="setup__subtitle">
                                {intl.getMessage("setup_dns_title_banner")}
                            </div>

                            <div className="setup__banner--setting-group">
                                <div className="form-group">
                                    <label>
                                        <Trans>network_interface</Trans>
                                    </label>
                                    <Controller
                                        name="dns.ip"
                                        control={control}
                                        render={({ field }) => (
                                            <Select
                                                options={dnsIpOptions}
                                                value={dnsIpOptions.find(option => option.value === field.value)}
                                                onChange={(selectedOption) => field.onChange(selectedOption?.value)}
                                                placeholder={intl.getMessage('network_interface')}
                                                size="medium"
                                                height="big"
                                                id="install_dns_ip"
                                            />
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
                            </div>

                            <div className="col-12">
                                {dnsStatus && (
                                    <>
                                        <div className="setup__error text-danger">
                                            {dnsStatus}
                                            {isDnsFixAvailable && (
                                                <button
                                                    type="button"
                                                    id="install_dns_fix"
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
                            </div>
                        </div>
                    </div>
                </div>
            </form>
        </div>
    );
};
