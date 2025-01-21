import React, { useMemo } from 'react';
import { useFormContext } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import { UINT32_RANGE } from '../../../helpers/constants';
import { validateIpv6, validateRequiredValue } from '../../../helpers/validators';
import { DhcpFormValues } from '.';

type FormValues = {
    v6?: {
        range_start?: string;
        range_end?: string;
        lease_duration?: number;
    };
};

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
        register,
        handleSubmit,
        formState: { errors, isSubmitting, isValid },
        watch,
    } = useFormContext<DhcpFormValues>();

    const interfaceName = watch('interface_name');
    const isInterfaceIncludesIpv6 = interfaces?.[interfaceName]?.ipv6_addresses;

    const formValues = watch('v6');
    const isEmptyConfig = !Object.values(formValues || {}).some(Boolean);

    const handleFormSubmit = async (data: FormValues) => {
        if (onSubmit) {
            await onSubmit(data);
        }
    };

    const isDisabled = useMemo(() => {
        return isSubmitting || !isValid || processingConfig || !isInterfaceIncludesIpv6 || isEmptyConfig;
    }, [isSubmitting, isValid, processingConfig, isInterfaceIncludesIpv6, isEmptyConfig]);

    return (
        <form onSubmit={handleSubmit(handleFormSubmit)}>
            <div className="row">
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
                                    placeholder={t(ipv6placeholders?.range_start || '')}
                                    disabled={!isInterfaceIncludesIpv6}
                                    {...register('v6.range_start', {
                                        validate: {
                                            ipv6: validateIpv6,
                                            required: (value) =>
                                                isEmptyConfig ? undefined : validateRequiredValue(value),
                                        },
                                    })}
                                />
                                {errors.v6?.range_start && (
                                    <div className="form__message form__message--error">
                                        {t(errors.v6.range_start.message)}
                                    </div>
                                )}
                            </div>

                            <div className="col">
                                <input
                                    type="text"
                                    className="form-control"
                                    placeholder={t(ipv6placeholders?.range_end || '')}
                                    disabled={!isInterfaceIncludesIpv6}
                                    {...register('v6.range_end', {
                                        validate: {
                                            ipv6: validateIpv6,
                                            required: (value) =>
                                                isEmptyConfig ? undefined : validateRequiredValue(value),
                                        },
                                    })}
                                />
                                {errors.v6?.range_end && (
                                    <div className="form__message form__message--error">
                                        {t(errors.v6.range_end.message)}
                                    </div>
                                )}
                            </div>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row">
                <div className="col-lg-6 form__group form__group--settings">
                    <label>{t('dhcp_form_lease_title')}</label>
                    <input
                        type="number"
                        className="form-control"
                        placeholder={t(ipv6placeholders?.lease_duration || '')}
                        disabled={!isInterfaceIncludesIpv6}
                        min={1}
                        max={UINT32_RANGE.MAX}
                        {...register('v6.lease_duration', {
                            valueAsNumber: true,
                            validate: {
                                required: (value) => (isEmptyConfig ? undefined : validateRequiredValue(value)),
                            },
                        })}
                    />
                    {errors.v6?.lease_duration && (
                        <div className="form__message form__message--error">{t(errors.v6.lease_duration.message)}</div>
                    )}
                </div>
            </div>

            <div className="btn-list">
                <button type="submit" className="btn btn-success btn-standard" disabled={isDisabled}>
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default FormDHCPv6;
