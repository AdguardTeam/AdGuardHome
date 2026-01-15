import React, { useEffect } from 'react';
import { useForm } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import setup from 'panel/install/Setup/styles.module.pcss';
import Controls from './Controls';
import { SetupBannerFormField } from './SetupBannerFormField';

import AddressList from './AddressList';
import { getInterfaceIp } from '../../helpers/helpers';

import {
    ALL_INTERFACES_IP,
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
} from '../../helpers/constants';

import { validateRequiredValue, validateInstallPort } from '../../helpers/validators';
import { InstallInterface } from '../../initialState';
import { toNumber } from '../../helpers/form';

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

    const getInterfaceDisplayName = (iface: InstallInterface) => {
        const zoneAddr = iface?.ip_addresses?.find((addr) => typeof addr === 'string' && addr.includes('%'));
        const zone = zoneAddr?.split('%')[1];

        return zone || iface.name;
    };

    const dnsIpOptions = [
        {
            value: ALL_INTERFACES_IP,
            label: intl.getMessage('install_settings_all_interfaces'),
            isDisabled: false,
        },
        ...(Array.isArray(interfaces)
            ? interfaces
                  .filter((iface) => iface?.ip_addresses?.length > 0)
                  .map((iface) => {
                      const ip = getInterfaceIp(iface);
                      const displayName = getInterfaceDisplayName(iface);
                      const isUp = iface.flags?.includes('up');

                      return {
                          value: ip,
                          label: `${displayName} – ${ip}${!isUp ? ` (${intl.getMessage('down')})` : ''}`,
                          isDisabled: !isUp,
                      };
                  })
            : []),
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

    const DnsBanner = ({ className }: { className: string }) => (
        <div className={className}>
            <div className={setup.bannerInputs}>
                <div className={setup.group}>
                    <div className={setup.bannerTitle}>
                        {intl.getMessage('setup_dns_title_banner')}
                    </div>

                    <SetupBannerFormField
                        label={intl.getMessage('network_interface')}
                        name="dns.ip"
                        control={control}
                        render={({ field }) => (
                            <Select
                                options={dnsIpOptions}
                                value={dnsIpOptions.find((option) => option.value === field.value)}
                                onChange={(selectedOption) => field.onChange(selectedOption?.value)}
                                placeholder={intl.getMessage('network_interface')}
                                size="responsive"
                                height="big"
                                id="install_dns_ip"
                            />
                        )}
                    />

                    <SetupBannerFormField
                        label={intl.getMessage('install_settings_port')}
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

                    <div>
                        {dnsStatus && (
                            <>
                                <div className="setup__error text-danger">
                                    {dnsStatus}
                                    {isDnsFixAvailable && (
                                        <button
                                            type="button"
                                            id="install_dns_fix"
                                            className="btn btn-secondary"
                                            onClick={() => handleAutofix('dns')}>
                                            {intl.getMessage('fix')}
                                        </button>
                                    )}
                                </div>
                                {isDnsFixAvailable && (
                                    <div className="text-muted">
                                        <p className="mb-1">
                                            {intl.getMessage('autofix_warning_text')}
                                        </p>
                                        {intl.getMessage('autofix_warning_list')}
                                        <p className="mb-1">
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
                </div>
            </div>
        </div>
    );

    return (
        <div className={setup.configSetting}>
            <form className={setup.step} onSubmit={reactHookFormSubmit(onSubmit)}>
                <div className={setup.info}>
                    <div>
                        <div className={setup.titleStep}>{intl.getMessage('setup_dns_title')}</div>

                        <p className={setup.descAdresses}>{intl.getMessage('setup_dns_desc')}</p>

                        <DnsBanner className={`${setup.banner} ${setup.bannerMobile}`} />
                    </div>
                    <div className={setup.addressListWrapper}>
                        <AddressList
                            interfaces={interfaces}
                            address={watchFields.dns?.ip}
                            port={watchFields.dns?.port}
                            isDns={true}
                        />
                    </div>
                    <div className={setup.quote}>
                        <div className={setup.quoteTitle}>
                            {intl.getMessage('setup_dns_quote_title')}
                        </div>
                        <div className={setup.quoteDesc}>
                            {intl.getMessage(("setup_dns_quote_desc"))}
                        </div>
                    </div>

                    <Controls invalid={!isValid} />
                </div>
                <div className={setup.content}>
                    <DnsBanner className={setup.banner} />
                </div>
            </form>
        </div>
    );
};
