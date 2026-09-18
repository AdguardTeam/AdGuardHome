import { createEffect, createSignal, on, Show, type JSX } from 'solid-js';
import { createStore } from 'solid-js/store';
import cn from 'clsx';

import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Button } from 'panel/common/ui/Button';
import { InlineLoader } from 'panel/common/ui/Loader';
import intl from 'panel/common/intl';
import theme from 'panel/lib/theme';
import {
    DNS_OVER_QUIC_PORT,
    DNS_OVER_TLS_PORT,
    ENCRYPTION_SOURCE,
    LETSENCRYPT_LINK,
    STANDARD_HTTPS_PORT,
} from 'panel/helpers/constants';
import { normalizeServerName, toNumber } from 'panel/helpers/form';
import { validateServerName } from 'panel/helpers/validators';
import { encryptionState, setTlsConfig } from 'panel/stores/encryption';
import {
    validateCertFields,
    validateEncryptionForm,
    validateKeyFields,
    validatePortField,
    type EncryptionFormValues,
    type ServerSettingsField,
} from '../../validate';
import { defaultTlsValues, getExternalPorts, getSubmitValues, type WizardStep } from '../helpers';
import { ServerSettingsFields } from '../ServerSettingsFields';
import { CERT_CONFIG, createPemFields, KEY_CONFIG } from './pemFields';
import { PemSourceFields } from './PemSourceFields';
import { StepFormMessage } from './StepFormMessage';
import { createStepCheck } from './useStepCheck';
import { WizardSteps } from './WizardSteps';
import s from './styles.module.pcss';

type Props = {
    open: boolean;
    onClose: () => void;
};

