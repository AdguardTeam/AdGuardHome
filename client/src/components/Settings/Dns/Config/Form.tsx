import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import i18next from 'i18next';
import { validateIp, validateIpv4, validateIpv6, validateRequiredValue } from '../../../../helpers/validators';

import { BLOCKING_MODES, UINT32_RANGE } from '../../../../helpers/constants';
import { removeEmptyLines } from '../../../../helpers/helpers';
import { Checkbox } from '../../../ui/Controls/Checkbox';

const checkboxes: {
    name: 'dnssec_enabled' | 'disable_ipv6';
    placeholder: string;
    subtitle: string;
}[] = [
    {
        name: 'dnssec_enabled',
        placeholder: i18next.t('dnssec_enable'),
        subtitle: i18next.t('dnssec_enable_desc'),
    },
    {
        name: 'disable_ipv6',
        placeholder: i18next.t('disable_ipv6'),
        subtitle: i18next.t('disable_ipv6_desc'),
    },
];

const customIps = [
    {
        description: 'blocking_ipv4_desc',
        name: 'blocking_ipv4',
        validateIp: validateIpv4,
    },
    {
        description: 'blocking_ipv6_desc',
        name: 'blocking_ipv6',
        validateIp: validateIpv6,
    },
];

type FormData = {
    ratelimit: number;
    ratelimit_subnet_len_ipv4: number;
    ratelimit_subnet_len_ipv6: number;
    ratelimit_whitelist: string;
    edns_cs_enabled: boolean;
    edns_cs_use_custom: boolean;
    edns_cs_custom_ip?: boolean;
    dnssec_enabled: boolean;
    disable_ipv6: boolean;
    blocking_mode: string;
    blocking_ipv4?: string;
    blocking_ipv6?: string;
    blocked_response_ttl: number;
};

