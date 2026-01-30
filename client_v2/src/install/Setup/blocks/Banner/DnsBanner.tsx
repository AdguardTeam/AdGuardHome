import React from 'react';
import { Controller, type Control } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';

import cn from 'clsx';

import {
    ADDRESS_IN_USE_TEXT,
    PORT_53_FAQ_LINK,
    STANDARD_DNS_PORT,
    STANDARD_WEB_PORT,
} from 'panel/helpers/constants';

import { validateRequiredValue, validateInstallPort } from 'panel/helpers/validators';
import { toNumber } from 'panel/helpers/form';
import styles from './styles.module.pcss';

import type { SettingsFormValues } from '../../types';

type SelectOption = {
    value: string;
    label: string;
    isDisabled: boolean;
};

type Props = {
    className: string;
    control: Control<SettingsFormValues>;
    dnsIpOptions: SelectOption[];
    dnsStatus?: string;
    isDnsFixAvailable: boolean;
    dnsPortVal?: number;
    onAutofix: () => void;
};

export const DnsBanner = ({
    className,
    control,
    dnsIpOptions,
    dnsStatus,
    isDnsFixAvailable,
    dnsPortVal,
    onAutofix,
}: Props) => (
    <div className={className}>
        <div className={styles.bannerInputs}>
            <div className={styles.group}>
                <div className={styles.bannerTitle}>
                    {intl.getMessage('setup_dns_title_banner')}
                </div>

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
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

                <div className={styles.form}>
                    <label className={styles.bannerLabel}>
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
                        render={({ field, fieldState }) => {
                            const isPortInUse = Boolean(dnsStatus && dnsStatus.includes(ADDRESS_IN_USE_TEXT));
                            const errorMessage = fieldState.error?.message || (isPortInUse ? intl.getMessage('port_in_use') : undefined);
                            return (
                                <Input
                                    {...field}
                                    type="number"
                                    id="install_dns_port"
                                    errorMessage={errorMessage}
                                    placeholder={STANDARD_WEB_PORT.toString()}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                />
                            );
                        }}
                    />
                </div>

                <div>
                    {dnsStatus && (
                        <>
                            <div className={cn(styles.setupError, styles.errorRow, styles.errorText)}>
                                {dnsStatus}
                                {isDnsFixAvailable && (
                                    <Button
                                        type="button"
                                        id="install_dns_fix"
                                        size="small"
                                        variant="primary"
                                        className={styles.inlineButton}
                                        onClick={onAutofix}>
                                    </Button>
                                )}
                            </div>
                            {isDnsFixAvailable && (
                                <div className={styles.mutedText}>
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
                    {dnsPortVal === STANDARD_DNS_PORT &&
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
