import React from 'react';

import { Trans, useTranslation } from 'react-i18next';

import { Controller, useForm } from 'react-hook-form';
import i18next from 'i18next';
import {
    validateServerName,
    validateIsSafePort,
    validatePort,
    validatePortQuic,
    validatePortTLS,
    validatePlainDns,
} from '../../../helpers/validators';

import KeyStatus from './KeyStatus';

import CertificateStatus from './CertificateStatus';
import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    STANDARD_HTTPS_PORT,
    ENCRYPTION_SOURCE,
} from '../../../helpers/constants';
import { Checkbox } from '../../ui/Controls/Checkbox';
import { Radio } from '../../ui/Controls/Radio';
import { Input } from '../../ui/Controls/Input';
import { Textarea } from '../../ui/Controls/Textarea';
import { EncryptionData } from '../../../initialState';
import { toNumber } from '../../../helpers/form';

const certificateSourceOptions = [
    {
        label: i18next.t('encryption_certificates_source_path'),
        value: ENCRYPTION_SOURCE.PATH,
    },
    {
        label: i18next.t('encryption_certificates_source_content'),
        value: ENCRYPTION_SOURCE.CONTENT,
    },
];

const keySourceOptions = [
    {
        label: i18next.t('encryption_key_source_path'),
        value: ENCRYPTION_SOURCE.PATH,
    },
    {
        label: i18next.t('encryption_key_source_content'),
        value: ENCRYPTION_SOURCE.CONTENT,
    },
];

const validationMessage = (warningValidation: string, isWarning: boolean) => {
    if (!warningValidation) {
        return null;
    }

    if (isWarning) {
        return (
            <div className="col-12">
                <p>
                    <Trans>encryption_warning</Trans>: {warningValidation}
                </p>
            </div>
        );
    }

    return (
        <div className="col-12">
            <p className="text-danger">{warningValidation}</p>
        </div>
    );
};

export type EncryptionFormValues = {
    enabled?: boolean;
    serve_plain_dns?: boolean;
    server_name?: string;
    force_https?: boolean;
    port_https?: number;
    port_dns_over_tls?: number;
    port_dns_over_quic?: number;
    certificate_chain?: string;
    private_key?: string;
    certificate_path?: string;
    private_key_path?: string;
    certificate_source?: string;
    key_source?: string;
    private_key_saved?: boolean;
};

type Props = {
    initialValues: EncryptionFormValues;
    encryption: EncryptionData;
    onSubmit: (values: EncryptionFormValues) => void;
    debouncedConfigValidation: (values: EncryptionFormValues) => void;
    setTlsConfig: (values: Partial<EncryptionData>) => void;
    validateTlsConfig: (values: Partial<EncryptionData>) => void;
};

const defaultValues = {
    enabled: false,
    serve_plain_dns: true,
    server_name: '',
    force_https: false,
    port_https: STANDARD_HTTPS_PORT,
    port_dns_over_tls: DNS_OVER_TLS_PORT,
    port_dns_over_quic: DNS_OVER_QUIC_PORT,
    certificate_chain: '',
    private_key: '',
    certificate_path: '',
    private_key_path: '',
    certificate_source: ENCRYPTION_SOURCE.PATH,
    key_source: ENCRYPTION_SOURCE.PATH,
    private_key_saved: false,
};

