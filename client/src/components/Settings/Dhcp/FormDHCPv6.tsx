import React from 'react';
import { useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';
import { useSelector } from 'react-redux';

import { UINT32_RANGE } from '../../../helpers/constants';
import { validateIpv6, validateRequiredValue } from '../../../helpers/validators';
import { RootState } from '../../../initialState';

type FormValues = {
    v6?: {
        range_start?: string;
        range_end?: string;
        lease_duration?: number;
    };
}

type FormDHCPv6Props = {
    processingConfig?: boolean;
    initialValues?: FormValues;
    ipv6placeholders?: {
        range_start: string;
        range_end: string;
        lease_duration: string;
    };
    onSubmit?: (data: FormValues) => Promise<void> | void;
}

const FormDHCPv6 = ({
    processingConfig,
    initialValues,
    ipv6placeholders,
    onSubmit,
}: FormDHCPv6Props) => {
    const { t } = useTranslation();

    const interfaces = useSelector((state: RootState) => state.form.DHCP_INTERFACES);
    const interface_name = interfaces?.values?.interface_name;

    const isInterfaceIncludesIpv6 = useSelector(
        (state: RootState) => !!state.dhcp?.interfaces?.[interface_name]?.ipv6_addresses,
    );

    const {
        register,
        handleSubmit,
        formState: { errors, isSubmitting },
        watch,
    } = useForm<FormValues>({
        mode: 'onChange',
        defaultValues: {
            v6: initialValues?.v6 || {
                range_start: '',
                range_end: '',
                lease_duration: 0,
            },
        },
    });

    const formValues = watch('v6');
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
                                            required: (value) => isEmptyConfig ? undefined : validateRequiredValue(value),
                                        }
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
                                            required: (value) => isEmptyConfig ? undefined : validateRequiredValue(value),
                                        }
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
                                required: (value) => isEmptyConfig ? undefined : validateRequiredValue(value),
                            }
                        })}
                    />
                    {errors.v6?.lease_duration && (
                        <div className="form__message form__message--error">
                            {t(errors.v6.lease_duration.message)}
                        </div>
                    )}
                </div>
            </div>

            <div className="btn-list">
                <button
                    type="submit"
                    className="btn btn-success btn-standard"
                    disabled={isSubmitting || processingConfig || !isInterfaceIncludesIpv6 || isEmptyConfig || Object.keys(errors).length > 0}
                >
                    {t('save_config')}
                </button>
            </div>
        </form>
    );
};

export default FormDHCPv6;
