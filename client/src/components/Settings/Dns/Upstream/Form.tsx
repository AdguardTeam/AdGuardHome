import classnames from 'classnames';
import React, { useRef } from 'react';
import { Controller, useForm } from 'react-hook-form';
import { Trans, useTranslation } from 'react-i18next';
import { useDispatch, useSelector } from 'react-redux';

import { testUpstreamWithFormValues } from '../../../../actions';
import { DNS_REQUEST_OPTIONS, UPSTREAM_CONFIGURATION_WIKI_LINK } from '../../../../helpers/constants';
import { removeEmptyLines } from '../../../../helpers/helpers';
import { getTextareaCommentsHighlight, syncScroll } from '../../../../helpers/highlightTextareaComments';
import { RootState } from '../../../../initialState';
import '../../../ui/texareaCommentsHighlight.css';
import Examples from './Examples';
import { Checkbox } from '../../../ui/Controls/Checkbox';

const UPSTREAM_DNS_NAME = 'upstream_dns';
const UPSTREAM_MODE_NAME = 'upstream_mode';

type FormData = {
    upstream_dns: string;
    upstream_mode: string;
    fallback_dns: string;
    bootstrap_dns: string;
    local_ptr_upstreams: string;
    use_private_ptr_resolvers: boolean;
    resolve_clients: boolean;
};

type FormProps = {
    initialValues?: Partial<FormData>;
    onSubmit: (data: FormData) => void;
};

const INPUT_FIELDS = [
    {
        name: UPSTREAM_MODE_NAME,
        value: DNS_REQUEST_OPTIONS.LOAD_BALANCING,
        subtitle: 'load_balancing_desc',
        placeholder: 'load_balancing',
    },
    {
        name: UPSTREAM_MODE_NAME,
        value: DNS_REQUEST_OPTIONS.PARALLEL,
        subtitle: 'upstream_parallel',
        placeholder: 'parallel_requests',
    },
    {
        name: UPSTREAM_MODE_NAME,
        value: DNS_REQUEST_OPTIONS.FASTEST_ADDR,
        subtitle: 'fastest_addr_desc',
        placeholder: 'fastest_addr',
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
        mode: 'onChange',
        defaultValues: {
            upstream_dns: initialValues?.upstream_dns || '',
            upstream_mode: initialValues?.upstream_mode || DNS_REQUEST_OPTIONS.LOAD_BALANCING,
            fallback_dns: initialValues?.fallback_dns || '',
            bootstrap_dns: initialValues?.bootstrap_dns || '',
            local_ptr_upstreams: initialValues?.local_ptr_upstreams || '',
            use_private_ptr_resolvers: initialValues?.use_private_ptr_resolvers || false,
            resolve_clients: initialValues?.resolve_clients || false,
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

    const testButtonClass = classnames('btn btn-primary btn-standard mr-2', {
        'btn-loading': processingTestUpstream,
    });

    return (
        <form onSubmit={handleSubmit(onSubmit)} className="form--upstream">
            <div className="row">
                <label className="col form__label" htmlFor={UPSTREAM_DNS_NAME}>
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
                                    <textarea
                                        {...field}
                                        id={UPSTREAM_DNS_NAME}
                                        className="form-control form-control--textarea font-monospace text-input"
                                        placeholder={t('upstream_dns')}
                                        disabled={!!upstream_dns_file || processingSetConfig || processingTestUpstream}
                                        onScroll={(e) => syncScroll(e, textareaRef)}
                                        onBlur={(e) => {
                                            const value = removeEmptyLines(e.target.value);
                                            field.onChange(value);
                                        }}
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

                {INPUT_FIELDS.map(({ name, value, subtitle, placeholder }) => (
                    <div key={value} className="col-12 mb-4">
                        <Controller
                            name="upstream_mode"
                            control={control}
                            render={({ field }) => (
                                <div className="custom-control custom-radio">
                                    <input
                                        {...field}
                                        type="radio"
                                        className="custom-control-input"
                                        id={`${name}_${value}`}
                                        value={value}
                                        checked={field.value === value}
                                        disabled={processingSetConfig || processingTestUpstream}
                                    />
                                    <label className="custom-control-label" htmlFor={`${name}_${value}`}>
                                        <span className="custom-control-label__title">{t(placeholder)}</span>
                                        <span className="custom-control-label__subtitle">{t(subtitle)}</span>
                                    </label>
                                </div>
                            )}
                        />
                    </div>
                ))}

                <div className="col-12">
                    <label className="form__label form__label--with-desc" htmlFor="fallback_dns">
                        {t('fallback_dns_title')}
                    </label>

                    <div className="form__desc form__desc--top">{t('fallback_dns_desc')}</div>

                    <Controller
                        name="fallback_dns"
                        control={control}
                        render={({ field }) => (
                            <textarea
                                {...field}
                                id="fallback_dns"
                                className="form-control form-control--textarea form-control--textarea-small font-monospace"
                                placeholder={t('fallback_dns_placeholder')}
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

                <div className="col-12 mb-2">
                    <label className="form__label form__label--with-desc" htmlFor="bootstrap_dns">
                        {t('bootstrap_dns')}
                    </label>

                    <div className="form__desc form__desc--top">{t('bootstrap_dns_desc')}</div>

                    <Controller
                        name="bootstrap_dns"
                        control={control}
                        render={({ field }) => (
                            <textarea
                                {...field}
                                id="bootstrap_dns"
                                className="form-control form-control--textarea form-control--textarea-small font-monospace"
                                placeholder={t('bootstrap_dns')}
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
                            <textarea
                                {...field}
                                id="local_ptr_upstreams"
                                className="form-control form-control--textarea form-control--textarea-small font-monospace"
                                placeholder={t('local_ptr_placeholder')}
                                disabled={processingSetConfig}
                                onBlur={(e) => {
                                    const value = removeEmptyLines(e.target.value);
                                    field.onChange(value);
                                }}
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
                                title={t('resolve_clients_title')}
                                subtitle={t('resolve_clients_desc')}
                                disabled={processingSetConfig}
                            />
                        )}
                    />
                </div>
            </div>

            <div className="card-actions">
                <div className="btn-list">
                    <button
                        type="button"
                        className={testButtonClass}
                        onClick={handleUpstreamTest}
                        disabled={!upstream_dns || processingTestUpstream}>
                        {t('test_upstream_btn')}
                    </button>

                    <button
                        type="submit"
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
