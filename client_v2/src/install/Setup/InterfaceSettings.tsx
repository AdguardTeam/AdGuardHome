import React, { useEffect, useCallback } from 'react';
import { useForm } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import setup from 'panel/install/Setup/styles.module.pcss';
import Controls from './Controls';
import AddressList from './AddressList';
import { SetupBannerFormField } from './SetupBannerFormField';

import { getInterfaceIp } from '../../helpers/helpers';
import {
    ALL_INTERFACES_IP,
    STATUS_RESPONSE,
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

export const InterfaceSettings = ({ handleSubmit, handleFix, validateForm, config, interfaces }: Props) => {

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

    const getInterfaceDisplayName = (iface: InstallInterface) => {
        const zoneAddr = iface?.ip_addresses?.find((addr) => typeof addr === 'string' && addr.includes('%'));
        const zone = zoneAddr?.split('%')[1];

        return zone || iface.name;
    };

    const webIpOptions = [
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
                            <div className="mb-2">
                                {intl.getMessage('install_static_configure', { ip }).replace('{ip}', ip)}
                            </div>

                            <button
                                type="button"
                                className="btn btn-outline-primary btn-sm"
                                onClick={() => handleStaticIp(ip)}>
                                {intl.getMessage('set_static_ip')}
                            </button>
                        </>
                    );
                case STATUS_RESPONSE.ERROR:
                    return (
                        <div className="text-danger">
                            {intl.getMessage('install_static_error')}
                        </div>
                    );
                case STATUS_RESPONSE.YES:
                    return (
                        <div className="text-success">
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
            <h3 className={setup.bannerTitle}>{intl.getMessage('setup_ui_title_banner')}</h3>
            <div className={setup.bannerInputs}>
                <SetupBannerFormField
                    label={intl.getMessage('network_interface')}
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

                <SetupBannerFormField
                    outerClassName="col-4"
                    innerClassName="form-group"
                    label={intl.getMessage('install_settings_port')}
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
                                    {intl.getMessage('fix')}
                                </button>
                            )}
                        </div>
                    )}
                </div>
            </div>
        </div>
    );

    return (
        <div className={setup.configSetting}>
            <form className={setup.step} onSubmit={reactHookFormSubmit(onSubmit)}>

                <div className={setup.info}>
                    <div>
                        <div className={setup.titleStep}>{intl.getMessage('setup_ui_title')}</div>

                        <p className={setup.descAdresses}>{intl.getMessage('setup_ui_desc')}</p>

                        <WebBanner className={`${setup.banner} ${setup.bannerMobile}`} />
                    </div>

                    <AddressList
                        interfaces={interfaces}
                        address={webIpVal || ALL_INTERFACES_IP}
                        port={webPortVal || STANDARD_WEB_PORT}
                    />

                    <div className={setup.group}>
                        {getStaticIpMessage(staticIp)}
                    </div>

                    <Controls invalid={!isValid} />
                </div>

                <div className={setup.content}>
                    <WebBanner className={setup.banner} />
                </div>
            </form>
        </div>
    );
};
