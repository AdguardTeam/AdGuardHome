import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import cn from 'clsx';

import intl from 'panel/common/intl';
import { validateIp, validateIpv4, validateIpv6, validateRequiredValue } from 'panel/helpers/validators';
import { BLOCKING_MODES, UINT32_RANGE } from 'panel/helpers/constants';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Input } from 'panel/common/controls/Input';
import { toNumber } from 'panel/helpers/form';
import { Textarea } from 'panel/common/controls/Textarea';
import { Radio } from 'panel/common/controls/Radio';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import theme from 'panel/lib/theme';

import s from './Form.module.pcss';

const blockingModesDescriptions = [
    intl.getMessage('server_config_blocking_mode_default_desc'),
    intl.getMessage('server_config_blocking_mode_refused_desc'),
    intl.getMessage('server_config_blocking_mode_nxdomain_desc'),
    intl.getMessage('server_config_blocking_mode_null_ip_desc'),
    intl.getMessage('server_config_blocking_mode_custom_ip_desc'),
];

const checkboxes: {
    name: 'dnssec_enabled' | 'disable_ipv6';
    placeholder: string;
    subtitle: string;
}[] = [
    {
        name: 'dnssec_enabled',
        placeholder: intl.getMessage('server_config_dnssec_enable'),
        subtitle: intl.getMessage('server_config_dnssec_enable_desc'),
    },
    {
        name: 'disable_ipv6',
        placeholder: intl.getMessage('server_config_disable_ipv6'),
        subtitle: intl.getMessage('server_config_disable_ipv6_desc'),
    },
];

const customIps: {
    name: 'blocking_ipv4' | 'blocking_ipv6';
    label: string;
    placeholder: string;
    faq: string;
    validateIp: (value: string) => string;
}[] = [
    {
        name: 'blocking_ipv4',
        label: intl.getMessage('server_config_blocking_mode_ipv4'),
        placeholder: intl.getMessage('server_config_blocking_mode_ipv4_placeholder'),
        faq: intl.getMessage('server_config_blocking_mode_ipv4_faq'),
        validateIp: validateIpv4,
    },
    {
        name: 'blocking_ipv6',
        label: intl.getMessage('server_config_blocking_mode_ipv6'),
        placeholder: intl.getMessage('server_config_blocking_mode_ipv6_placeholder'),
        faq: intl.getMessage('server_config_blocking_mode_ipv6_faq'),
        validateIp: validateIpv6,
    },
];

