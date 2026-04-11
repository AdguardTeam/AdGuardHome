import React, { useMemo } from 'react';
import { Controller, useFormContext } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import {
    DHCP_V6_PREFIX_SOURCE_OPTIONS,
    DHCP_V6_PREFIX_SOURCE_VALUES,
    UINT32_RANGE,
} from '../../../helpers/constants';
import { validateIpv6, validateRequiredValue } from '../../../helpers/validators';
import { DhcpFormValues } from '.';
import { Input } from '../../ui/Controls/Input';
import { Select } from '../../ui/Controls/Select';
import { toNumber } from '../../../helpers/form';

type FormDHCPv6Props = {
    processingConfig?: boolean;
    ipv6placeholders?: {
        range_start: string;
        range_end: string;
        lease_duration: string;
    };
    interfaces: any;
    onSubmit?: (data: DhcpFormValues) => Promise<void> | void;
};

const FormDHCPv6 = ({ processingConfig, ipv6placeholders, interfaces, onSubmit }: FormDHCPv6Props) => {
    const { t } = useTranslation();
    const {
        handleSubmit,
        formState: { isSubmitting, isValid },
        control,
        watch,
    } = useFormContext<DhcpFormValues>();

    const interfaceName = watch('interface_name');
    const isInterfaceIncludesIpv6 = interfaces?.[interfaceName]?.ipv6_addresses;
    const prefixSource = watch('v6.prefix_source') ?? DHCP_V6_PREFIX_SOURCE_VALUES.STATIC;
    const isRASLAACOnly = watch('v6.ra_slaac_only');
    const canConfigureV6 = Boolean(isInterfaceIncludesIpv6) || prefixSource === DHCP_V6_PREFIX_SOURCE_VALUES.INTERFACE;
    const isInterfaceRASLAACOnly = (
        prefixSource === DHCP_V6_PREFIX_SOURCE_VALUES.INTERFACE
        && isRASLAACOnly
    );
    const shouldRequireRangeStart = !isInterfaceRASLAACOnly;
    const shouldRequireLeaseDuration = !isInterfaceRASLAACOnly;

    const formValues = watch('v6');
    const isEmptyConfig = !Object.entries(formValues || {}).some(
        ([key, value]) => key !== 'prefix_source' && Boolean(value),
    );

    const isDisabled = useMemo(() => {
        return isSubmitting || !isValid || processingConfig || !canConfigureV6 || isEmptyConfig;
    }, [isSubmitting, isValid, processingConfig, canConfigureV6, isEmptyConfig]);

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="row">
                <div className="col-lg-6 form__group form__group--settings">
                    <Controller
                        name="v6.prefix_source"
                        control={control}
                        render={({ field }) => (
                            <Select
                                {...field}
                                value={field.value ?? DHCP_V6_PREFIX_SOURCE_VALUES.STATIC}
                                label={t('dhcp_form_prefix_source_title')}
                                data-testid="v6_prefix_source"
                                disabled={!interfaceName}>
                                {DHCP_V6_PREFIX_SOURCE_OPTIONS.map((value) => (
                                    <option key={value} value={value}>
                                        {t(
                                            value === DHCP_V6_PREFIX_SOURCE_VALUES.STATIC
                                                ? 'dhcp_form_prefix_source_static'
                                                : 'dhcp_form_prefix_source_interface',
                                        )}
                                    </option>
                                ))}
                            </Select>
                        )}
                    />
                </div>
            </div>

            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group mb-0">
                        <div className="row">
                            <div className="col-12">
                                <label>{t('dhcp_form_range_title')}</label>
                            </div>

                            <div className="col">
                                <Controller
                                    name="v6.range_start"
                                    control={control}
                                    rules={{
                                        validate: canConfigureV6
                                            ? {
                                                  ipv6: validateIpv6,
                                                  required: shouldRequireRangeStart
                                                      ? validateRequiredValue
                                                      : undefined,
                                              }
                                            : undefined,
                                    }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="v6_range_start"
                                            label={
                                                prefixSource === DHCP_V6_PREFIX_SOURCE_VALUES.INTERFACE
                                                    ? t('dhcp_form_host_template_input')
                                                    : t('dhcp_form_range_start')
                                            }
                                            placeholder={t(ipv6placeholders.range_start)}
                                            desc={
                                                prefixSource === DHCP_V6_PREFIX_SOURCE_VALUES.INTERFACE
                                                    ? t('dhcp_form_host_template_desc')
                                                    : undefined
                                            }
                                            error={fieldState.error?.message}
                                            disabled={!canConfigureV6}
                                        />
                                    )}
                                />
                            </div>

                            <div className="col">
                                <Controller
                                    name="v6.range_end"
                                    control={control}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="v6_range_end"
                                            placeholder={t(ipv6placeholders.range_end)}
                                            error={fieldState.error?.message}
                                            disabled
                                        />
                                    )}
                                />
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row">
                <div className="col-lg-6 form__group form__group--settings">
                    <Controller
                        name="v6.lease_duration"
                        control={control}
                        rules={{
                            validate: canConfigureV6
                                ? {
                                      required: shouldRequireLeaseDuration
                                          ? validateRequiredValue
                                          : undefined,
                                  }
                                : undefined,
                        }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                type="number"
                                data-testid="v6_lease_duration"
                                label={t('dhcp_form_lease_title')}
                                placeholder={t(ipv6placeholders.lease_duration)}
                                error={fieldState.error?.message}
                                disabled={!canConfigureV6}
                                min={1}
                                max={UINT32_RANGE.MAX}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>
            </div>

            <div className="btn-list">
                <button
                    data-testid="v6_save"
                    type="submit"
                    className="btn btn-success btn-standard"
                    disabled={isDisabled}>
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default FormDHCPv6;