export const TlsSetupWizard = (props: Props) => {
    const [step, setStep] = createSignal<WizardStep>(1);
    const [values, setValues] = createStore<EncryptionFormValues>({ ...defaultTlsValues });
    const [errors, setErrors] = createSignal<Record<string, string>>({});
    const [saving, setSaving] = createSignal(false);

    const stepCheck = createStepCheck({ step, values, goToStep: setStep });

    createEffect(
        on(
            () => props.open,
            (open) => {
                // Drop messages and pending checks left over from a previous
                // open — a debounce may still be waiting to fire.
                stepCheck.clear();
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
                    // Always collect a fresh key, never the stored one.
                    private_key_saved: false,
                });
                setErrors({});
                setSaving(false);
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

    const setFieldError = (field: string, message?: string) => {
        setErrors((prev) => {
            const next = { ...prev };
            if (message) {
                next[field] = message;
            } else {
                delete next[field];
            }
            return next;
        });
    };

    const error = (field: string) => errors()[field];

    /**
     * Clears the given field errors and the inline step message before a value
     * change.  Must run *before* the store update: on step 3 the pre-flight
     * effect reschedules the debounced check when the values change, so
     * clearing afterwards would cancel the freshly scheduled check.
     */
    const resetValidation = (...fields: string[]) => {
        fields.forEach(clearError);
        stepCheck.clear();
    };

    // ----- Steps 1 & 2: certificate and private key -----

    const certFields = createPemFields(CERT_CONFIG, {
        values,
        setValues,
        resetValidation,
        validate: validateCertFields,
        setErrors,
        scheduleCheck: () => stepCheck.schedule(1),
        clearMessage: stepCheck.clear,
    });

    const keyFields = createPemFields(KEY_CONFIG, {
        values,
        setValues,
        resetValidation,
        validate: validateKeyFields,
        setErrors,
        scheduleCheck: () => stepCheck.schedule(2),
        clearMessage: stepCheck.clear,
    });

    /** Validates the step, then advances once the backend lets us through. */
    const goToNextStep = async () => {
        const from = step();
        if (from === 3) return;

        const validate = from === 1 ? validateCertFields : validateKeyFields;
        stepCheck.cancel();
        const errs = validate(values);
        setErrors(errs);
        if (Object.values(errs).some(Boolean)) return;

        if (await stepCheck.requestAdvance(from)) {
            setStep((from + 1) as WizardStep);
        }
    };

    // ----- Step 3: config & enable -----

    /**
     * Client gate of the config step.  The server name is optional — an empty
     * one only disables DDR, ClientID detection and the certificate hostname
     * check on the backend, so it must not block.
     */
    const clientErrors = (): Record<string, string> =>
        validateEncryptionForm(values, getExternalPorts());

    const handleConfigFieldChange = (field: ServerSettingsField, raw: string) => {
        resetValidation(field);
        if (field === 'server_name') {
            setValues('server_name', raw);
            return;
        }
        setValues(field, toNumber(raw));
    };

    const handleConfigFieldBlur = (field: ServerSettingsField) => {
        if (field === 'server_name') {
            const normalized = normalizeServerName(String(values.server_name ?? ''));
            setValues('server_name', normalized);
            setFieldError(field, validateServerName(normalized));
            return;
        }

        const port = Number(values[field]) || 0;
        setFieldError(field, validatePortField(field, port));
    };

    /**
     * Blur and backend errors win; the live port conflicts fill the gaps.  The
     * server name has no live rule — it is validated on blur, so typing does
     * not flash an error mid-word.
     */
    const configFieldError = (field: ServerSettingsField) =>
        error(field) ??
        stepCheck.fieldError(field) ??
        (field === 'server_name' ? undefined : clientErrors()[field]);

    // Pre-flight for step 3: debounce a check with the exact final-save
    // payload whenever the config values change and the form is clean, so port
    // probes and the plain-DNS rule see what "Enable" will send.
    createEffect(() => {
        if (step() !== 3) return;
        if (Object.values(clientErrors()).some(Boolean)) return;
        stepCheck.schedule(3);
    });

    const handleGoBack = () => {
        setErrors({});
        // Messages may travel forward (a step-1 warning stays visible on the
        // later steps) but never backwards, so drop them here.
        stepCheck.clear();
        setStep((prev) => (prev > 1 ? ((prev - 1) as WizardStep) : prev));
    };

    const handleEnable = async () => {
        stepCheck.cancel();
        const errs = clientErrors();
        if (Object.values(errs).some(Boolean)) {
            setErrors(errs);
            return;
        }

        // The pending state covers the whole submit — the backend check and
        // the save — so the button never looks idle while it is working.
        setSaving(true);
        try {
            if (!(await stepCheck.requestAdvance(3))) return;

            const res = await setTlsConfig(
                getSubmitValues({ ...values, enabled: true, serve_plain_dns: true }),
                { suppressErrorToast: true },
            );

            if (!res.ok) {
                stepCheck.fail(intl.getMessage('tls_setup_error_enable_failed'));
                return;
            }

            props.onClose();
        } finally {
            setSaving(false);
        }
    };

    const enableDisabled = () =>
        saving() ||
        stepCheck.validating() ||
        Object.values(clientErrors()).some(Boolean) ||
        stepCheck.blocked();

    const titles: readonly (() => string)[] = [
        () => intl.getMessage('tls_setup_cert_title'),
        () => intl.getMessage('tls_setup_key_title'),
        () => intl.getMessage('tls_setup_config_title'),
    ];
    const descriptions: readonly (() => JSX.Element | undefined)[] = [
        () =>
            intl.getMessage('tls_setup_cert_description', {
                a: (text: string) => (
                    <a
                        href={LETSENCRYPT_LINK}
                        target="_blank"
                        rel="noopener noreferrer"
                        class={cn(theme.link.link, theme.link.hoverDecoration)}
                        data-testid="tls-setup-letsencrypt-link"
                    >
                        {text}
                    </a>
                ),
            }),
        () => intl.getMessage('tls_setup_key_description'),
        () => undefined,
    ];

    const title = () => titles[step() - 1]();
    const description = () => descriptions[step() - 1]();
    const submitVariant = () => (stepCheck.hasWarning() ? 'warning' : 'primary');

    const footer = () =>
        step() === 3 ? (
            <div class={s.footer}>
                <div class={s.footerButtons}>
                    <Button
                        variant={submitVariant()}
                        onClick={handleEnable}
                        disabled={enableDisabled()}
                        data-testid="tls-setup-enable"
                        rightAddon={saving() ? <InlineLoader /> : undefined}
                    >
                        {stepCheck.hasWarning()
                            ? intl.getMessage('tls_setup_enable_anyway')
                            : intl.getMessage('enable')}
                    </Button>
                    <Button variant="secondary" onClick={props.onClose}>
                        {intl.getMessage('cancel')}
                    </Button>
                </div>
                <StepFormMessage message={stepCheck.formMessage()} class={s.footerMessage} />
            </div>
        ) : (
            <div class={s.footerButtons}>
                <Button
                    variant={submitVariant()}
                    onClick={() => void goToNextStep()}
                    disabled={stepCheck.validating()}
                    data-testid="tls-setup-add"
                >
                    {stepCheck.hasWarning()
                        ? intl.getMessage('tls_setup_add_anyway')
                        : intl.getMessage('add')}
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
            processing={saving()}
            hideSubmit
            footer={footer()}
        >
            <WizardSteps step={step()} onGoBack={handleGoBack} />
            <h2 class={cn(theme.title.h4, s.wizardTitle)} data-testid="tls-setup-title">
                {title()}
            </h2>
            <Show when={description()}>
                <div class={cn(theme.text.t2, s.wizardDescription)}>{description()}</div>
            </Show>

            <Show when={step() === 1}>
                <div class={s.content} data-testid="tls-setup-step-certificate">
                    <PemSourceFields
                        config={CERT_CONFIG}
                        fields={certFields}
                        errorFor={(field) => error(field) ?? stepCheck.fieldError(field)}
                        warningFor={stepCheck.fieldWarning}
                    />
                    <StepFormMessage message={stepCheck.formMessage()} />
                </div>
            </Show>

            <Show when={step() === 2}>
                <div class={s.content} data-testid="tls-setup-step-key">
                    <PemSourceFields
                        config={KEY_CONFIG}
                        fields={keyFields}
                        errorFor={(field) => error(field) ?? stepCheck.fieldError(field)}
                        warningFor={stepCheck.fieldWarning}
                    />
                    <StepFormMessage message={stepCheck.formMessage()} />
                </div>
            </Show>

            <Show when={step() === 3}>
                <div class={s.content} data-testid="tls-setup-step-config">
                    <ServerSettingsFields
                        idPrefix="tls_setup_"
                        values={values}
                        clearable
                        onFieldChange={handleConfigFieldChange}
                        onFieldBlur={handleConfigFieldBlur}
                        errorFor={configFieldError}
                        warningFor={stepCheck.fieldWarning}
                        testIdPrefix="tls-setup"
                    />
                </div>
            </Show>
        </ConfigDialog>
    );
};
