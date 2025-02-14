import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useTranslation } from 'react-i18next';

import i18next from 'i18next';
import { validateIp, validateIpv4, validateIpv6, validateRequiredValue } from '../../../../helpers/validators';

import { BLOCKING_MODES, UINT32_RANGE } from '../../../../helpers/constants';
import { Checkbox } from '../../../ui/Controls/Checkbox';
import { Input } from '../../../ui/Controls/Input';
import { toNumber } from '../../../../helpers/form';
import { Textarea } from '../../../ui/Controls/Textarea';
import { Radio } from '../../../ui/Controls/Radio';

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

const customIps: {
    name: 'blocking_ipv4' | 'blocking_ipv6';
    label: string;
    description: string;
    validateIp: (value: string) => string;
}[] = [
    {
        name: 'blocking_ipv4',
        label: i18next.t('blocking_ipv4'),
        description: i18next.t('blocking_ipv4_desc'),
        validateIp: validateIpv4,
    },
    {
        name: 'blocking_ipv6',
        label: i18next.t('blocking_ipv6'),
        description: i18next.t('blocking_ipv6_desc'),
        validateIp: validateIpv6,
    },
];

const blockingModeOptions = [
    {
        value: BLOCKING_MODES.default,
        label: i18next.t('default'),
    },
    {
        value: BLOCKING_MODES.refused,
        label: i18next.t('refused'),
    },
    {
        value: BLOCKING_MODES.nxdomain,
        label: i18next.t('nxdomain'),
    },
    {
        value: BLOCKING_MODES.null_ip,
        label: i18next.t('null_ip'),
    },
    {
        value: BLOCKING_MODES.custom_ip,
        label: i18next.t('custom_ip'),
    },
];

const blockingModeDescriptions = [
    i18next.t(`blocking_mode_default`),
    i18next.t(`blocking_mode_refused`),
    i18next.t(`blocking_mode_nxdomain`),
    i18next.t(`blocking_mode_null_ip`),
    i18next.t(`blocking_mode_custom_ip`),
];

type FormData = {
    ratelimit: number;
    ratelimit_subnet_len_ipv4: number;
    ratelimit_subnet_len_ipv6: number;
    ratelimit_whitelist: string;
    edns_cs_enabled: boolean;
    edns_cs_use_custom: boolean;
    edns_cs_custom_ip?: string;
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
        handleSubmit,
        watch,
        control,
        formState: { isSubmitting, isDirty },
    } = useForm<FormData>({
        mode: 'onBlur',
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
                        <Controller
                            name="ratelimit"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    data-testid="dns_config_ratelimit"
                                    type="number"
                                    label={t('rate_limit')}
                                    desc={t('rate_limit_desc')}
                                    error={fieldState.error?.message}
                                    min={UINT32_RANGE.MIN}
                                    max={UINT32_RANGE.MAX}
                                    disabled={processing}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="ratelimit_subnet_len_ipv4"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    data-testid="dns_config_subnet_ipv4"
                                    type="number"
                                    label={t('rate_limit_subnet_len_ipv4')}
                                    desc={t('rate_limit_subnet_len_ipv4_desc')}
                                    error={fieldState.error?.message}
                                    min={0}
                                    max={32}
                                    disabled={processing}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="ratelimit_subnet_len_ipv6"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    data-testid="dns_config_subnet_ipv6"
                                    type="number"
                                    label={t('rate_limit_subnet_len_ipv6')}
                                    desc={t('rate_limit_subnet_len_ipv6_desc')}
                                    error={fieldState.error?.message}
                                    min={0}
                                    max={128}
                                    disabled={processing}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="ratelimit_whitelist"
                            control={control}
                            render={({ field, fieldState }) => (
                                <Textarea
                                    {...field}
                                    data-testid="dns_config_subnet_ipv6"
                                    label={t('rate_limit_whitelist')}
                                    desc={t('rate_limit_whitelist_desc')}
                                    error={fieldState.error?.message}
                                    disabled={processing}
                                    trimOnBlur
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="edns_cs_enabled"
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    {...field}
                                    data-testid="dns_config_edns_cs_enabled"
                                    title={t('edns_enable')}
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
                            render={({ field }) => (
                                <Checkbox
                                    {...field}
                                    data-testid="dns_config_edns_use_custom_ip"
                                    title={t('edns_use_custom_ip')}
                                    disabled={processing || !edns_cs_enabled}
                                />
                            )}
                        />
                    </div>

                    {edns_cs_use_custom && (
                        <Controller
                            name="edns_cs_custom_ip"
                            control={control}
                            rules={{
                                validate: {
                                    required: validateRequiredValue,
                                    id: validateIp,
                                },
                            }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    data-testid="dns_config_edns_cs_custom_ip"
                                    error={fieldState.error?.message}
                                    disabled={processing || !edns_cs_enabled}
                                />
                            )}
                        />
                    )}
                </div>

                {checkboxes.map(({ name, placeholder, subtitle }) => (
                    <div className="col-12" key={name}>
                        <div className="form__group form__group--settings">
                            <Controller
                                name={name}
                                control={control}
                                render={({ field }) => (
                                    <Checkbox
                                        {...field}
                                        data-testid={`dns_config_${name}`}
                                        title={placeholder}
                                        subtitle={subtitle}
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
                            {blockingModeDescriptions.map((desc: string) => (
                                <li key={desc}>{desc}</li>
                            ))}
                        </div>

                        <div className="custom-controls-stacked">
                            <Controller
                                name="blocking_mode"
                                control={control}
                                render={({ field }) => (
                                    <Radio {...field} options={blockingModeOptions} disabled={processing} />
                                )}
                            />
                        </div>
                    </div>
                </div>
                {blocking_mode === BLOCKING_MODES.custom_ip && (
                    <>
                        {customIps.map(({ label, description, name, validateIp }) => (
                            <div className="col-12 col-sm-6" key={name}>
                                <div className="form__group form__group--settings">
                                    <Controller
                                        name={name}
                                        control={control}
                                        rules={{
                                            validate: {
                                                required: validateRequiredValue,
                                                ip: validateIp,
                                            },
                                        }}
                                        render={({ field, fieldState }) => (
                                            <Input
                                                {...field}
                                                data-testid="dns_config_blocked_response_ttl"
                                                type="text"
                                                label={label}
                                                desc={description}
                                                error={fieldState.error?.message}
                                                disabled={processing}
                                            />
                                        )}
                                    />
                                </div>
                            </div>
                        ))}
                    </>
                )}

                <div className="col-12 col-md-7">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="blocked_response_ttl"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    data-testid="dns_config_blocked_response_ttl"
                                    type="number"
                                    label={t('blocked_response_ttl')}
                                    desc={t('blocked_response_ttl_desc')}
                                    error={fieldState.error?.message}
                                    min={UINT32_RANGE.MIN}
                                    max={UINT32_RANGE.MAX}
                                    disabled={processing}
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

            <button
                type="submit"
                data-testid="dns_config_save"
                className="btn btn-success btn-standard btn-large"
                disabled={isSubmitting || !isDirty || processing}>
                {t('save_btn')}
            </button>
        </form>
    );
};

export default Form;