const blockingModeOptions = [
    {
        value: BLOCKING_MODES.default,
        text: intl.getMessage('server_config_default'),
    },
    {
        value: BLOCKING_MODES.refused,
        text: intl.getMessage('server_config_refused'),
    },
    {
        value: BLOCKING_MODES.nxdomain,
        text: intl.getMessage('server_config_nxdomain'),
    },
    {
        value: BLOCKING_MODES.null_ip,
        text: intl.getMessage('server_config_null_ip'),
    },
    {
        value: BLOCKING_MODES.custom_ip,
        text: intl.getMessage('server_config_custom_ip'),
    },
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

export const Form = ({ processing, initialValues, onSubmit }: Props) => {
    const {
        handleSubmit,
        watch,
        control,
        formState: { isSubmitting },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: initialValues,
    });

    const blocking_mode = watch('blocking_mode');
    const edns_cs_enabled = watch('edns_cs_enabled');
    const edns_cs_use_custom = watch('edns_cs_use_custom');

    return (
        <form onSubmit={handleSubmit(onSubmit)} className={theme.form.form}>
            <div className={theme.form.group}>
                <div className={theme.form.input}>
                    <Controller
                        name="ratelimit"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                data-testid="dns_config_ratelimit"
                                type="number"
                                label={
                                    <>
                                        {intl.getMessage('server_config_rate_limit')}
                                        <FaqTooltip
                                            text={intl.getMessage('server_config_rate_limit_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('server_config_rate_limit_placeholder')}
                                errorMessage={fieldState.error?.message}
                                min={UINT32_RANGE.MIN}
                                max={UINT32_RANGE.MAX}
                                disabled={!!processing}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="ratelimit_subnet_len_ipv4"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                data-testid="dns_config_subnet_ipv4"
                                type="number"
                                label={
                                    <>
                                        {intl.getMessage('server_config_subnet_len_ipv4')}
                                        <FaqTooltip
                                            text={intl.getMessage('server_config_subnet_len_ipv4_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('server_config_subnet_len_placeholder')}
                                errorMessage={fieldState.error?.message}
                                min={0}
                                max={32}
                                disabled={!!processing}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="ratelimit_subnet_len_ipv6"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                data-testid="dns_config_subnet_ipv6"
                                type="number"
                                label={
                                    <>
                                        {intl.getMessage('server_config_subnet_len_ipv6')}
                                        <FaqTooltip
                                            text={intl.getMessage('server_config_subnet_len_ipv6_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('server_config_subnet_len_placeholder')}
                                errorMessage={fieldState.error?.message}
                                min={0}
                                max={128}
                                disabled={!!processing}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="ratelimit_whitelist"
                        control={control}
                        render={({ field, fieldState }) => (
                            <Textarea
                                {...field}
                                data-testid="dns_config_ratelimit_whitelist"
                                label={
                                    <>
                                        {intl.getMessage('server_config_rate_limit_whitelist')}
                                        <FaqTooltip
                                            text={intl.getMessage('server_config_rate_limit_whitelist_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('ip_addresses_placeholder')}
                                errorMessage={fieldState.error?.message}
                                disabled={!!processing}
                                size="medium"
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="edns_cs_enabled"
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                name={field.name}
                                checked={field.value}
                                onChange={field.onChange}
                                onBlur={field.onBlur}
                                data-testid="dns_config_edns_cs_enabled"
                                disabled={!!processing}
                                verticalAlign="start">
                                <div>
                                    <div className={theme.text.t2}>{intl.getMessage('server_config_edns_enable')}</div>
                                    <div className={theme.text.t4}>{intl.getMessage('server_config_edns_cs_desc')}</div>
                                </div>
                            </Checkbox>
                        )}
                    />
                </div>

                <div className={theme.form.inner}>
                    <div className={theme.form.input}>
                        <Controller
                            name="edns_cs_use_custom"
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    name={field.name}
                                    checked={field.value}
                                    onChange={field.onChange}
                                    onBlur={field.onBlur}
                                    data-testid="dns_config_edns_use_custom_ip"
                                    disabled={processing || !edns_cs_enabled}
                                    verticalAlign="start">
                                    <div>
                                        <div className={theme.text.t2}>
                                            {intl.getMessage('server_config_edns_use_custom_ip')}
                                        </div>
                                        <div className={theme.text.t4}>
                                            {intl.getMessage('server_config_edns_use_custom_ip_desc')}
                                        </div>
                                    </div>
                                </Checkbox>
                            )}
                        />
                    </div>

                    {edns_cs_use_custom && (
                        <div className={theme.form.input}>
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
                                        placeholder={intl.getMessage('enter_ip_address_placeholder')}
                                        errorMessage={fieldState.error?.message}
                                        disabled={processing || !edns_cs_enabled}
                                    />
                                )}
                            />
                        </div>
                    )}
                </div>

                {checkboxes.map(({ name, placeholder, subtitle }) => (
                    <div key={name} className={theme.form.input}>
                        <Controller
                            name={name}
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    name={field.name}
                                    checked={field.value}
                                    onChange={field.onChange}
                                    onBlur={field.onBlur}
                                    id={`dns_config_${name}`}
                                    disabled={!!processing}
                                    verticalAlign="start">
                                    <div>
                                        <div className={theme.text.t2}>{placeholder}</div>
                                        <div className={theme.text.t4}>{subtitle}</div>
                                    </div>
                                </Checkbox>
                            )}
                        />
                    </div>
                ))}

                <div className={theme.form.input}>
                    <div className={cn(s.subtitle, theme.title.h6)}>
                        {intl.getMessage('server_config_blocking_mode')}
                    </div>
                    <div className={s.descriptions}>
                        {blockingModesDescriptions.map((description, index) => (
                            <div className={theme.text.t2} key={index}>
                                {description}
                            </div>
                        ))}
                    </div>

                    <Controller
                        name="blocking_mode"
                        control={control}
                        render={({ field }) => (
                            <Radio
                                value={field.value}
                                handleChange={field.onChange}
                                name={field.name}
                                options={blockingModeOptions}
                                disabled={!!processing}
                            />
                        )}
                    />
                </div>

                {blocking_mode === BLOCKING_MODES.custom_ip && (
                    <>
                        {customIps.map(({ label, name, placeholder, faq, validateIp }) => (
                            <div key={name} className={theme.form.input}>
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
                                            data-testid={`dns_config_${name}`}
                                            type="text"
                                            label={
                                                <>
                                                    {label}
                                                    <FaqTooltip text={faq} menuSize="large" />
                                                </>
                                            }
                                            placeholder={placeholder}
                                            errorMessage={fieldState.error?.message}
                                            disabled={!!processing}
                                        />
                                    )}
                                />
                            </div>
                        ))}
                    </>
                )}

                <div className={theme.form.input}>
                    <Controller
                        name="blocked_response_ttl"
                        control={control}
                        rules={{ validate: validateRequiredValue }}
                        render={({ field, fieldState }) => (
                            <Input
                                {...field}
                                data-testid="dns_config_blocked_response_ttl"
                                type="number"
                                label={
                                    <>
                                        {intl.getMessage('server_config_blocking_mode_ttl')}
                                        <FaqTooltip
                                            text={intl.getMessage('server_config_blocking_mode_ttl_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('form_enter_blocked_response_ttl')}
                                errorMessage={fieldState.error?.message}
                                min={UINT32_RANGE.MIN}
                                max={UINT32_RANGE.MAX}
                                disabled={!!processing}
                                onChange={(e) => {
                                    const { value } = e.target;
                                    field.onChange(toNumber(value));
                                }}
                            />
                        )}
                    />
                </div>
            </div>

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    id="dns_config_save"
                    variant="primary"
                    disabled={isSubmitting || processing}
                    className={theme.form.button}
                    size="small">
                    {intl.getMessage('save')}
                </Button>
            </div>
        </form>
    );
};
