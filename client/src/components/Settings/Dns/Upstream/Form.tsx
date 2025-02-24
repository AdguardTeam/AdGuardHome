import React, { useRef } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';

import i18next from 'i18next';
import clsx from 'clsx';
import { testUpstreamWithFormValues } from '../../../../actions';
import { DNS_REQUEST_OPTIONS, UINT32_RANGE, UPSTREAM_CONFIGURATION_WIKI_LINK } from '../../../../helpers/constants';
import { removeEmptyLines } from '../../../../helpers/helpers';
import { getTextareaCommentsHighlight, syncScroll } from '../../../../helpers/highlightTextareaComments';
import { RootState } from '../../../../initialState';
import '../../../ui/texareaCommentsHighlight.css';
import Examples from './Examples';
import { Checkbox } from '../../../ui/Controls/Checkbox';
import { Textarea } from '../../../ui/Controls/Textarea';
import { Radio } from '../../../ui/Controls/Radio';
import { Input } from '../../../ui/Controls/Input';
import { validateRequiredValue } from '../../../../helpers/validators';
import { toNumber } from '../../../../helpers/form';

const UPSTREAM_DNS_NAME = 'upstream_dns';

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
        label: i18next.t('load_balancing'),
        desc: <Trans components={{ br: <br />, b: <b /> }}>load_balancing_desc</Trans>,
        value: DNS_REQUEST_OPTIONS.LOAD_BALANCING,
    },
    {
        label: i18next.t('parallel_requests'),
        desc: <Trans components={{ br: <br />, b: <b /> }}>upstream_parallel</Trans>,
        value: DNS_REQUEST_OPTIONS.PARALLEL,
    },
    {
        label: i18next.t('fastest_addr'),
        desc: <Trans components={{ br: <br />, b: <b /> }}>fastest_addr_desc</Trans>,
        value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
    },
];

