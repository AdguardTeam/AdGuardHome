import React from 'react';
import { Controller } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import setup from 'panel/install/Setup/styles.module.pcss';
import Controls from './Controls';

import AddressList from './AddressList';
import { buildInterfaceOptions } from './interfaceOptions';
import { createHandleAutofix, useInstallSettingsForm } from './useInstallSettingsForm';

import {
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
} from '../../helpers/constants';

import { validateRequiredValue, validateInstallPort } from '../../helpers/validators';
import { InstallInterface } from '../../initialState';
import { toNumber } from '../../helpers/form';

import type { ConfigType, DnsConfig, SettingsFormValues, WebConfig } from './types';

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

    const DnsBanner = ({ className }: { className: string }) => (
        <div className={className}>
            <div className={setup.bannerInputs}>
                <div className={setup.group}>
                    <div className={setup.bannerTitle}>
                        {intl.getMessage('setup_dns_title_banner')}
                    </div>

                    <div className={setup.form}>
                        <label className={setup.bannerLabel}>
                            {intl.getMessage('network_interface')}
                        </label>
                        <Controller<SettingsFormValues, 'dns.ip'>
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
                    </div>

                    <div className={setup.form}>
                        <label className={setup.bannerLabel}>
                            {intl.getMessage('install_settings_port')}
                        </label>
                        <Controller<SettingsFormValues, 'dns.port'>
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
                                <div className={`${setup.setup__error} ${setup.errorRow} ${setup.errorText}`}>
                                    {dnsStatus}
                                    {isDnsFixAvailable && (
                                        <Button
                                            type="button"
                                            id="install_dns_fix"
                                            size="small"
                                            variant="secondary"
                                            className={setup.inlineButton}
                                            onClick={() => handleAutofix('dns')}>
                                            {intl.getMessage('fix')}
                                        </Button>
                                    )}
                                </div>
                                {isDnsFixAvailable && (
                                    <div className={setup.mutedText}>
                                        <p className={setup.compactParagraph}>
                                            {intl.getMessage('autofix_warning_text')}
                                        </p>
                                        {intl.getMessage('autofix_warning_list')}
                                        <p className={setup.compactParagraph}>
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