type Props = {
    processing?: boolean;
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

const Form = ({ processing, initialValues, onSubmit }: Props) => {
    const { t } = useTranslation();

    const {
        register,
        handleSubmit,
        watch,
        control,
        formState: { errors, isSubmitting, isDirty },
    } = useForm<FormData>({
        mode: 'onChange',
        defaultValues: initialValues,
    });

    const blocking_mode = watch('blocking_mode');
    const edns_cs_enabled = watch('edns_cs_enabled');
    const edns_cs_use_custom = watch('edns_cs_use_custom');

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className="row">
                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit" className="form__label form__label--with-desc">
                            {t('rate_limit')}
                        </label>

                        <div className="form__desc form__desc--top">{t('rate_limit_desc')}</div>

                        <input
                            id="ratelimit"
                            type="number"
                            className="form-control"
                            disabled={processing}
                            {...register('ratelimit', {
                                required: t('form_error_required'),
                                valueAsNumber: true,
                                min: UINT32_RANGE.MIN,
                                max: UINT32_RANGE.MAX,
                            })}
                        />
                        {errors.ratelimit && (
                            <div className="form__message form__message--error">{errors.ratelimit.message}</div>
                        )}
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit_subnet_len_ipv4" className="form__label form__label--with-desc">
                            {t('rate_limit_subnet_len_ipv4')}
                        </label>

                        <div className="form__desc form__desc--top">{t('rate_limit_subnet_len_ipv4_desc')}</div>

                        <input
                            id="ratelimit_subnet_len_ipv4"
                            type="number"
                            className="form-control"
                            disabled={processing}
                            {...register('ratelimit_subnet_len_ipv4', {
                                required: t('form_error_required'),
                                valueAsNumber: true,
                                min: 0,
                                max: 32,
                            })}
                        />
                        {errors.ratelimit_subnet_len_ipv4 && (
                            <div className="form__message form__message--error">
                                {errors.ratelimit_subnet_len_ipv4.message}
                            </div>
                        )}
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit_subnet_len_ipv6" className="form__label form__label--with-desc">
                            {t('rate_limit_subnet_len_ipv6')}
                        </label>

                        <div className="form__desc form__desc--top">{t('rate_limit_subnet_len_ipv6_desc')}</div>

                        <input
                            id="ratelimit_subnet_len_ipv6"
                            type="number"
                            className="form-control"
                            disabled={processing}
                            {...register('ratelimit_subnet_len_ipv6', {
                                required: t('form_error_required'),
                                valueAsNumber: true,
                                min: 0,
                                max: 128,
                            })}
                        />
                        {errors.ratelimit_subnet_len_ipv6 && (
                            <div className="form__message form__message--error">
                                {errors.ratelimit_subnet_len_ipv6.message}
                            </div>
                        )}
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="ratelimit_whitelist" className="form__label form__label--with-desc">
                            {t('rate_limit_whitelist')}
                        </label>

                        <div className="form__desc form__desc--top">{t('rate_limit_whitelist_desc')}</div>

                        <textarea
                            id="ratelimit_whitelist"
                            className="form-control"
                            disabled={processing}
                            {...register('ratelimit_whitelist', {
                                onChange: removeEmptyLines,
                            })}
                        />
                        {errors.ratelimit_whitelist && (
                            <div className="form__message form__message--error">
                                {errors.ratelimit_whitelist.message}
                            </div>
                        )}
                    </div>
                </div>

                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="edns_cs_enabled"
                            control={control}
                            render={({ field: { name, value, onChange } }) => (
                                <Checkbox
                                    name={name}
                                    title={t('edns_enable')}
                                    value={value}
                                    onChange={(value) => onChange(value)}
                                    disabled={processing}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-12 form__group form__group--inner">
                    <div className="form__group">
                        <Controller
                            name="edns_cs_use_custom"
                            control={control}
                            render={({ field: { name, value, onChange } }) => (
                                <Checkbox
                                    name={name}
                                    title={t('edns_use_custom_ip')}
                                    value={value}
                                    onChange={(value) => onChange(value)}
                                    disabled={processing || !edns_cs_enabled}
                                />
                            )}
                        />
                    </div>

                    {edns_cs_use_custom && (
                        <input
                            id="edns_cs_custom_ip"
                            type="text"
                            className="form-control"
                            disabled={processing || !edns_cs_enabled}
                            {...register('edns_cs_custom_ip', {
                                required: t('form_error_required'),
                                validate: (value) => validateIp(value) || validateRequiredValue(value),
                            })}
                        />
                    )}
                </div>

                {checkboxes.map(({ name, placeholder, subtitle }) => (
                    <div className="col-12" key={name}>
                        <div className="form__group form__group--settings">
                            <Controller
                                name={name}
                                control={control}
                                render={({ field: { name, value, onChange } }) => (
                                    <Checkbox
                                        name={name}
                                        title={placeholder}
                                        subtitle={subtitle}
                                        value={value}
                                        onChange={(value) => onChange(value)}
                                        disabled={processing}
                                    />
                                )}
                            />
                        </div>
                    </div>
                ))}

                <div className="col-12">
                    <div className="form__group form__group--settings mb-4">
                        <label className="form__label form__label--with-desc">{t('blocking_mode')}</label>

                        <div className="form__desc form__desc--top">
                            {Object.values(BLOCKING_MODES).map((mode: any) => (
                                <li key={mode}>{t(`blocking_mode_${mode}`)}</li>
                            ))}
                        </div>

                        <div className="custom-controls-stacked">
                            {Object.values(BLOCKING_MODES).map((mode: any) => (
                                <label key={mode} className="custom-control custom-radio">
                                    <input
                                        type="radio"
                                        className="custom-control-input"
                                        value={mode}
                                        disabled={processing}
                                        {...register('blocking_mode')}
                                    />
                                    <span className="custom-control-label">{t(mode)}</span>
                                </label>
                            ))}
                        </div>
                    </div>
                </div>
                {blocking_mode === BLOCKING_MODES.custom_ip && (
                    <>
                        {customIps.map(({ description, name, validateIp }) => (
                            <div className="col-12 col-sm-6" key={name}>
                                <div className="form__group form__group--settings">
                                    <label className="form__label form__label--with-desc" htmlFor={name}>
                                        {t(name)}
                                    </label>

                                    <div className="form__desc form__desc--top">{t(description)}</div>

                                    <input
                                        id={name}
                                        type="text"
                                        className="form-control"
                                        disabled={processing}
                                        {...register(name as keyof FormData, {
                                            required: t('form_error_required'),
                                            validate: (value) => validateIp(value) || validateRequiredValue(value),
                                        })}
                                    />
                                    {errors[name as keyof FormData] && (
                                        <div className="form__message form__message--error">
                                            {errors[name as keyof FormData]?.message}
                                        </div>
                                    )}
                                </div>
                            </div>
                        ))}
                    </>
                )}

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <label htmlFor="blocked_response_ttl" className="form__label form__label--with-desc">
                            {t('blocked_response_ttl')}
                        </label>

                        <div className="form__desc form__desc--top">{t('blocked_response_ttl_desc')}</div>

                        <input
                            id="blocked_response_ttl"
                            type="number"
                            className="form-control"
                            disabled={processing}
                            {...register('blocked_response_ttl', {
                                required: t('form_error_required'),
                                valueAsNumber: true,
                                min: UINT32_RANGE.MIN,
                                max: UINT32_RANGE.MAX,
                            })}
                        />
                        {errors.blocked_response_ttl && (
                            <div className="form__message form__message--error">
                                {errors.blocked_response_ttl.message}
                            </div>
                        )}
                    </div>
                </div>
            </div>

            <button
                type="submit"
                className="btn btn-success btn-standard btn-large"
                disabled={isSubmitting || !isDirty || processing}>
                {t('save_btn')}
            </button>
        </form>
    );
};

export default Form;
