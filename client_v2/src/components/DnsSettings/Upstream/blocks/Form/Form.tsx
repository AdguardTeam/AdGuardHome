import React from 'react';
import { Controller, useForm } from 'react-hook-form';
import { useDispatch, useSelector } from 'react-redux';

import { testUpstreamWithFormValues } from 'panel/actions';
import { Textarea } from 'panel/common/controls/Textarea';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Button } from 'panel/common/ui/Button';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { toNumber } from 'panel/helpers/form';
import intl from 'panel/common/intl';
import { DNS_REQUEST_OPTIONS, UINT32_RANGE } from 'panel/helpers/constants';
import theme from 'panel/lib/theme';
import { RootState } from 'panel/initialState';

import { Examples } from '../Examples';

import s from './Form.module.pcss';

type FormData = {
    upstream_dns: string;
    upstream_mode: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    use_private_ptr_resolvers: boolean;
    resolve_clients: boolean;
    upstream_timeout: number;
};

type FormProps = {
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

const upstreamModeOptions = [
    {
        text: intl.getMessage('upstream_dns_load_balancing'),
        value: DNS_REQUEST_OPTIONS.LOAD_BALANCING,
        description: intl.getMessage('upstream_dns_load_balancing_desc'),
    },
    {
        text: intl.getMessage('upstream_dns_parallel_requests'),
        value: DNS_REQUEST_OPTIONS.PARALLEL,
        description: intl.getMessage('upstream_dns_parallel_requests_desc'),
    },
    {
        text: intl.getMessage('upstream_dns_fastest_addr'),
        value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
        description: (
            <>
                {intl.getMessage('upstream_dns_fastest_addr_desc')}
                <div className={s.warning}>{intl.getMessage('upstream_dns_fastest_addr_warning')}</div>
            </>
        ),
    },
];

export const Form = ({ initialValues, onSubmit }: FormProps) => {
    const dispatch = useDispatch();

    const {
        control,
        handleSubmit,
        watch,
        formState: { isSubmitting, isDirty },
    } = useForm<FormData>({
        mode: 'onBlur',
        defaultValues: {
            upstream_dns: initialValues?.upstream_dns || '',
            upstream_mode: initialValues?.upstream_mode || DNS_REQUEST_OPTIONS.LOAD_BALANCING,
            fallback_dns: initialValues?.fallback_dns || '',
            bootstrap_dns: initialValues?.bootstrap_dns || '',
            local_ptr_upstreams: initialValues?.local_ptr_upstreams || '',
            use_private_ptr_resolvers: initialValues?.use_private_ptr_resolvers || false,
            resolve_clients: initialValues?.resolve_clients || false,
            upstream_timeout: initialValues?.upstream_timeout || 0,
        },
    });

    const upstream_dns = watch('upstream_dns');
    const processingTestUpstream = useSelector((state: RootState) => state.settings.processingTestUpstream);
    const processingSetConfig = useSelector((state: RootState) => state.dnsConfig.processingSetConfig);
    const defaultLocalPtrUpstreams = useSelector((state: RootState) => state.dnsConfig.default_local_ptr_upstreams);
    const upstreamDnsFile = useSelector((state: RootState) => state.dnsConfig.upstream_dns_file);

    const handleUpstreamTest = () => {
        const formValues = {
            bootstrap_dns: watch('bootstrap_dns'),
            upstream_dns: watch('upstream_dns'),
            local_ptr_upstreams: watch('local_ptr_upstreams'),
            fallback_dns: watch('fallback_dns'),
        };
        dispatch(testUpstreamWithFormValues(formValues));
    };

    const isSavingDisabled = () => {
        return isSubmitting || !isDirty || processingSetConfig || processingTestUpstream;
    };

    const isTestDisabled = () => {
        return !upstream_dns || processingTestUpstream;
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)}>
            <div className={theme.form.group}>
                <div className={theme.form.input}>
                    <Controller
                        name="upstream_dns"
                        control={control}
                        render={({ field }) => (
                            <>
                                <Textarea
                                    {...field}
                                    id="upstream_dns"
                                    label={
                                        <>
                                            {intl.getMessage('upstream_dns_addresses')}
                                            <FaqTooltip
                                                text={intl.getMessage('upstream_dns_addresses_faq', {
                                                    a: (text: string) => (
                                                        <a
                                                            href="https://link.adtidy.org/forward.html?action=dns_kb_providers&from=ui&app=home"
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className={theme.link.link}>
                                                            {text}
                                                        </a>
                                                    ),
                                                    b: (text: string) => (
                                                        <a
                                                            href="https://link.adtidy.org/forward.html?action=dns_kb_providers&from=ui&app=home"
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className={theme.link.link}>
                                                            {text}
                                                        </a>
                                                    ),
                                                })}
                                                menuSize="large"
                                            />
                                        </>
                                    }
                                    placeholder={intl.getMessage('upstream_dns_placeholder')}
                                    disabled={!!upstreamDnsFile || processingSetConfig || processingTestUpstream}
                                    size="medium"
                                />
                            </>
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="upstream_mode"
                        control={control}
                        render={({ field }) => (
                            <Radio
                                {...field}
                                handleChange={field.onChange}
                                options={upstreamModeOptions}
                                disabled={processingSetConfig || processingTestUpstream}
                                verticalAlign="start"
                                textClassName={s.radioText}
                            />
                        )}
                    />
                </div>
            </div>

            <Examples />

            <div className={theme.form.group}>
                <div className={theme.form.input}>
                    <Controller
                        name="fallback_dns"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="fallback_dns"
                                label={
                                    <>
                                        {intl.getMessage('upstream_fallback_title')}
                                        <FaqTooltip
                                            text={intl.getMessage('upstream_fallback_title_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('ip_addresses_placeholder')}
                                disabled={processingSetConfig}
                                size="medium"
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="bootstrap_dns"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="bootstrap_dns"
                                data-testid="bootstrap_dns"
                                label={
                                    <>
                                        {intl.getMessage('upstream_bootstrap_dns_title')}
                                        <FaqTooltip
                                            text={intl.getMessage('upstream_bootstrap_dns_faq')}
                                            menuSize="large"
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('ip_addresses_placeholder')}
                                disabled={processingSetConfig}
                                size="medium"
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="local_ptr_upstreams"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="local_ptr_upstreams"
                                data-testid="local_ptr_upstreams"
                                label={
                                    <>
                                        {intl.getMessage('upstream_ptr')}
                                        <FaqTooltip
                                            text={
                                                <>
                                                    <div>
                                                        {intl.getMessage('upstream_ptr_faq_1', {
                                                            value: '192.168.1.1/24',
                                                        })}
                                                    </div>
                                                    <div>{intl.getMessage('upstream_ptr_faq_2')}</div>
                                                    {defaultLocalPtrUpstreams?.length > 0 && (
                                                        <div>
                                                            {intl.getMessage('upstream_ptr_faq_3', {
                                                                value_1: defaultLocalPtrUpstreams[0],
                                                                value_2: defaultLocalPtrUpstreams[1] || '',
                                                            })}
                                                        </div>
                                                    )}
                                                </>
                                            }
                                            menuSize="large"
                                            spacing
                                        />
                                    </>
                                }
                                placeholder={intl.getMessage('ip_addresses_placeholder')}
                                disabled={processingSetConfig}
                                size="medium"
                            />
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="use_private_ptr_resolvers"
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                id="dns_use_private_ptr_resolvers"
                                name={field.name}
                                checked={field.value}
                                onChange={field.onChange}
                                onBlur={field.onBlur}
                                disabled={processingSetConfig}
                                verticalAlign="start">
                                <div>
                                    <div className={theme.text.t2}>
                                        {intl.getMessage('upstream_private_ptr_resolvers_title')}
                                    </div>
                                    <div className={theme.text.t4}>
                                        {intl.getMessage('upstream_private_ptr_resolvers_desc')}
                                    </div>
                                </div>
                            </Checkbox>
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="resolve_clients"
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                id="dns_resolve_clients"
                                name={field.name}
                                checked={field.value}
                                onChange={field.onChange}
                                onBlur={field.onBlur}
                                disabled={processingSetConfig}
                                verticalAlign="start">
                                <div>
                                    <div className={theme.text.t2}>
                                        {intl.getMessage('upstream_enable_reverse_lookup_title')}
                                    </div>
                                    <div className={theme.text.t4}>
                                        {intl.getMessage('upstream_enable_reverse_lookup_desc')}
                                    </div>
                                </div>
                            </Checkbox>
                        )}
                    />
                </div>

                <div className={theme.form.input}>
                    <Controller
                        name="upstream_timeout"
                        control={control}
                        render={({ field }) => (
                            <Input
                                {...field}
                                type="number"
                                id="upstream_timeout"
                                label={
                                    <>
                                        {intl.getMessage('upstream_timeout')}
                                        <FaqTooltip text={intl.getMessage('upstream_timeout_faq')} menuSize="large" />
                                    </>
                                }
                                placeholder={intl.getMessage('upstream_timeout_placeholder')}
                                disabled={processingSetConfig}
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

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    id="dns_upstream_save"
                    disabled={isSavingDisabled()}
                    className={theme.form.button}>
                    {intl.getMessage('apply')}
                </Button>

                <Button
                    type="button"
                    variant="secondary"
                    size="small"
                    id="dns_upstream_test"
                    onClick={handleUpstreamTest}
                    disabled={isTestDisabled()}
                    className={theme.form.button}>
                    {intl.getMessage('test_upstreams')}
                </Button>
            </div>
        </form>
    );
};