const Form = ({ initialValues, onSubmit }: FormProps) => {
    const { t } = useTranslation();
    const dispatch = useDispatch();
    const textareaRef = useRef<HTMLTextAreaElement>(null);

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
    const upstream_dns_file = useSelector((state: RootState) => state.dnsConfig.upstream_dns_file);

    const handleUpstreamTest = () => {
        const formValues = {
            bootstrap_dns: watch('bootstrap_dns'),
            upstream_dns: watch('upstream_dns'),
            local_ptr_upstreams: watch('local_ptr_upstreams'),
            fallback_dns: watch('fallback_dns'),
        };
        dispatch(testUpstreamWithFormValues(formValues));
    };

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="form--upstream">
            <div className="row">
                <label className="col form__label" htmlFor="upstream_dns">
                    <Trans
                        components={{
                            a: <a href={UPSTREAM_CONFIGURATION_WIKI_LINK} target="_blank" rel="noopener noreferrer" />,
                        }}>
                        upstream_dns_help
                    </Trans>{' '}
                    <Trans
                        components={[
                            <a
                                href="https://link.adtidy.org/forward.html?action=dns_kb_providers&from=ui&app=home"
                                target="_blank"
                                rel="noopener noreferrer"
                                key="0">
                                DNS providers
                            </a>,
                        ]}>
                        dns_providers
                    </Trans>
                </label>

                <div className="col-12 mb-4">
                    <div className="text-edit-container">
                        <Controller
                            name="upstream_dns"
                            control={control}
                            render={({ field }) => (
                                <>
                                    <Textarea
                                        {...field}
                                        id={UPSTREAM_DNS_NAME}
                                        data-testid="upstream_dns"
                                        className="form-control--textarea-large text-input"
                                        wrapperClassName="mb-0"
                                        placeholder={t('upstream_dns')}
                                        disabled={!!upstream_dns_file || processingSetConfig || processingTestUpstream}
                                        onScroll={(e) => syncScroll(e, textareaRef)}
                                        trimOnBlur
                                    />
                                    {getTextareaCommentsHighlight(textareaRef, upstream_dns)}
                                </>
                            )}
                        />
                    </div>
                </div>

                <div className="col-12">
                    <Examples />
                    <hr />
                </div>

                <div className="col-12 mb-4">
                    <Controller
                        name="upstream_mode"
                        control={control}
                        render={({ field }) => (
                            <Radio
                                {...field}
                                options={upstreamModeOptions}
                                disabled={processingSetConfig || processingTestUpstream}
                            />
                        )}
                    />
                </div>

                <div className="col-12">
                    <label className="form__label form__label--with-desc" htmlFor="fallback_dns">
                        {t('fallback_dns_title')}
                    </label>

                    <div className="form__desc form__desc--top">{t('fallback_dns_desc')}</div>

                    <Controller
                        name="fallback_dns"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="fallback_dns"
                                data-testid="fallback_dns"
                                wrapperClassName="mb-0"
                                placeholder={t('fallback_dns_placeholder')}
                                disabled={processingSetConfig}
                                trimOnBlur
                            />
                        )}
                    />
                </div>

                <div className="col-12">
                    <hr />
                </div>

                <div className="col-12">
                    <label className="form__label form__label--with-desc" htmlFor="bootstrap_dns">
                        {t('bootstrap_dns')}
                    </label>

                    <div className="form__desc form__desc--top">{t('bootstrap_dns_desc')}</div>

                    <Controller
                        name="bootstrap_dns"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="bootstrap_dns"
                                data-testid="bootstrap_dns"
                                placeholder={t('bootstrap_dns')}
                                wrapperClassName="mb-0"
                                disabled={processingSetConfig}
                                onBlur={(e) => {
                                    const value = removeEmptyLines(e.target.value);
                                    field.onChange(value);
                                }}
                            />
                        )}
                    />
                </div>

                <div className="col-12">
                    <hr />
                </div>

                <div className="col-12">
                    <label className="form__label form__label--with-desc" htmlFor="local_ptr">
                        {t('local_ptr_title')}
                    </label>

                    <div className="form__desc form__desc--top">{t('local_ptr_desc')}</div>

                    <div className="form__desc form__desc--top">
                        {defaultLocalPtrUpstreams?.length > 0
                            ? t('local_ptr_default_resolver', {
                                  ip: defaultLocalPtrUpstreams.map((s: any) => `"${s}"`).join(', '),
                              })
                            : t('local_ptr_no_default_resolver')}
                    </div>

                    <Controller
                        name="local_ptr_upstreams"
                        control={control}
                        render={({ field }) => (
                            <Textarea
                                {...field}
                                id="local_ptr_upstreams"
                                data-testid="local_ptr_upstreams"
                                placeholder={t('local_ptr_placeholder')}
                                disabled={processingSetConfig}
                                trimOnBlur
                            />
                        )}
                    />

                    <div className="mt-4">
                        <Controller
                            name="use_private_ptr_resolvers"
                            control={control}
                            render={({ field }) => (
                                <Checkbox
                                    {...field}
                                    data-testid="dns_use_private_ptr_resolvers"
                                    title={t('use_private_ptr_resolvers_title')}
                                    subtitle={t('use_private_ptr_resolvers_desc')}
                                    disabled={processingSetConfig}
                                />
                            )}
                        />
                    </div>
                </div>

                <div className="col-12">
                    <hr />
                </div>

                <div className="col-12 mb-4">
                    <Controller
                        name="resolve_clients"
                        control={control}
                        render={({ field }) => (
                            <Checkbox
                                {...field}
                                data-testid="dns_resolve_clients"
                                title={t('resolve_clients_title')}
                                subtitle={t('resolve_clients_desc')}
                                disabled={processingSetConfig}
                            />
                        )}
                    />
                </div>

                <div className="col-12">
                    <hr />
                </div>

                <div className="col-12 col-md-7">
                    <div className="form__group">
                        <label htmlFor="upstream_timeout" className="form__label form__label--with-desc">
                            <Trans>upstream_timeout</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            <Trans>upstream_timeout_desc</Trans>
                        </div>

                        <Controller
                            name="upstream_timeout"
                            control={control}
                            rules={{ validate: validateRequiredValue }}
                            render={({ field }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    id="upstream_timeout"
                                    data-testid="upstream_timeout"
                                    placeholder={t('form_enter_upstream_timeout')}
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
            </div>

            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="button"
                        data-testid="dns_upstream_test"
                        className={clsx('btn btn-primary btn-standard mr-2', {
                            'btn-loading': processingTestUpstream,
                        })}
                        onClick={handleUpstreamTest}
                        disabled={!upstream_dns || processingTestUpstream}>
                        {t('test_upstream_btn')}
                    </button>

                    <button
                        type="submit"
                        data-testid="dns_upstream_save"
                        className="btn btn-success btn-standard"
                        disabled={isSubmitting || !isDirty || processingSetConfig || processingTestUpstream}>
                        {t('apply_btn')}
                    </button>
                </div>
            </div>
        </form>
    );
};

export default Form;
