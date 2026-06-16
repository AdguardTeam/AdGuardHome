import { createSignal, createMemo, Show } from 'solid-js';
import cn from 'clsx';

import { toNumber } from 'panel/helpers/form';
import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    STANDARD_HTTPS_PORT,
    ENCRYPTION_SOURCE,
} from 'panel/helpers/constants';
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
import { setTlsConfig, validateTlsConfig } from 'panel/stores/encryption';
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
        text: intl.getMessage('encryption_certificates_source_path'),
        value: ENCRYPTION_SOURCE.PATH,
    },
    {
        text: intl.getMessage('encryption_certificates_source_content'),
        value: ENCRYPTION_SOURCE.CONTENT,
    },
];

const keySourceOptions = [
    {
        text: intl.getMessage('encryption_key_source_path'),
        value: ENCRYPTION_SOURCE.PATH,
    },
    {
        text: intl.getMessage('encryption_key_source_content'),
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

export const Form = (props: Props) => {
    const iv = props.initialValues;
    const [enabled, setEnabled] = createSignal(iv.enabled ?? defaultValues.enabled);
    const [servePlainDns, setServePlainDns] = createSignal(iv.serve_plain_dns ?? defaultValues.serve_plain_dns);
    const [serverName, setServerName] = createSignal(iv.server_name ?? defaultValues.server_name);
    const [forceHttps, setForceHttps] = createSignal(iv.force_https ?? defaultValues.force_https);
    const [portHttps, setPortHttps] = createSignal(iv.port_https ?? defaultValues.port_https);
    const [portDot, setPortDot] = createSignal(iv.port_dns_over_tls ?? defaultValues.port_dns_over_tls);
    const [portDoq, setPortDoq] = createSignal(iv.port_dns_over_quic ?? defaultValues.port_dns_over_quic);
    const [certChain, setCertChain] = createSignal(iv.certificate_chain ?? defaultValues.certificate_chain);
    const [privateKey, setPrivateKey] = createSignal(iv.private_key ?? defaultValues.private_key);
    const [certPath, setCertPath] = createSignal(iv.certificate_path ?? defaultValues.certificate_path);
    const [privateKeyPath, setPrivateKeyPath] = createSignal(iv.private_key_path ?? defaultValues.private_key_path);
    const [certSource, setCertSource] = createSignal(iv.certificate_source ?? defaultValues.certificate_source);
    const [keySource, setKeySource] = createSignal(iv.key_source ?? defaultValues.key_source);
    const [privateKeySaved, setPrivateKeySaved] = createSignal(iv.private_key_saved ?? defaultValues.private_key_saved);

    // Validation errors
    const [errors, setErrors] = createSignal<Record<string, string>>({});
    const [openConfirmReset, setOpenConfirmReset] = createSignal(false);
    const [openPlainDnsDisable, setOpenPlainDnsDisable] = createSignal(false);
    const [stagedFormValues, setStagedFormValues] = createSignal<EncryptionFormValues | null>(null);
    const [isSubmitting, setIsSubmitting] = createSignal(false);

    const enc = () => props.encryption;

    const getFormValues = (): EncryptionFormValues => ({
        enabled: enabled(),
        serve_plain_dns: servePlainDns(),
        server_name: serverName(),
        force_https: forceHttps(),
        port_https: portHttps(),
        port_dns_over_tls: portDot(),
        port_dns_over_quic: portDoq(),
        certificate_chain: certChain(),
        private_key: privateKey(),
        certificate_path: certPath(),
        private_key_path: privateKeyPath(),
        certificate_source: certSource(),
        key_source: keySource(),
        private_key_saved: privateKeySaved(),
    });

    const handleBlur = () => {
        props.debouncedConfigValidation(getFormValues());
    };

    const isSavingDisabled = createMemo(() => {
        const processing = isSubmitting() || enc().processingConfig || enc().processingValidate;
        const errs = errors();
        const hasErrors = Object.keys(errs).length > 0;

        if (servePlainDns() && !enabled()) {
            return hasErrors || processing;
        }

        return hasErrors || processing || !enc().valid_key || !enc().valid_cert || !enc().valid_pair;
    });

    const handleReset = () => {
        setEnabled(defaultValues.enabled);
        setServePlainDns(defaultValues.serve_plain_dns);
        setServerName(defaultValues.server_name);
        setForceHttps(defaultValues.force_https);
        setPortHttps(defaultValues.port_https);
        setPortDot(defaultValues.port_dns_over_tls);
        setPortDoq(defaultValues.port_dns_over_quic);
        setCertChain(defaultValues.certificate_chain);
        setPrivateKey(defaultValues.private_key);
        setCertPath(defaultValues.certificate_path);
        setPrivateKeyPath(defaultValues.private_key_path);
        setCertSource(defaultValues.certificate_source);
        setKeySource(defaultValues.key_source);
        setPrivateKeySaved(defaultValues.private_key_saved);
        setTlsConfig(defaultValues);
        validateTlsConfig(defaultValues);
        setOpenConfirmReset(false);
    };

    const validatePorts = (values: EncryptionFormValues) => {
        const errs: Record<string, string> = {};
        if (values.port_dns_over_tls && values.port_https) {
            if (values.port_dns_over_tls === values.port_https) {
                errs.port_dns_over_tls = intl.getMessage('form_error_equal');
                errs.port_https = intl.getMessage('form_error_equal');
            }
        }
        return errs;
    };

    const onFormSubmit = (e: Event) => {
        e.preventDefault();
        setIsSubmitting(true);
        const data = getFormValues();

        // Validate server name
        const serverNameErr = validateServerName(serverName());
        if (serverNameErr) { setErrors(prev => ({ ...prev, server_name: serverNameErr })); return; }

        // Validate ports
        const portHttpsErr = validatePort(portHttps()) || validateIsSafePort(portHttps());
        if (portHttpsErr) { setErrors(prev => ({ ...prev, port_https: portHttpsErr as string })); }

        const portDotErr = validatePortTLS(portDot());
        if (portDotErr) { setErrors(prev => ({ ...prev, port_dns_over_tls: portDotErr as string })); }

        const portDoqErr = validatePortQuic(portDoq());
        if (portDoqErr) { setErrors(prev => ({ ...prev, port_dns_over_quic: portDoqErr as string })); }

        const validationErrors = validatePorts(data);
        if (Object.keys(validationErrors).length > 0) {
            setErrors(prev => ({ ...prev, ...validationErrors }));
            setIsSubmitting(false);
            return;
        }

        // Validate plain DNS
        const plainDnsErr = validatePlainDns(servePlainDns(), data);
        if (plainDnsErr) { setErrors(prev => ({ ...prev, serve_plain_dns: plainDnsErr as string })); setIsSubmitting(false); return; }

        setErrors({});
        setIsSubmitting(false);

        if (data.serve_plain_dns === false) {
            setStagedFormValues(data);
            setOpenPlainDnsDisable(true);
            return;
        }

        props.onSubmit(data);
    };

    const renderCertificateStatus = () => {
        if (enc().warning_validation) {
            const isWarning = enc().valid_key && enc().valid_cert && enc().valid_pair;
            return (
                <ValidationStatus
                    type={isWarning ? 'warning' : 'error'}
                    message={enc().warning_validation}
                />
            );
        }

        if (!certChain() && !certPath()) {
            return null;
        }

        return (
            <CertificateStatus
                validChain={enc().valid_chain}
                validCert={enc().valid_cert}
                subject={enc().subject}
                issuer={enc().issuer}
                notAfter={enc().not_after}
                dnsNames={enc().dns_names}
            />
        );
    };

    return (
        <form onSubmit={onFormSubmit}>
            <SwitchGroup
                id="enabled"
                title={intl.getMessage('encryption_encrypted_dns')}
                description={intl.getMessage('encryption_encrypted_dns_desc')}
                checked={enabled()}
                onChange={(e: Event) => setEnabled((e.target as HTMLInputElement).checked)}
            >
                <div class={s.group}>
                    <div>
                        <Input
                            type="text"
                            name="server_name"
                            value={serverName()}
                            onChange={(e: Event) => setServerName((e.target as HTMLInputElement).value)}
                            label={
                                <>
                                    {intl.getMessage('encryption_server')}

                                    <FaqTooltip
                                        text={
                                            <>
                                                <div class={s.tooltipText}>
                                                    {intl.getMessage(
                                                        'encryption_server_tooltip_1',
                                                    )}
                                                </div>
                                                <div class={s.tooltipText}>
                                                    {intl.getMessage(
                                                        'encryption_server_tooltip_2',
                                                    )}
                                                </div>
                                            </>
                                        }
                                        menuSize="large"
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('encryption_server_enter')}
                            errorMessage={errors().server_name}
                            disabled={!enabled()}
                            onBlur={handleBlur}
                        />
                    </div>
                    <div>
                        <Input
                            type="number"
                            name="port_https"
                            value={portHttps()}
                            onChange={(e: Event) => {
                                setPortHttps(toNumber((e.target as HTMLInputElement).value));
                            }}
                            label={
                                <>
                                    {intl.getMessage('encryption_https')}

                                    <FaqTooltip
                                        text={intl.getMessage(
                                            'encryption_https_tooltip',
                                        )}
                                        menuSize="large"
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('encryption_https')}
                            errorMessage={errors().port_https}
                            disabled={!enabled()}
                            onBlur={handleBlur}
                        />
                    </div>
                    <div>
                        <Input
                            type="number"
                            name="port_dns_over_tls"
                            value={portDot()}
                            onChange={(e: Event) => {
                                setPortDot(toNumber((e.target as HTMLInputElement).value));
                            }}
                            label={
                                <>
                                    {intl.getMessage('encryption_dot')}

                                    <FaqTooltip
                                        text={intl.getMessage(
                                            'encryption_dot_tooltip',
                                        )}
                                        menuSize="large"
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('encryption_dot')}
                            errorMessage={errors().port_dns_over_tls}
                            disabled={!enabled()}
                            onBlur={handleBlur}
                        />
                    </div>
                    <div>
                        <Input
                            type="number"
                            name="port_dns_over_quic"
                            value={portDoq()}
                            onChange={(e: Event) => {
                                setPortDoq(toNumber((e.target as HTMLInputElement).value));
                            }}
                            label={
                                <>
                                    {intl.getMessage('encryption_doq')}

                                    <FaqTooltip
                                        text={intl.getMessage(
                                            'encryption_doq_tooltip',
                                        )}
                                        menuSize="large"
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('encryption_doq')}
                            errorMessage={errors().port_dns_over_quic}
                            disabled={!enabled()}
                            onBlur={handleBlur}
                        />
                    </div>
                </div>
            </SwitchGroup>

            <SwitchGroup
                id="serve_plain_dns"
                title={intl.getMessage('encryption_plain_dns')}
                description={intl.getMessage('encryption_plain_dns_desc')}
                checked={servePlainDns()}
                onChange={(e: Event) => setServePlainDns((e.target as HTMLInputElement).checked)}
                disabled={!enabled()}
            />

            <SwitchGroup
                id="force_https"
                title={intl.getMessage('encryption_force_redirect')}
                checked={forceHttps()}
                onChange={(e: Event) => setForceHttps((e.target as HTMLInputElement).checked)}
                disabled={!enabled()}
            />

            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('encryption_certificates')}
            </h2>

            <p class={cn(s.description, theme.text.t2)}>
                {intl.getMessage('encryption_certificates_desc', {
                    a: (text: string) => (
                        <a
                            href="https://letsencrypt.org/"
                            target="_blank"
                            rel="noreferrer"
                            class={theme.link.link}
                        >
                            {text}
                        </a>
                    ),
                })}
            </p>

            <div class={theme.form.group}>
                <Radio
                    value={certSource()}
                    handleChange={(v: string) => setCertSource(v)}
                    name="certificate_source"
                    options={certificateSourceOptions}
                    disabled={!enabled()}
                />

                <div class={theme.form.input}>
                    <Show when={certSource() === ENCRYPTION_SOURCE.CONTENT} fallback={
                        <Input
                            type="text"
                            name="certificate_path"
                            value={certPath()}
                            onChange={(e: Event) => setCertPath((e.target as HTMLInputElement).value)}
                            placeholder={intl.getMessage('encryption_certificate_path')}
                            errorMessage={errors().certificate_path}
                            disabled={!enabled()}
                            onBlur={handleBlur}
                            size="medium"
                        />
                    }>
                        <Textarea
                            name="certificate_chain"
                            value={certChain()}
                            onChange={(e: Event) => setCertChain((e.target as HTMLTextAreaElement).value)}
                            placeholder={intl.getMessage('encryption_certificates_input')}
                            disabled={!enabled()}
                            errorMessage={errors().certificate_chain}
                            onBlur={handleBlur}
                            size="large"
                        />
                    </Show>
                </div>

                {renderCertificateStatus()}
            </div>

            <h2 class={cn(theme.layout.subtitle, theme.title.h5, theme.title.h4_tablet)}>
                {intl.getMessage('encryption_key')}
            </h2>

            <div class={theme.form.group}>
                <Radio
                    value={keySource()}
                    handleChange={(v: string) => setKeySource(v)}
                    name="key_source"
                    options={keySourceOptions}
                    disabled={!enabled()}
                />

                <Checkbox
                    name="private_key_saved"
                    disabled={!enabled() || keySource() !== ENCRYPTION_SOURCE.CONTENT}
                    checked={privateKeySaved()}
                    onChange={(e: Event) => {
                        const checked = (e.target as HTMLInputElement).checked;
                        if (checked) {
                            setPrivateKey('');
                        }
                        setPrivateKeySaved(checked);
                    }}
                    onBlur={handleBlur}
                    class={s.useSavedKey}
                >
                    {intl.getMessage('use_saved_key')}
                </Checkbox>

                <div class={theme.form.input}>
                    <Show when={keySource() === ENCRYPTION_SOURCE.CONTENT} fallback={
                        <Input
                            type="text"
                            name="private_key_path"
                            value={privateKeyPath()}
                            onChange={(e: Event) => setPrivateKeyPath((e.target as HTMLInputElement).value)}
                            placeholder={intl.getMessage('encryption_private_key_path')}
                            errorMessage={errors().private_key_path}
                            disabled={!enabled()}
                            onBlur={handleBlur}
                            size="medium"
                        />
                    }>
                        <Textarea
                            name="private_key"
                            value={privateKey()}
                            onChange={(e: Event) => setPrivateKey((e.target as HTMLTextAreaElement).value)}
                            placeholder={intl.getMessage('encryption_key_input')}
                            disabled={!enabled() || privateKeySaved()}
                            errorMessage={errors().private_key}
                            onBlur={handleBlur}
                            size="large"
                        />
                    </Show>
                </div>

                <Show when={privateKey() || privateKeyPath()}>
                    <KeyStatus validKey={enc().valid_key} keyType={enc().key_type} />
                </Show>
            </div>

            <div class={theme.form.buttonGroup}>
                <Button
                    type="submit"
                    variant="primary"
                    size="small"
                    disabled={isSavingDisabled()}
                    class={theme.form.button}
                >
                    {intl.getMessage('save')}
                </Button>

                <Button
                    type="button"
                    variant="secondary-danger"
                    size="small"
                    disabled={isSubmitting() || enc().processingConfig}
                    onClick={() => setOpenConfirmReset(true)}
                    class={theme.form.button}
                >
                    {intl.getMessage('reset')}
                </Button>
            </div>

            <Show when={openConfirmReset()}>
                <ConfirmDialog
                    onClose={() => setOpenConfirmReset(false)}
                    onConfirm={handleReset}
                    buttonText={intl.getMessage('reset')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('encryption_confirm_clear')}
                    text={intl.getMessage('encryption_confirm_clear_desc')}
                    buttonVariant="danger"
                />
            </Show>

            <Show when={openPlainDnsDisable()}>
                <ConfirmDialog
                    onClose={() => {
                        setOpenPlainDnsDisable(false);
                        setStagedFormValues(null);
                    }}
                    onConfirm={() => {
                        const vals = stagedFormValues();
                        if (vals) {
                            props.onSubmit(vals);
                            setStagedFormValues(null);
                        }
                        setOpenPlainDnsDisable(false);
                    }}
                    buttonText={intl.getMessage('disable')}
                    cancelText={intl.getMessage('cancel')}
                    title={intl.getMessage('encryption_disable_plain_dns')}
                    text={intl.getMessage('encryption_disable_plain_dns_desc')}
                    buttonVariant="danger"
                />
            </Show>
        </form>
    );
};
