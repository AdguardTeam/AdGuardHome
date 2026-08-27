import { createEffect, createSignal, on, onCleanup, Show } from 'solid-js';
import { createStore } from 'solid-js/store';

import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Button } from 'panel/common/ui/Button';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { Textarea } from 'panel/common/controls/Textarea';
import { FaqTooltip } from 'panel/common/ui/FaqTooltip';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    ENCRYPTION_SOURCE,
    STANDARD_HTTPS_PORT,
} from 'panel/helpers/constants';
import { normalizeServerName, toNumber } from 'panel/helpers/form';
import { validatePort, validateIsSafePort, validateServerName } from 'panel/helpers/validators';
import { encryptionState, setTlsConfig, validateTlsConfig } from 'panel/stores/encryption';
import {
    validateCertFields,
    validateEncryptionForm,
    validateKeyFields,
    type EncryptionFormValues,
} from '../../validate';
import { ValidationStatus } from '../../Status';
import {
    createDebouncedValidator,
    defaultTlsValues,
    getSubmitValues,
    type ValidationResult,
} from '../helpers';
import { FileBrowseButton } from '../AddTlsCert/FileBrowseButton';
import { WizardSteps } from './WizardSteps';
import s from './styles.module.pcss';

type Step = 1 | 2 | 3;

type Props = {
    open: boolean;
    onClose: () => void;
};

/**
 * Maps backend validation errors to user-facing messages.
 *
 * Contract (see the plan's Appendix A): 400 responses are plain text from
 * `validateTLSSettings`; 200 responses carry the diagnosis in
 * `warning_validation` with `valid_cert`/`valid_key`/`valid_pair` flags.
 * Only the stable inner substrings are matched — cert errors are wrapped with
 * the `validating certificate pair: ` prefix, file-read errors are not.
 */
