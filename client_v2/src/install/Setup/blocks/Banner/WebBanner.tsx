import React from 'react';
import { Controller, type Control } from 'react-hook-form';

import { Input } from 'panel/common/controls/Input';
import { Select } from 'panel/common/controls/Select';
import intl from 'panel/common/intl';
import { Button } from 'panel/common/ui/Button';
import cn from 'clsx';

import { STANDARD_WEB_PORT, ADDRESS_IN_USE_TEXT } from 'panel/helpers/constants';

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
    webIpOptions: SelectOption[];
    webStatus?: string;
    isWebFixAvailable: boolean;
    onAutofix: () => void;
};

export const WebBanner = ({
    className,
    control,
    webIpOptions,
    webStatus,
    isWebFixAvailable,
    onAutofix,
}: Props) => (
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
                    render={({ field, fieldState }) => {
                        const isPortInUse = Boolean(webStatus && webStatus.includes(ADDRESS_IN_USE_TEXT));
                        const errorMessage = fieldState.error?.message || (isPortInUse ? intl.getMessage('port_in_use') : undefined);
                        return (
                            <Input
                                {...field}
                                type="number"
                                id="install_web_port"
                                placeholder={STANDARD_WEB_PORT.toString()}
                                errorMessage={errorMessage}
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
                {webStatus && (
                    <div className={cn(styles.setupError, styles.errorRow, styles.errorText)}>
                        {webStatus}
                        {isWebFixAvailable && (
                            <Button
                                type="button"
                                id="install_web_fix"
                                size="small"
                                variant="primary"
                                className={styles.inlineButton}
                                onClick={onAutofix}>
                            </Button>
                        )}
                    </div>
                )}
            </div>
        </div>
    </div>
);
