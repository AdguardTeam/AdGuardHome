import React, { useState } from 'react';
import { Controller, useForm } from 'react-hook-form';
import i18next from 'i18next';
import cn from 'clsx';

import { toNumber } from 'panel/helpers/form';
import { DNS_OVER_QUIC_PORT, DNS_OVER_TLS_PORT, STANDARD_HTTPS_PORT, ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { EncryptionData } from 'panel/initialState';
import {
    validateServerName,
    validateIsSafePort,
    validatePort,
    validatePortQuic,
    validatePortTLS,
    validatePlainDns,
} from 'panel/helpers/validators';
import { Checkbox } from 'panel/common/controls/Checkbox';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { Textarea } from 'panel/common/controls/Textarea';
import { useDispatch } from 'react-redux';
import { setTlsConfig, validateTlsConfig } from 'panel/actions/encryption';
import { Button } from 'panel/common/ui/Button';
import intl from 'panel/common/intl';
import { SwitchGroup } from 'panel/common/ui/SettingsGroup';
import theme from 'panel/lib/theme';

import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import { ConfirmDialog } from 'panel/common/ui/ConfirmDialog';
import { KeyStatus, CertificateStatus, ValidationStatus } from './Status';

import s from './styles.module.pcss';

const certificateSourceOptions = [
    {
        text: i18next.t('encryption_certificates_source_path'),
        value: ENCRYPTION_SOURCE.PATH,
    },
    {
        text: i18next.t('encryption_certificates_source_content'),
        value: ENCRYPTION_SOURCE.CONTENT,
    },
];

const keySourceOptions = [
    {
        text: i18next.t('encryption_key_source_path'),
        value: ENCRYPTION_SOURCE.PATH,
    },
    {
        text: i18next.t('encryption_key_source_content'),
        value: ENCRYPTION_SOURCE.CONTENT,
    },
];

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

export const Form = ({ initialValues, encryption, onSubmit, debouncedConfigValidation }: Props) => {
    const dispatch = useDispatch();
    const [openConfirmReset, setOpenConfirmReset] = useState(false);
    const [openPlainDnsDisable, setOpenPlainDnsDisable] = useState(false);
    const [stagedFormValues, setStagedFormValues] = useState<EncryptionFormValues | null>(null);

    const {
        not_after,
        valid_chain,
        valid_key,
        valid_cert,
        valid_pair,
        key_type,
        dns_names,
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

    const handleResetOpen = () => setOpenConfirmReset(true);

    const handleResetClose = () => setOpenConfirmReset(false);

    const handlePlainDnsDisableOpen = () => setOpenPlainDnsDisable(true);

    const handlePlainDnsDisableClose = () => {
        setOpenPlainDnsDisable(false);
        setStagedFormValues(null);
    };

    const handlePlainDnsDisableConfirm = () => {
        if (stagedFormValues) {
            onSubmit(stagedFormValues);
            setStagedFormValues(null);
        }
        setOpenPlainDnsDisable(false);
    };

    const handleReset = () => {
        reset();
        dispatch(setTlsConfig(defaultValues));
        dispatch(validateTlsConfig(defaultValues));
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
            return;
        }

        if (data.serve_plain_dns === false) {
            setStagedFormValues(data);
            handlePlainDnsDisableOpen();
            return;
        }

        onSubmit(data);
    };

    const renderCertificateStatus = () => {
        if (warning_validation) {
            const isWarning = valid_key && valid_cert && valid_pair;

            return <ValidationStatus type={isWarning ? 'warning' : 'error'} message={warning_validation} />;
        }

        if (!certificateChain && !certificatePath) {
            return null;
        }

        return (
            <CertificateStatus
                validChain={valid_chain}
                validCert={valid_cert}
                subject={subject}
                issuer={issuer}
                notAfter={not_after}
                dnsNames={dns_names}
            />
        );
    };

    const isDisabled = isSavingDisabled();

    return (
        <form onSubmit={handleSubmit(onFormSubmit)}>
            <Controller
                name="enabled"
                control={control}
                render={({ field }) => (
                    <SwitchGroup
                        id="enabled"
                        title={intl.getMessage('encryption_encrypted_dns')}
                        description={intl.getMessage('encryption_encrypted_dns_desc')}
                        checked={field.value}
                        onChange={field.onChange}>
                        <div className={s.group}>
                            <div>
                                <Controller
                                    name="server_name"
                                    control={control}
                                    rules={{ validate: validateServerName }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="text"
                                            label={
                                                <>
                                                    {intl.getMessage('encryption_server')}

                                                    <FaqTooltip
                                                        text={
                                                            <>
                                                                <div className={s.tooltipText}>
                                                                    {intl.getMessage('encryption_server_tooltip_1')}
                                                                </div>
                                                                <div className={s.tooltipText}>
                                                                    {intl.getMessage('encryption_server_tooltip_2')}
                                                                </div>
                                                            </>
                                                        }
                                                        menuSize="large"
                                                    />
                                                </>
                                            }
                                            placeholder={intl.getMessage('encryption_server_enter')}
                                            errorMessage={fieldState.error?.message}
                                            disabled={!isEnabled}
                                            onBlur={handleBlur}
                                        />
                                    )}
                                />
                            </div>
                            <div>
                                <Controller
                                    name="port_https"
                                    control={control}
                                    rules={{ validate: { validatePort, validateIsSafePort } }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="number"
                                            label={
                                                <>
                                                    {intl.getMessage('encryption_https')}

                                                    <FaqTooltip
                                                        text={intl.getMessage('encryption_https_tooltip')}
                                                        menuSize="large"
                                                    />
                                                </>
                                            }
                                            placeholder={intl.getMessage('encryption_https')}
                                            errorMessage={fieldState.error?.message}
                                            disabled={!isEnabled}
                                            onChange={(e) => {
                                                const { value } = e.target;
                                                field.onChange(toNumber(value));
                                            }}
                                            onBlur={handleBlur}
                                        />
                                    )}
                                />
                            </div>
                            <div>
                                <Controller
                                    name="port_dns_over_tls"
                                    control={control}
                                    rules={{ validate: validatePortTLS }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="number"
                                            label={
                                                <>
                                                    {intl.getMessage('encryption_dot')}

                                                    <FaqTooltip
                                                        text={intl.getMessage('encryption_dot_tooltip')}
                                                        menuSize="large"
                                                    />
                                                </>
                                            }
                                            placeholder={intl.getMessage('encryption_dot')}
                                            errorMessage={fieldState.error?.message}
                                            disabled={!isEnabled}
                                            onChange={(e) => {
                                                const { value } = e.target;
                                                field.onChange(toNumber(value));
                                            }}
                                            onBlur={handleBlur}
                                        />
                                    )}
                                />
                            </div>
                            <div>
                                <Controller
                                    name="port_dns_over_quic"
                                    control={control}
                                    rules={{ validate: validatePortQuic }}
                                    render={({ field, fieldState }) => (
                                        <Input
                                            {...field}
                                            type="number"
                                            label={
                                                <>
                                                    {intl.getMessage('encryption_doq')}

                                                    <FaqTooltip
                                                        text={intl.getMessage('encryption_doq_tooltip')}
                                                        menuSize="large"
                                                    />
                                                </>
                                            }
                                            placeholder={intl.getMessage('encryption_doq')}
                                            errorMessage={fieldState.error?.message}
                                            disabled={!isEnabled}
                                            onChange={(e) => {
                                                const { value } = e.target;
                                                field.onChange(toNumber(value));
                                            }}
                                            onBlur={handleBlur}
                                        />
                                    )}
                                />
                            </div>
                        </div>
                    </SwitchGroup>
                )}
            />

            <Controller
                name="serve_plain_dns"
                control={control}
                rules={{
                    validate: (value) => validatePlainDns(value, getValues()),
                }}
                render={({ field }) => (
                    <SwitchGroup
                        id="serve_plain_dns"
                        title={intl.getMessage('encryption_plain_dns')}
                        description={intl.getMessage('encryption_plain_dns_desc')}
                        checked={field.value}
                        onChange={field.onChange}
                        disabled={!isEnabled}
                    />
                )}
            />

            <Controller
                name="force_https"
                control={control}
                render={({ field }) => (
                    <SwitchGroup
                        id="force_https"
                        title={intl.getMessage('encryption_force_redirect')}
                        checked={field.value}
                        onChange={field.onChange}
                        disabled={!isEnabled}
                    />
                )}
            />

            <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('encryption_certificates')}
            </h2>

            <p className={cn(s.description, theme.text.t2)}>
                {intl.getMessage('encryption_certificates_desc', {
                    a: (text: string) => (
                        <a href="https://letsencrypt.org/" target="_blank" rel="noreferrer" className={theme.link.link}>
                            {text}
                        </a>
                    ),
                })}
            </p>

            <div className={theme.form.group}>
                <Controller
                    name="certificate_source"
                    control={control}
                    render={({ field }) => (
                        <Radio
                            value={field.value}
                            handleChange={field.onChange}
                            name={field.name}
                            options={certificateSourceOptions}
                            disabled={!isEnabled}
                        />
                    )}
                />

                <div className={theme.form.input}>
                    {certificateSource === ENCRYPTION_SOURCE.CONTENT ? (
                        <Controller
                            name="certificate_chain"
                            control={control}
                            render={({ field, fieldState }) => (
                                <Textarea
                                    {...field}
                                    placeholder={intl.getMessage('encryption_certificates_input')}
                                    disabled={!isEnabled}
                                    errorMessage={fieldState.error?.message}
                                    onBlur={handleBlur}
                                    size="large"
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
                                    placeholder={intl.getMessage('encryption_certificate_path')}
                                    errorMessage={fieldState.error?.message}
                                    disabled={!isEnabled}
                                    onBlur={handleBlur}
                                    size="medium"
                                />
                            )}
                        />
                    )}
                </div>

                {renderCertificateStatus()}
            </div>

            <h2 className={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('encryption_key')}
            </h2>

            <div className={theme.form.group}>
                <Controller
                    name="key_source"
                    control={control}
                    render={({ field }) => (
                        <Radio
                            value={field.value}
                            handleChange={field.onChange}
                            name={field.name}
                            options={keySourceOptions}
                            disabled={!isEnabled}
                        />
                    )}
                />

                <Controller
                    name="private_key_saved"
                    control={control}
                    render={({ field: { value, onChange, name } }) => (
                        <Checkbox
                            name={name}
                            disabled={!isEnabled || privateKeySource !== ENCRYPTION_SOURCE.CONTENT}
                            checked={value}
                            onChange={({ target: { checked } }) => {
                                if (checked) {
                                    setValue('private_key', '');
                                }
                                onChange(checked);
                            }}
                            onBlur={handleBlur}
                            className={s.useSavedKey}>
                            {intl.getMessage('use_saved_key')}
                        </Checkbox>
                    )}
                />

                <div className={theme.form.input}>
                    {privateKeySource === ENCRYPTION_SOURCE.CONTENT ? (
                        <Controller
                            name="private_key"
                            control={control}
                            render={({ field, fieldState }) => (
                                <Textarea
                                    {...field}
                                    placeholder={intl.getMessage('encryption_key_input')}
                                    disabled={!isEnabled || privateKeySaved}
                                    errorMessage={fieldState.error?.message}
                                    onBlur={handleBlur}
                                    size="large"
                                />
                            )}
                        />
                    ) : (
                        <Controller
                            name="private_key_path"
                            control={control}
                            render={({ field, fieldState }) => (
                                <Input
                                    {...field}
                                    type="text"
                                    placeholder={intl.getMessage('encryption_private_key_path')}
                                    errorMessage={fieldState.error?.message}
                                    disabled={!isEnabled}
                                    onBlur={handleBlur}
                                    size="medium"
                                />
                            )}
                        />
                    )}
                </div>

                {(privateKey || privateKeyPath) && <KeyStatus validKey={valid_key} keyType={key_type} />}
            </div>

            <div className={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={isDisabled}
                    className={theme.form.button}>
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    variant="secondary-danger"
                    size="small"
                    disabled={isSubmitting || processingConfig}
                    onClick={handleResetOpen}
                    className={theme.form.button}>
                    {intl.getMessage('reset')}
                </Button>
            </div>

            {openConfirmReset && (
                <ConfirmDialog
                    onClose={handleResetClose}
                    onConfirm={handleReset}
                    buttonText={intl.getMessage('reset')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('encryption_confirm_clear')}
                    text={intl.getMessage('encryption_confirm_clear_desc')}
                    buttonVariant="danger"
                />
            )}

            {openPlainDnsDisable && (
                <ConfirmDialog
                    onClose={handlePlainDnsDisableClose}
                    onConfirm={handlePlainDnsDisableConfirm}
                    buttonText={intl.getMessage('disable')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('encryption_disable_plain_dns')}
                    text={intl.getMessage('encryption_disable_plain_dns_desc')}
                    buttonVariant="danger"
                />
            )}
        </form>
    );
};