export const Form = ({
    initialValues,
    encryption,
    onSubmit,
    setTlsConfig,
    debouncedConfigValidation,
    validateTlsConfig,
}: Props) => {
    const { t } = useTranslation();

    const {
        not_after,
        valid_chain,
        valid_key,
        valid_cert,
        valid_pair,
        dns_names,
        key_type,
        issuer,
        subject,
        warning_validation,
        processingConfig,
        processingValidate,
    } = encryption;

    const {
        control,
        handleSubmit,
        watch,
        reset,
        setValue,
        setError,
        getValues,
        formState: { isSubmitting, isValid },
    } = useForm<EncryptionFormValues>({
        defaultValues: {
            ...defaultValues,
            ...initialValues,
        },
        mode: 'onBlur',
    });

    const {
        enabled: isEnabled,
        serve_plain_dns: servePlainDns,
        certificate_chain: certificateChain,
        private_key: privateKey,
        private_key_path: privateKeyPath,
        key_source: privateKeySource,
        private_key_saved: privateKeySaved,
        certificate_path: certificatePath,
        certificate_source: certificateSource,
    } = watch();

    const handleBlur = () => {
        debouncedConfigValidation(getValues());
    };

    const isSavingDisabled = () => {
        const processing = isSubmitting || processingConfig || processingValidate;

        if (servePlainDns && !isEnabled) {
            return !isValid || processing;
        }

        return !isValid || processing || !valid_key || !valid_cert || !valid_pair;
    };

    const clearFields = () => {
        if (window.confirm(t('encryption_reset'))) {
            reset();
            setTlsConfig(defaultValues);
            validateTlsConfig(defaultValues);
        }
    };

    const validatePorts = (values: EncryptionFormValues) => {
        const errors: { port_dns_over_tls?: string; port_https?: string } = {};

        if (values.port_dns_over_tls && values.port_https) {
            if (values.port_dns_over_tls === values.port_https) {
                errors.port_dns_over_tls = i18next.t('form_error_equal');
                errors.port_https = i18next.t('form_error_equal');
            }
        }

        return errors;
    };

    const onFormSubmit = (data: EncryptionFormValues) => {
        const validationErrors = validatePorts(data);

        if (Object.keys(validationErrors).length > 0) {
            Object.entries(validationErrors).forEach(([field, message]) => {
                setError(field as keyof EncryptionFormValues, { type: 'manual', message });
            });
        } else {
            onSubmit(data);
        }
    };

    const isDisabled = isSavingDisabled();
    const isWarning = valid_key && valid_cert && valid_pair;

    return (
        <form onSubmit={handleSubmit(onFormSubmit)}>
            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings mb-3">
                        <Controller
                            name="enabled"
                            control={control}
                            render={({ field }) => (
                                <Checkbox {...field} title={t('encryption_enable')} onBlur={handleBlur} />
                            )}
                        />
                    </div>

                    <div className="form__desc">
                        <Trans>encryption_enable_desc</Trans>
                    </div>

                    <div className="form__group mb-3 mt-5">
                        <Controller
                            name="serve_plain_dns"
                            control={control}
                            rules={{
                                validate: (value) => validatePlainDns(value, getValues()),
                            }}
                            render={({ field }) => <Checkbox {...field} title={t('encryption_plain_dns_enable')} />}
                        />
                    </div>

                    <div className="form__desc">
                        <Trans>encryption_plain_dns_desc</Trans>
                    </div>

                    <hr />
                </div>

                <div className="col-12">
                    <label className="form__label" htmlFor="server_name">
                        <Trans>encryption_server</Trans>
                    </label>
                </div>

                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="server_name"
                            control={control}
                            rules={{ validate: validateServerName }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    placeholder={t('encryption_server_enter')}
                                    error={fieldState.error?.message}
                                    disabled={!isEnabled}
                                    onBlur={handleBlur}
                                />
                            )}
                        />

                        <div className="form__desc">
                            <Trans>encryption_server_desc</Trans>
                        </div>
                    </div>
                </div>

                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <Controller
                            name="force_https"
                            control={control}
                            render={({ field }) => (
                                <Checkbox {...field} title={t('encryption_redirect')} disabled={!isEnabled} />
                            )}
                        />

                        <div className="form__desc">
                            <Trans>encryption_redirect_desc</Trans>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row">
                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="port_https">
                            <Trans>encryption_https</Trans>
                        </label>

                        <Controller
                            name="port_https"
                            control={control}
                            rules={{ validate: { validatePort, validateIsSafePort } }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    placeholder={t('encryption_https')}
                                    error={fieldState.error?.message}
                                    disabled={!isEnabled}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                    onBlur={handleBlur}
                                />
                            )}
                        />

                        <div className="form__desc">
                            <Trans>encryption_https_desc</Trans>
                        </div>
                    </div>
                </div>

                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="port_dns_over_tls">
                            <Trans>encryption_dot</Trans>
                        </label>

                        <Controller
                            name="port_dns_over_tls"
                            control={control}
                            rules={{ validate: validatePortTLS }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    placeholder={t('encryption_dot')}
                                    error={fieldState.error?.message}
                                    disabled={!isEnabled}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                    onBlur={handleBlur}
                                />
                            )}
                        />

                        <div className="form__desc">
                            <Trans>encryption_dot_desc</Trans>
                        </div>
                    </div>
                </div>

                <div className="col-lg-6">
                    <div className="form__group form__group--settings">
                        <label className="form__label" htmlFor="port_dns_over_quic">
                            <Trans>encryption_doq</Trans>
                        </label>

                        <Controller
                            name="port_dns_over_quic"
                            control={control}
                            rules={{ validate: validatePortQuic }}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="number"
                                    placeholder={t('encryption_doq')}
                                    error={fieldState.error?.message}
                                    disabled={!isEnabled}
                                    onChange={(e) => {
                                        const { value } = e.target;
                                        field.onChange(toNumber(value));
                                    }}
                                    onBlur={handleBlur}
                                />
                            )}
                        />

                        <div className="form__desc">
                            <Trans>encryption_doq_desc</Trans>
                        </div>
                    </div>
                </div>
            </div>

            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings">
                        <label
                            className="form__label form__label--with-desc form__label--bold"
                            htmlFor="certificate_chain">
                            <Trans>encryption_certificates</Trans>
                        </label>

                        <div className="form__desc form__desc--top">
                            <Trans
                                values={{ link: 'letsencrypt.org' }}
                                components={[
                                    <a
                                        target="_blank"
                                        rel="noopener noreferrer"
                                        href="https://letsencrypt.org/"
                                        key="0">
                                        link
                                    </a>,
                                ]}>
                                encryption_certificates_desc
                            </Trans>
                        </div>

                        <div className="form__inline mb-2">
                            <div className="custom-controls-stacked">
                                <Controller
                                    name="certificate_source"
                                    control={control}
                                    render={({ field }) => (
                                        <Radio {...field} options={certificateSourceOptions} disabled={!isEnabled} />
                                    )}
                                />
                            </div>
                        </div>

                        {certificateSource === ENCRYPTION_SOURCE.CONTENT ? (
                            <Controller
                                name="certificate_chain"
                                control={control}
                                render={({ field, fieldState }) => (
                                    <Textarea
                                        {...field}
                                        placeholder={t('encryption_certificates_input')}
                                        disabled={!isEnabled}
                                        error={fieldState.error?.message}
                                        onBlur={handleBlur}
                                    />
                                )}
                            />
                        ) : (
                            <Controller
                                name="certificate_path"
                                control={control}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        type="text"
                                        placeholder={t('encryption_certificate_path')}
                                        error={fieldState.error?.message}
                                        disabled={!isEnabled}
                                        onBlur={handleBlur}
                                    />
                                )}
                            />
                        )}
                    </div>

                    <div className="form__status">
                        {(certificateChain || certificatePath) && (
                            <CertificateStatus
                                validChain={valid_chain}
                                validCert={valid_cert}
                                subject={subject}
                                issuer={issuer}
                                notAfter={not_after}
                                dnsNames={dns_names}
                            />
                        )}
                    </div>
                </div>
            </div>

            <div className="row">
                <div className="col-12">
                    <div className="form__group form__group--settings mt-3">
                        <label className="form__label form__label--bold" htmlFor="private_key">
                            <Trans>encryption_key</Trans>
                        </label>

                        <div className="form__inline mb-2">
                            <div className="custom-controls-stacked">
                                <Controller
                                    name="key_source"
                                    control={control}
                                    render={({ field }) => (
                                        <Radio {...field} options={keySourceOptions} disabled={!isEnabled} />
                                    )}
                                />
                            </div>
                        </div>

                        {privateKeySource === ENCRYPTION_SOURCE.CONTENT ? (
                            <>
                                <Controller
                                    name="private_key_saved"
                                    control={control}
                                    render={({ field }) => (
                                        <Checkbox
                                            {...field}
                                            title={t('use_saved_key')}
                                            disabled={!isEnabled}
                                            onChange={(checked: boolean) => {
                                                if (checked) {
                                                    setValue('private_key', '');
                                                }
                                                field.onChange(checked);
                                            }}
                                            onBlur={handleBlur}
                                        />
                                    )}
                                />

                                <Controller
                                    name="private_key"
                                    control={control}
                                    render={({ field, fieldState }) => (
                                        <Textarea
                                            {...field}
                                            placeholder={t('encryption_key_input')}
                                            disabled={!isEnabled || privateKeySaved}
                                            error={fieldState.error?.message}
                                            onBlur={handleBlur}
                                        />
                                    )}
                                />
                            </>
                        ) : (
                            <Controller
                                name="private_key_path"
                                control={control}
                                render={({ field, fieldState }) => (
                                    <Input
                                        {...field}
                                        type="text"
                                        placeholder={t('encryption_private_key_path')}
                                        error={fieldState.error?.message}
                                        disabled={!isEnabled}
                                        onBlur={handleBlur}
                                    />
                                )}
                            />
                        )}
                    </div>

                    <div className="form__status">
                        {(privateKey || privateKeyPath) && <KeyStatus validKey={valid_key} keyType={key_type} />}
                    </div>
                </div>
                {validationMessage(warning_validation, isWarning)}
            </div>

            <div className="btn-list mt-2">
                <button type="submit" disabled={isDisabled} className="btn btn-success btn-standart">
                    <Trans>save_config</Trans>
                </button>

                <button
                    type="button"
                    className="btn btn-secondary btn-standart"
                    disabled={isSubmitting || processingConfig}
                    onClick={clearFields}>
                    <Trans>reset_settings</Trans>
                </button>
            </div>
        </form>
    );
};