const mapBackendError = (text: string): string => {
    const portBusyMatch = text.match(
        /port (\d+) for (HTTPS|DNS-over-TLS|DNS-over-QUIC) is not available/,
    );
    if (portBusyMatch) {
        return intl.getMessage('tls_setup_error_port_busy', {
            port: portBusyMatch[1],
            protocol: portBusyMatch[2],
        });
    }

    const dupMatch = text.match(/validating (?:tcp|udp) ports: duplicated values: \[(\d+)/);
    if (dupMatch) {
        return intl.getMessage('tls_setup_error_duplicate_port', { port: dupMatch[1] });
    }

    if (text.includes('reading cert file')) {
        return intl.getMessage('encryption_unable_read_cert');
    }
    if (text.includes('empty certificate')) {
        return intl.getMessage('tls_setup_error_empty_cert');
    }
    if (text.includes('parsing certificate at index')) {
        return intl.getMessage('tls_setup_error_parse_cert');
    }
    if (text.includes('reading key file')) {
        return intl.getMessage('encryption_unable_read_key');
    }
    if (text.includes('no valid keys were found')) {
        return intl.getMessage('tls_setup_error_no_key');
    }
    if (text.includes('parsing private key')) {
        return intl.getMessage('tls_setup_error_parse_key');
    }
    if (text.includes('ED25519 keys are not supported')) {
        return intl.getMessage('tls_setup_error_ed25519_key');
    }
    if (text.includes('certificate-key pair')) {
        return intl.getMessage('encryption_key_cert_mismatch');
    }

    if (text.includes('certificate does not verify')) {
        return intl.getMessage('tls_setup_warning_cert_untrusted');
    }
    if (text.includes('certificates has no IP addresses')) {
        return intl.getMessage('tls_setup_warning_no_ip');
    }

    // Unrecognized text — show raw as fallback.
    return text;
};

export const TlsSetupWizard = (props: Props) => {
    const [step, setStep] = createSignal<Step>(1);
    const [values, setValues] = createStore<EncryptionFormValues>({ ...defaultTlsValues });
    const [errors, setErrors] = createSignal<Record<string, string>>({});

    // Backend pre-flight result (backend errors are inline only — no toasts).
    const [backendMessage, setBackendMessage] = createSignal('');
    const [backendBlocking, setBackendBlocking] = createSignal(false);
    const [backendWarning, setBackendWarning] = createSignal(false);
    const [validating, setValidating] = createSignal(false);

    createEffect(
        on(
            () => props.open,
            (open) => {
                if (!open) return;

                setStep(1);
                setValues({ ...defaultTlsValues });
                setValues({
                    enabled: encryptionState.enabled ?? false,
                    // The wizard always enables encrypted DNS together with
                    // plain DNS — don't inherit a flipped-off store value.
                    serve_plain_dns: true,
                    server_name: encryptionState.server_name ?? '',
                    force_https: encryptionState.force_https ?? false,
                    port_https: Number(encryptionState.port_https) || STANDARD_HTTPS_PORT,
                    port_dns_over_tls:
                        Number(encryptionState.port_dns_over_tls) || DNS_OVER_TLS_PORT,
                    port_dns_over_quic:
                        Number(encryptionState.port_dns_over_quic) || DNS_OVER_QUIC_PORT,
                    certificate_source: ENCRYPTION_SOURCE.CONTENT,
                    key_source: ENCRYPTION_SOURCE.CONTENT,
                });
                setErrors({});
                setBackendMessage('');
                setBackendBlocking(false);
                setBackendWarning(false);
                setValidating(false);
            },
        ),
    );

    const clearError = (field: string) => {
        setErrors((prev) => {
            if (!(field in prev)) return prev;
            const next = { ...prev };
            delete next[field];
            return next;
        });
    };

    const error = (field: string) => errors()[field];

    /**
     * Wizard-specific client gate: the server name is required in the wizard
     * (`validateEncryptionForm`'s `validateServerName` is format-only and
     * accepts empty values), plus the shared full-form validation.
     */
    const clientErrors = (): Record<string, string> => {
        const errs: Record<string, string> = {};
        if (!String(values.server_name ?? '').trim()) {
            errs.server_name = intl.getMessage('form_error_required');
        }
        Object.assign(errs, validateEncryptionForm(values));
        return errs;
    };

    const handleValidationResult = (res: ValidationResult) => {
        setBackendWarning(false);

        if ('error' in res) {
            setBackendMessage(mapBackendError(res.error));
            setBackendBlocking(true);
            return;
        }

        const pairValid = !!(res.valid_cert && res.valid_key && res.valid_pair);
        if (!pairValid) {
            setBackendMessage(mapBackendError(res.warning_validation ?? ''));
            setBackendBlocking(true);
            return;
        }

        if (res.warning_validation) {
            setBackendMessage(mapBackendError(res.warning_validation));
            setBackendBlocking(false);
            setBackendWarning(true);
            return;
        }

        setBackendMessage('');
        setBackendBlocking(false);
    };

    const [validateConfig, cancelValidation] = createDebouncedValidator({
        persist: false,
        onResult: handleValidationResult,
    });

    onCleanup(() => {
        cancelValidation();
    });

    // Pre-flight: on step 3, whenever the client-side values are clean,
    // debounce a validation call with the exact final-save payload so port
    // probes and the plain-DNS rule see what "Enable" will send.
    createEffect(() => {
        if (step() !== 3) return;
        const errs = clientErrors();
        if (Object.values(errs).some(Boolean)) return;
        validateConfig({ ...values, enabled: true, serve_plain_dns: true });
    });

    // ----- Step 1: certificate -----

    const certSourceOptions = [
        {
            text: intl.getMessage('tls_setup_cert_text_option'),
            description: intl.getMessage('tls_setup_cert_text_option_desc'),
            value: ENCRYPTION_SOURCE.CONTENT,
        },
        {
            text: intl.getMessage('tls_setup_cert_path_option'),
            description: intl.getMessage('tls_setup_cert_path_option_desc'),
            value: ENCRYPTION_SOURCE.PATH,
        },
    ];

    const handleCertSourceChange = (v: string) => {
        setValues('certificate_source', v);
        clearError('certificate_chain');
        clearError('certificate_path');
    };

    const handleCertChainChange = (e: Event) => {
        setValues('certificate_chain', (e.target as HTMLTextAreaElement).value);
        clearError('certificate_chain');
    };

    const handleCertPathChange = (e: Event) => {
        setValues('certificate_path', (e.target as HTMLInputElement).value);
        clearError('certificate_path');
    };

    const handleCertFileSelect = (content: string) => {
        setValues('certificate_chain', content);
        setValues('certificate_source', ENCRYPTION_SOURCE.CONTENT);
        clearError('certificate_chain');
        clearError('certificate_path');
    };

    const validateCertOnBlur = () => {
        const errs = validateCertFields(values);
        setErrors((prev) => ({ ...prev, ...errs }));
    };

    const handleCertNext = () => {
        const errs = validateCertFields(values);
        setErrors(errs);
        if (Object.values(errs).some(Boolean)) return;
        setStep(2);
    };

    // ----- Step 2: private key -----

    const keySourceOptions = () => [
        {
            text: intl.getMessage('tls_setup_key_text_option'),
            description: intl.getMessage('tls_setup_key_text_option_desc'),
            value: ENCRYPTION_SOURCE.CONTENT,
        },
        {
            text: intl.getMessage('tls_setup_key_path_option'),
            description: intl.getMessage('tls_setup_key_path_option_desc'),
            value: ENCRYPTION_SOURCE.PATH,
        },
        {
            text: intl.getMessage('use_saved_key'),
            value: ENCRYPTION_SOURCE.SAVED,
            disabled: !encryptionState.private_key_saved,
        },
    ];

    // Mirrors AddTlsCertModal's handleKeySourceChange: SAVED clears the
    // pasted key and flags the saved one; the path is emptied on submit by
    // getSubmitValues.
    const handleKeySourceChange = (v: string) => {
        setValues('key_source', v);
        if (v === ENCRYPTION_SOURCE.SAVED) {
            setValues('private_key', '');
            setValues('private_key_saved', true);
        } else {
            setValues('private_key_saved', false);
        }
        clearError('private_key');
        clearError('private_key_path');
    };

    const handleKeyChange = (e: Event) => {
        setValues('private_key', (e.target as HTMLTextAreaElement).value);
        clearError('private_key');
    };

    const handleKeyPathChange = (e: Event) => {
        setValues('private_key_path', (e.target as HTMLInputElement).value);
        clearError('private_key_path');
    };

    const handleKeyFileSelect = (content: string) => {
        setValues('private_key', content);
        setValues('key_source', ENCRYPTION_SOURCE.CONTENT);
        setValues('private_key_saved', false);
        clearError('private_key');
        clearError('private_key_path');
    };

    const validateKeyOnBlur = () => {
        const errs = validateKeyFields(values);
        setErrors((prev) => ({ ...prev, ...errs }));
    };

    const handleKeyNext = () => {
        const errs = validateKeyFields(values);
        setErrors(errs);
        if (Object.values(errs).some(Boolean)) return;
        setStep(3);
    };

    // ----- Step 3: config & enable -----

    const handleServerNameChange = (e: Event) => {
        setValues('server_name', (e.target as HTMLInputElement).value);
        clearError('server_name');
    };

    const handleServerNameBlur = () => {
        const normalized = normalizeServerName(String(values.server_name ?? ''));
        setValues('server_name', normalized);

        const err = validateServerName(normalized);
        setErrors((prev) => {
            const next = { ...prev };
            if (err) {
                next.server_name = err;
            } else {
                delete next.server_name;
            }
            return next;
        });
    };

    const portHandler = (field: 'port_https' | 'port_dns_over_tls' | 'port_dns_over_quic') => {
        return (e: Event) => {
            setValues(field, toNumber((e.target as HTMLInputElement).value));
            clearError(field);
        };
    };

    const portBlurHandler = (field: 'port_https' | 'port_dns_over_tls' | 'port_dns_over_quic') => {
        return (e: Event) => {
            const port = toNumber((e.target as HTMLInputElement).value) || 0;
            const err =
                validatePort(port) ||
                (field === 'port_https' ? validateIsSafePort(port) : undefined);
            setErrors((prev) => {
                const next = { ...prev };
                if (err) {
                    next[field] = err as string;
                } else {
                    delete next[field];
                }
                return next;
            });
        };
    };

    const handleGoBack = () => {
        setErrors({});
        setBackendMessage('');
        setBackendBlocking(false);
        setBackendWarning(false);
        setStep((prev) => (prev > 1 ? ((prev - 1) as Step) : prev));
    };

    const handleEnable = async () => {
        const errs = clientErrors();
        if (Object.values(errs).some(Boolean)) {
            setErrors(errs);
            return;
        }

        cancelValidation();
        setValidating(true);
        const res = await validateTlsConfig(
            getSubmitValues({ ...values, enabled: true, serve_plain_dns: true }),
            { persist: false },
        );
        setValidating(false);

        if ('error' in res) {
            handleValidationResult(res);
            return;
        }

        const pairValid = !!(res.valid_cert && res.valid_key && res.valid_pair);
        if (!pairValid) {
            handleValidationResult(res);
            return;
        }

        setTlsConfig(getSubmitValues({ ...values, enabled: true, serve_plain_dns: true }));
        props.onClose();
    };

    const hasClientErrors = () => Object.values(clientErrors()).some(Boolean);
    const enableDisabled = () => validating() || hasClientErrors() || backendBlocking();

    const titles = [
        'tls_setup_cert_title',
        'tls_setup_key_title',
        'tls_setup_config_title',
    ] as const;
    const descriptions: readonly (string | undefined)[] = [
        'tls_setup_cert_description',
        'tls_setup_key_description',
        undefined,
    ];

    const footer = () =>
        step() === 3 ? (
            <div class={s.footer}>
                <Button
                    variant="primary"
                    onClick={handleEnable}
                    disabled={enableDisabled()}
                    data-testid="tls-setup-enable"
                >
                    {intl.getMessage('enable')}
                </Button>
                <Button variant="secondary" onClick={props.onClose}>
                    {intl.getMessage('cancel')}
                </Button>
            </div>
        ) : (
            <div class={s.footer}>
                <Button
                    variant="primary"
                    onClick={step() === 1 ? handleCertNext : handleKeyNext}
                    data-testid="tls-setup-add"
                >
                    {intl.getMessage('add')}
                </Button>
                <Button variant="secondary" onClick={props.onClose}>
                    {intl.getMessage('cancel')}
                </Button>
            </div>
        );

    return (
        <ConfigDialog
            open={props.open}
            title=""
            onClose={props.onClose}
            onSubmit={handleEnable}
            hideSubmit
            footer={footer()}
        >
            {/* Per design: the stepper sits at the top of the dialog, above
                the title, and the dialog has no close icon button. */}
            <WizardSteps step={step()} onGoBack={handleGoBack} />
            <h2 class={s.wizardTitle}>{intl.getMessage(titles[step() - 1])}</h2>
            <Show when={descriptions[step() - 1]}>
                <div class={s.wizardDescription}>
                    {intl.getMessage(descriptions[step() - 1] as string)}
                </div>
            </Show>

            <Show when={step() === 1}>
                <div class={s.content}>
                    <Radio
                        value={values.certificate_source ?? ''}
                        handleChange={handleCertSourceChange}
                        name="tls_setup_certificate_source"
                        options={certSourceOptions}
                        inModal
                    />
                    <Show
                        when={values.certificate_source === ENCRYPTION_SOURCE.CONTENT}
                        fallback={
                            <div class={theme.form.input}>
                                <Input
                                    id="tls_setup_certificate_path"
                                    name="certificate_path"
                                    value={values.certificate_path ?? ''}
                                    onChange={handleCertPathChange}
                                    onBlur={validateCertOnBlur}
                                    placeholder={intl.getMessage('path_to_file_placeholder')}
                                    errorMessage={error('certificate_path')}
                                    label={intl.getMessage('tls_cert_path_label')}
                                    suffixIcon={
                                        <FileBrowseButton onFileSelect={handleCertFileSelect} />
                                    }
                                    size="large"
                                />
                            </div>
                        }
                    >
                        <div class={theme.form.input}>
                            <Textarea
                                id="tls_setup_certificate_chain"
                                name="certificate_chain"
                                value={values.certificate_chain ?? ''}
                                onChange={handleCertChainChange}
                                onBlur={validateCertOnBlur}
                                placeholder="-----BEGIN CERTIFICATE-----"
                                errorMessage={error('certificate_chain')}
                                label={intl.getMessage('tls_setup_cert_paste_label')}
                                size="large"
                            />
                        </div>
                    </Show>
                </div>
            </Show>

            <Show when={step() === 2}>
                <div class={s.content}>
                    <Radio
                        value={values.key_source ?? ''}
                        handleChange={handleKeySourceChange}
                        name="tls_setup_key_source"
                        options={keySourceOptions()}
                        inModal
                    />
                    <Show
                        when={values.key_source === ENCRYPTION_SOURCE.CONTENT}
                        fallback={
                            <div class={theme.form.input}>
                                <Input
                                    id="tls_setup_private_key_path"
                                    name="private_key_path"
                                    value={values.private_key_path ?? ''}
                                    onChange={handleKeyPathChange}
                                    onBlur={validateKeyOnBlur}
                                    placeholder={intl.getMessage('path_to_file_placeholder')}
                                    errorMessage={error('private_key_path')}
                                    label={intl.getMessage('tls_key_path_label')}
                                    suffixIcon={
                                        <FileBrowseButton onFileSelect={handleKeyFileSelect} />
                                    }
                                    size="large"
                                />
                            </div>
                        }
                    >
                        <div class={theme.form.input}>
                            <Textarea
                                id="tls_setup_private_key"
                                name="private_key"
                                value={values.private_key ?? ''}
                                onChange={handleKeyChange}
                                onBlur={validateKeyOnBlur}
                                placeholder="-----BEGIN PRIVATE KEY-----"
                                errorMessage={error('private_key')}
                                disabled={!!values.private_key_saved}
                                label={intl.getMessage('tls_setup_key_paste_label')}
                                size="large"
                            />
                        </div>
                    </Show>
                </div>
            </Show>

            <Show when={step() === 3}>
                <div class={s.content}>
                    <div class={theme.form.input}>
                        <Input
                            id="tls_setup_server_name"
                            name="server_name"
                            value={values.server_name ?? ''}
                            onChange={handleServerNameChange}
                            onBlur={handleServerNameBlur}
                            label={
                                <>
                                    {intl.getMessage('tls_setup_server_name_label')}
                                    <FaqTooltip
                                        menuSize="large"
                                        text={
                                            <>
                                                <div class={s.description}>
                                                    {intl.getMessage('encryption_server_tooltip_1')}
                                                </div>
                                                <div class={s.description}>
                                                    {intl.getMessage('encryption_server_tooltip_2')}
                                                </div>
                                            </>
                                        }
                                    />
                                </>
                            }
                            placeholder={intl.getMessage('encryption_server_enter')}
                            errorMessage={
                                error('server_name') ??
                                (String(values.server_name ?? '').trim()
                                    ? undefined
                                    : intl.getMessage('form_error_required'))
                            }
                            size="large"
                        />
                    </div>

                    <div class={theme.form.input}>
                        <Input
                            id="tls_setup_port_https"
                            name="port_https"
                            type="number"
                            value={values.port_https ?? ''}
                            onChange={portHandler('port_https')}
                            onBlur={portBlurHandler('port_https')}
                            isClearable
                            label={
                                <>
                                    {intl.getMessage('tls_setup_https_port_label')}
                                    <FaqTooltip
                                        menuSize="large"
                                        text={intl.getMessage('encryption_https_tooltip')}
                                    />
                                </>
                            }
                            errorMessage={error('port_https')}
                            size="large"
                        />
                    </div>

                    <div class={theme.form.input}>
                        <Input
                            id="tls_setup_port_dns_over_tls"
                            name="port_dns_over_tls"
                            type="number"
                            value={values.port_dns_over_tls ?? ''}
                            onChange={portHandler('port_dns_over_tls')}
                            onBlur={portBlurHandler('port_dns_over_tls')}
                            isClearable
                            label={
                                <>
                                    {intl.getMessage('tls_setup_dot_port_label')}
                                    <FaqTooltip
                                        menuSize="large"
                                        text={intl.getMessage('encryption_dot_tooltip')}
                                    />
                                </>
                            }
                            errorMessage={error('port_dns_over_tls')}
                            size="large"
                        />
                    </div>

                    <div class={theme.form.input}>
                        <Input
                            id="tls_setup_port_dns_over_quic"
                            name="port_dns_over_quic"
                            type="number"
                            value={values.port_dns_over_quic ?? ''}
                            onChange={portHandler('port_dns_over_quic')}
                            onBlur={portBlurHandler('port_dns_over_quic')}
                            isClearable
                            label={
                                <>
                                    {intl.getMessage('tls_setup_doq_port_label')}
                                    <FaqTooltip
                                        menuSize="large"
                                        text={intl.getMessage('encryption_doq_tooltip')}
                                    />
                                </>
                            }
                            errorMessage={error('port_dns_over_quic')}
                            size="large"
                        />
                    </div>

                    <Show when={backendMessage()}>
                        <div class={s.backendError}>
                            <ValidationStatus
                                type={backendWarning() ? 'warning' : 'error'}
                                message={backendMessage()}
                            />
                            {!backendWarning() && (
                                <div class={s.goBackHint}>
                                    {intl.getMessage('tls_setup_hint_go_back')}
                                </div>
                            )}
                        </div>
                    </Show>
                </div>
            </Show>
        </ConfigDialog>
    );
};
