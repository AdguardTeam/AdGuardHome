import React, { useMemo } from 'react';
import { Controller, useFormContext } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import { UINT32_RANGE } from '../../../helpers/constants';
import {
    validateGatewaySubnetMask,
    validateIpForGatewaySubnetMask,
    validateIpv4,
    validateIpv4RangeEnd,
    validateNotInRange,
    validateRequiredValue,
} from '../../../helpers/validators';
import { DhcpFormValues } from '.';
import { Input } from '../../ui/Controls/Input';
import { toNumber } from '../../../helpers/form';

type FormDHCPv4Props = {
    processingConfig?: boolean;
    ipv4placeholders?: {
        gateway_ip: string;
        subnet_mask: string;
        range_start: string;
        range_end: string;
        lease_duration: string;
    };
    interfaces: any;
    onSubmit?: (data: DhcpFormValues) => void;
};

const FormDHCPv4 = ({ processingConfig, ipv4placeholders, interfaces, onSubmit }: FormDHCPv4Props) => {
    const { t } = useTranslation();

    const {
        handleSubmit,
        formState: { errors, isSubmitting },
        control,
        watch,
    } = useFormContext<DhcpFormValues>();

    const interfaceName = watch('interface_name');
    const isInterfaceIncludesIpv4 = interfaces?.[interfaceName]?.ipv4_addresses;

    const formValues = watch('v4');
    const isEmptyConfig = !Object.values(formValues || {}).some(Boolean);
    const hasV4Errors = errors.v4 && Object.keys(errors.v4).length > 0;

    const isDisabled = useMemo(() => {
        return isSubmitting || hasV4Errors || processingConfig || !isInterfaceIncludesIpv4 || isEmptyConfig;
    }, [isSubmitting, hasV4Errors, processingConfig, isInterfaceIncludesIpv4, isEmptyConfig]);

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="v4.gateway_ip"
                            control={control}
                            rules={{
                                validate: {
                                    ipv4: validateIpv4,
                                    required: (value) => (isEmptyConfig ? undefined : validateRequiredValue(value)),
                                    notInRange: validateNotInRange,
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    data-testid="v4_gateway_ip"
                                    label={t('dhcp_form_gateway_input')}
                                    placeholder={t(ipv4placeholders.gateway_ip)}
                                    error={fieldState.error?.message}
                                    disabled={!isInterfaceIncludesIpv4}
                                />
                            )}
                        />
                    </div>

                    <div className="form__group form__group--settings">
                        <Controller
                            name="v4.subnet_mask"
                            control={control}
                            rules={{
                                validate: {
                                    required: (value) => (isEmptyConfig ? undefined : validateRequiredValue(value)),
                                    subnet: validateGatewaySubnetMask,
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    data-testid="v4_subnet_mask"
                                    label={t('dhcp_form_subnet_input')}
                                    placeholder={t(ipv4placeholders.subnet_mask)}
                                    error={fieldState.error?.message}
                                    disabled={!isInterfaceIncludesIpv4}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-lg-6">
                    <div className="form__group mb-0">
                        <div className="row">
                            <div className="col-12">
                                <label>{t('dhcp_form_range_title')}</label>
                            </div>

                            <div className="col">
                                <Controller
                                    name="v4.range_start"
                                    control={control}
                                    rules={{
                                        validate: {
                                            ipv4: validateIpv4,
                                            gateway: validateIpForGatewaySubnetMask,
                                        },
                                    }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="v4_range_start"
                                            placeholder={t(ipv4placeholders.range_start)}
                                            error={fieldState.error?.message}
                                            disabled={!isInterfaceIncludesIpv4}
                                        />
                                    )}
                                />
                            </div>

                            <div className="col">
                                <Controller
                                    name="v4.range_end"
                                    control={control}
                                    rules={{
                                        validate: {
                                            ipv4: validateIpv4,
                                            rangeEnd: validateIpv4RangeEnd,
                                            gateway: validateIpForGatewaySubnetMask,
                                        },
                                    }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            data-testid="v4_range_end"
                                            placeholder={t(ipv4placeholders.range_end)}
                                            error={fieldState.error?.message}
                                            disabled={!isInterfaceIncludesIpv4}
                                        />
                                    )}
                                />
                            </div>
                        </div>
                    </div>

                    <div className="form__group form__group--settings">
                        <Controller
                            name="v4.lease_duration"
                            control={control}
                            rules={{
                                validate: {
                                    required: (value) => (isEmptyConfig ? undefined : validateRequiredValue(value)),
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    data-testid="v4_lease_duration"
                                    label={t('dhcp_form_lease_title')}
                                    placeholder={t(ipv4placeholders.lease_duration)}
                                    error={fieldState.error?.message}
                                    disabled={!isInterfaceIncludesIpv4}
                                    min={1}
                                    max={UINT32_RANGE.MAX}
                                    value={field.value ?? ''}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                />
                            )}
                        />
                    </div>
                </div>
            </div>

            <div className="btn-list">
                <button
                    data-testid="v4_save"
                    type="submit"
                    className="btn btn-success btn-standard"
                    disabled={isDisabled}>
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default FormDHCPv4;
