import React from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useSelector } from 'react-redux';

import { UINT32_RANGE } from '../../../helpers/constants';
import {
    validateGatewaySubnetMask,
    validateIpForGatewaySubnetMask,
    validateIpv4,
    validateIpv4RangeEnd,
    validateNotInRange,
    validateRequiredValue,
} from '../../../helpers/validators';
import { RootState } from '../../../initialState';

type FormValues = {
    v4?: {
        gateway_ip?: string;
        subnet_mask?: string;
        range_start?: string;
        range_end?: string;
        lease_duration?: number;
    };
}

type FormDHCPv4Props = {
    processingConfig?: boolean;
    initialValues?: FormValues;
    ipv4placeholders?: {
        gateway_ip: string;
        subnet_mask: string;
        range_start: string;
        range_end: string;
        lease_duration: string;
    };
    onSubmit?: (data: FormValues) => Promise<void> | void;
}

const FormDHCPv4 = ({ 
    processingConfig,
    initialValues,
    ipv4placeholders,
    onSubmit 
}: FormDHCPv4Props) => {
    const { t } = useTranslation();

    const interfaces = useSelector((state: RootState) => state.form.DHCP_INTERFACES);
    const interface_name = interfaces?.values?.interface_name;

    const isInterfaceIncludesIpv4 = useSelector(
        (state: RootState) => !!state.dhcp?.interfaces?.[interface_name]?.ipv4_addresses,
    );

    const {
        register,
        handleSubmit,
        formState: { errors, isSubmitting },
        watch,
    } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
            v4: initialValues?.v4 || {
                gateway_ip: '',
                subnet_mask: '',
                range_start: '',
                range_end: '',
                lease_duration: 0,
            },
        },
    });

    const formValues = watch('v4');
    const isEmptyConfig = !Object.values(formValues || {}).some(Boolean);

    const handleFormSubmit = async (data: FormValues) => {
        if (onSubmit) {
            await onSubmit(data);
        }
    };

    return (
        <form onSubmit={handleSubmit(handleFormSubmit)}>
            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label>{t('dhcp_form_gateway_input')}</label>
                        <input
                            type="text"
                            className="form-control"
                            placeholder={t(ipv4placeholders?.gateway_ip || '')}
                            disabled={!isInterfaceIncludesIpv4}
                            {...register('v4.gateway_ip', {
                                validate: {
                                    ipv4: validateIpv4,
                                    required: (value) => isEmptyConfig ? undefined : validateRequiredValue(value),
                                    notInRange: validateNotInRange,
                                }
                            })}
                        />
                        {errors.v4?.gateway_ip && (
                            <div className="form__message form__message--error">
                                {t(errors.v4.gateway_ip.message)}
                            </div>
                        )}
                    </div>

                    <div className="form__group form__group--settings">
                        <label>{t('dhcp_form_subnet_input')}</label>
                        <input
                            type="text"
                            className="form-control"
                            placeholder={t(ipv4placeholders?.subnet_mask || '')}
                            disabled={!isInterfaceIncludesIpv4}
                            {...register('v4.subnet_mask', {
                                validate: {
                                    required: (value) => isEmptyConfig ? undefined : validateRequiredValue(value),
                                    subnet: validateGatewaySubnetMask,
                                }
                            })}
                        />
                        {errors.v4?.subnet_mask && (
                            <div className="form__message form__message--error">
                                {t(errors.v4.subnet_mask.message)}
                            </div>
                        )}
                    </div>
                </div>

                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <div className="row">
                            <div className="col-12">
                                <label>{t('dhcp_form_range_title')}</label>
                            </div>

                            <div className="col">
                                <input
                                    type="text"
                                    className="form-control"
                                    placeholder={t(ipv4placeholders?.range_start || '')}
                                    disabled={!isInterfaceIncludesIpv4}
                                    {...register('v4.range_start', {
                                        validate: {
                                            ipv4: validateIpv4,
                                            gateway: validateIpForGatewaySubnetMask,
                                        }
                                    })}
                                />
                                {errors.v4?.range_start && (
                                    <div className="form__message form__message--error">
                                        {t(errors.v4.range_start.message)}
                                    </div>
                                )}
                            </div>

                            <div className="col">
                                <input
                                    type="text"
                                    className="form-control"
                                    placeholder={t(ipv4placeholders?.range_end || '')}
                                    disabled={!isInterfaceIncludesIpv4}
                                    {...register('v4.range_end', {
                                        validate: {
                                            ipv4: validateIpv4,
                                            rangeEnd: validateIpv4RangeEnd,
                                            gateway: validateIpForGatewaySubnetMask,
                                        }
                                    })}
                                />
                                {errors.v4?.range_end && (
                                    <div className="form__message form__message--error">
                                        {errors.v4.range_end.message}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>

                    <div className="form__group form__group--settings">
                        <label>{t('dhcp_form_lease_title')}</label>
                        <input
                            type="number"
                            className="form-control"
                            placeholder={t(ipv4placeholders?.lease_duration || '')}
                            disabled={!isInterfaceIncludesIpv4}
                            min={1}
                            max={UINT32_RANGE.MAX}
                            {...register('v4.lease_duration', {
                                valueAsNumber: true,
                                validate: {
                                    required: (value) => isEmptyConfig ? undefined : validateRequiredValue(value),
                                }
                            })}
                        />
                        {errors.v4?.lease_duration && (
                            <div className="form__message form__message--error">
                                {t(errors.v4.lease_duration.message)}
                            </div>
                        )}
                    </div>
                </div>
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    className="btn btn-success btn-standard"
                    disabled={isSubmitting || processingConfig || !isInterfaceIncludesIpv4 || isEmptyConfig || Object.keys(errors).length > 0}
                >
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default FormDHCPv4;
