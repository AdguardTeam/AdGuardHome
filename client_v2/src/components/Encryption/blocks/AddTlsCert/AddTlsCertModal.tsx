import { createEffect, createSignal, on, Show } from 'solid-js';
import { ConfigDialog } from 'panel/common/ui/ConfigDialog';
import { Button } from 'panel/common/ui/Button';
import { Input } from 'panel/common/controls/Input';
import { Radio } from 'panel/common/controls/Radio';
import { Textarea } from 'panel/common/controls/Textarea';
import intl from 'panel/common/intl';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import { encryptionState, setTlsConfig, resetValidationStatus } from 'panel/stores/encryption';
import {
    validateCertFields,
    validateCertKeyFields,
    type EncryptionFormValues,
} from '../../validate';
import { getSubmitValues } from '../helpers';
import { FileBrowseButton } from './FileBrowseButton';
import s from './styles.module.pcss';
import theme from 'panel/lib/theme';

type FieldName = 'certificate_chain' | 'certificate_path' | 'private_key' | 'private_key_path';

type Props = {
    open: boolean;
    onClose: () => void;
};

export const AddTlsCertModal = (props: Props) => {
    const [step, setStep] = createSignal(1);
    const [certSource, setCertSource] = createSignal(ENCRYPTION_SOURCE.PATH);
    const [certChain, setCertChain] = createSignal('');
    const [certPath, setCertPath] = createSignal('');
    const [keySource, setKeySource] = createSignal(ENCRYPTION_SOURCE.PATH);
    const [privateKey, setPrivateKey] = createSignal('');
    const [privateKeyPath, setPrivateKeyPath] = createSignal('');
    const [privateKeySaved, setPrivateKeySaved] = createSignal(false);
    const [errors, setErrors] = createSignal<Partial<Record<FieldName, string>>>({});

    createEffect(
        on(
            () => props.open,
            (open) => {
                if (open) {
                    setStep(1);
                    setCertSource(ENCRYPTION_SOURCE.PATH);
                    setCertChain('');
                    setCertPath('');
                    setKeySource(ENCRYPTION_SOURCE.PATH);
                    setPrivateKey('');
                    setPrivateKeyPath('');
                    setPrivateKeySaved(false);
                    setErrors({});
                }
            },
        ),
    );

    const clearError = (field: FieldName) => {
        setErrors((prev) => {
            if (!(field in prev)) return prev;
            const next = { ...prev };
            delete next[field];
            return next;
        });
    };

    const error = (field: FieldName) => errors()[field];

    const buildValues = (): EncryptionFormValues => ({
        enabled: encryptionState.enabled,
        serve_plain_dns: encryptionState.serve_plain_dns,
        server_name: encryptionState.server_name,
        port_https: encryptionState.port_https || 0,
        port_dns_over_tls: encryptionState.port_dns_over_tls || 0,
        port_dns_over_quic: encryptionState.port_dns_over_quic || 0,
        certificate_chain: certChain(),
        private_key: privateKey(),
        certificate_path: certPath(),
        private_key_path: privateKeyPath(),
        certificate_source: certSource(),
        key_source: keySource(),
        private_key_saved: privateKeySaved(),
    });

    const certSourceOptions = [
        {
            text: intl.getMessage('tls_cert_path_option'),
            value: ENCRYPTION_SOURCE.PATH,
        },
        {
            text: intl.getMessage('encryption_certificates_source_content'),
            value: ENCRYPTION_SOURCE.CONTENT,
        },
    ];

    const keySourceOptions = () => [
        {
            text: intl.getMessage('tls_key_path_option'),
            value: ENCRYPTION_SOURCE.PATH,
        },
        {
            text: intl.getMessage('encryption_key_source_content'),
            value: ENCRYPTION_SOURCE.CONTENT,
        },
        {
            text: intl.getMessage('use_saved_key'),
            value: ENCRYPTION_SOURCE.SAVED,
            disabled: !encryptionState.private_key_saved,
        },
    ];

    const handleNext = () => {
        const values = buildValues();
        const errs = validateCertFields(values);
        if (Object.values(errs).some(Boolean)) {
            setErrors(errs);
            return;
        }
        setStep(2);
    };

    const handleAdd = () => {
        const values = buildValues();
        const errs = validateCertKeyFields(values);
        if (Object.values(errs).some(Boolean)) {
            setErrors(errs);
            return;
        }
        setTlsConfig(getSubmitValues(values));
        props.onClose();
    };

    const handleBack = () => {
        setStep(1);
        setErrors({});
    };

    const processing = () => encryptionState.processingConfig || encryptionState.processingValidate;
    const hasErrors = () => Object.values(errors()).some(Boolean);

    const isCertStep = () => step() === 1;
    const isKeyStep = () => step() === 2;

    const handleCertFileSelect = (content: string) => {
        setCertChain(content);
        setCertSource(ENCRYPTION_SOURCE.CONTENT);
        clearError('certificate_chain');
        clearError('certificate_path');
    };

    const handleKeyFileSelect = (content: string) => {
        setPrivateKey(content);
        setKeySource(ENCRYPTION_SOURCE.CONTENT);
        setPrivateKeySaved(false);
        clearError('private_key');
        clearError('private_key_path');
    };

    const handleCertSourceChange = (v: string) => {
        setCertSource(v);
        clearError('certificate_chain');
        clearError('certificate_path');
    };

    const handleCertPathChange = (e: Event) => {
        setCertPath((e.target as HTMLInputElement).value);
        clearError('certificate_path');
        resetValidationStatus();
    };

    const validateCertOnBlur = () => {
        const values = buildValues();
        const errs = validateCertFields(values);
        setErrors((prev) => ({ ...prev, ...errs }));
    };

    const handleCertChainChange = (e: Event) => {
        setCertChain((e.target as HTMLTextAreaElement).value);
        clearError('certificate_chain');
        resetValidationStatus();
    };

    const handleKeySourceChange = (v: string) => {
        setKeySource(v);
        if (v === ENCRYPTION_SOURCE.SAVED) {
            setPrivateKey('');
            setPrivateKeySaved(true);
        } else {
            setPrivateKeySaved(false);
        }
        clearError('private_key');
        clearError('private_key_path');
        resetValidationStatus();
    };

    const handleKeyPathChange = (e: Event) => {
        setPrivateKeyPath((e.target as HTMLInputElement).value);
        clearError('private_key_path');
        resetValidationStatus();
    };

    const validateKeyOnBlur = () => {
        const values = buildValues();
        const errs = validateCertKeyFields(values);
        setErrors((prev) => ({ ...prev, ...errs }));
    };

    const handleKeyChange = (e: Event) => {
        setPrivateKey((e.target as HTMLTextAreaElement).value);
        clearError('private_key');
        resetValidationStatus();
    };

    const CertStepFooter = (
        <div class={s.footer}>
            <Button variant="primary" onClick={handleNext} disabled={processing()}>
                {intl.getMessage('next')}
            </Button>
        </div>
    );

    const KeyStepFooter = (
        <div class={s.footer}>
            <Button variant="primary" onClick={handleAdd} disabled={processing() || hasErrors()}>
                {intl.getMessage('add')}
            </Button>
            <Button variant="secondary" onClick={handleBack}>
                {intl.getMessage('back')}
            </Button>
        </div>
    );

    return (
        <ConfigDialog
            open={props.open}
            title={
                isCertStep()
                    ? intl.getMessage('add_tls_certificate')
                    : intl.getMessage('add_tls_certificate_private_key')
            }
            description={intl.getMessage('tls_cert_modal_description')}
            onClose={props.onClose}
            onSubmit={handleAdd}
            processing={processing()}
            submitDisabled={hasErrors()}
            hideSubmit
            footer={isCertStep() ? CertStepFooter : KeyStepFooter}
        >
            <Show when={isCertStep()}>
                <div>
                    <Radio
                        value={certSource()}
                        handleChange={handleCertSourceChange}
                        name="certificate_source"
                        options={certSourceOptions}
                        inModal
                    />
                    <Show
                        when={certSource() === ENCRYPTION_SOURCE.CONTENT}
                        fallback={
                            <div class={theme.form.input}>
                                <Input
                                    id="certificate_path"
                                    name="certificate_path"
                                    value={certPath()}
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
                                id="certificate_chain"
                                name="certificate_chain"
                                value={certChain()}
                                onChange={handleCertChainChange}
                                onBlur={validateCertOnBlur}
                                placeholder={intl.getMessage('encryption_certificates_input')}
                                errorMessage={error('certificate_chain')}
                                size="large"
                            />
                        </div>
                    </Show>
                </div>
            </Show>

            <Show when={isKeyStep()}>
                <div>
                    <Radio
                        value={keySource()}
                        handleChange={handleKeySourceChange}
                        name="key_source"
                        options={keySourceOptions()}
                        inModal
                    />
                    <Show
                        when={keySource() === ENCRYPTION_SOURCE.CONTENT}
                        fallback={
                            <div class={theme.form.input}>
                                <Input
                                    id="private_key_path"
                                    name="private_key_path"
                                    value={privateKeyPath()}
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
                                id="private_key"
                                name="private_key"
                                value={privateKey()}
                                onChange={handleKeyChange}
                                onBlur={validateKeyOnBlur}
                                placeholder={intl.getMessage('encryption_key_input')}
                                errorMessage={error('private_key')}
                                disabled={privateKeySaved()}
                                size="large"
                            />
                        </div>
                    </Show>
                </div>
            </Show>
        </ConfigDialog>
    );
};
