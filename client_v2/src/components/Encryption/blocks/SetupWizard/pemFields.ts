import type { Setter } from 'solid-js';
import type { SetStoreFunction } from 'solid-js/store';

import intl from 'panel/common/intl';
import { ENCRYPTION_SOURCE } from 'panel/helpers/constants';
import type { EncryptionFormValues } from '../../validate';

/** Field names of one PEM source pair — the certificate or the private key. */
type PemFieldNames = {
    /** Selects which of the two inputs holds the PEM data. */
    source: 'certificate_source' | 'key_source';
    /** PEM data pasted or uploaded by the user, kept in the config file. */
    content: 'certificate_chain' | 'private_key';
    /** Path to the PEM file on the machine running AdGuard Home. */
    path: 'certificate_path' | 'private_key_path';
};

/**
 * Translated texts of a PEM step.  Accessors rather than plain strings: the
 * messages must be looked up while rendering so a language change is picked
 * up, and `intl.getMessage` keeps its literal key for the translation audit.
 */
type PemTexts = {
    textOption: () => string;
    textOptionDesc: () => string;
    pathOption: () => string;
    pathOptionDesc: () => string;
    contentLabel: () => string;
    pathLabel: () => string;
    dropzoneHint: () => string;
};

/** Everything the shared step-1/step-2 markup needs to render one PEM pair. */
export type PemStepConfig = {
    fields: PemFieldNames;
    /** Input ids are `idPrefix + <field name>`. */
    radioName: string;
    idPrefix: string;
    /** PEM header — not translated. */
    contentPlaceholder: string;
    testIdPrefix: string;
    texts: PemTexts;
};

/** Step 1 — the certificate chain. */
export const CERT_CONFIG: PemStepConfig = {
    fields: {
        source: 'certificate_source',
        content: 'certificate_chain',
        path: 'certificate_path',
    },
    radioName: 'tls_setup_certificate_source',
    idPrefix: 'tls_setup_',
    contentPlaceholder: '-----BEGIN CERTIFICATE-----',
    testIdPrefix: 'tls-setup-cert',
    texts: {
        textOption: () => intl.getMessage('tls_setup_cert_text_option'),
        textOptionDesc: () => intl.getMessage('tls_setup_cert_text_option_desc'),
        pathOption: () => intl.getMessage('tls_setup_cert_path_option'),
        pathOptionDesc: () => intl.getMessage('tls_setup_cert_path_option_desc'),
        contentLabel: () => intl.getMessage('tls_setup_cert_paste_label'),
        pathLabel: () => intl.getMessage('tls_cert_path_label'),
        dropzoneHint: () => intl.getMessage('encryption_cert_dropzone_hint'),
    },
};

/**
 * Step 2 — the private key.
 *
 * No "use the saved key" source: the key stored in the configuration file is
 * never returned by the API, and the wizard is only reachable while no
 * certificate is configured, so there is nothing to reuse here.
 */
export const KEY_CONFIG: PemStepConfig = {
    fields: {
        source: 'key_source',
        content: 'private_key',
        path: 'private_key_path',
    },
    radioName: 'tls_setup_key_source',
    idPrefix: 'tls_setup_',
    contentPlaceholder: '-----BEGIN PRIVATE KEY-----',
    testIdPrefix: 'tls-setup-key',
    texts: {
        textOption: () => intl.getMessage('tls_setup_key_text_option'),
        textOptionDesc: () => intl.getMessage('tls_setup_key_text_option_desc'),
        pathOption: () => intl.getMessage('tls_setup_key_path_option'),
        pathOptionDesc: () => intl.getMessage('tls_setup_key_path_option_desc'),
        contentLabel: () => intl.getMessage('tls_setup_key_paste_label'),
        pathLabel: () => intl.getMessage('tls_key_path_label'),
        dropzoneHint: () => intl.getMessage('encryption_key_dropzone_hint'),
    },
};

type Deps = {
    values: EncryptionFormValues;
    setValues: SetStoreFunction<EncryptionFormValues>;
    /**
     * Clears the given field errors and the inline step message.  The caller
     * owns the *order* rule: this must run before the store update, see
     * `resetValidation` in `TlsSetupWizard`.
     */
    resetValidation: (...fields: string[]) => void;
    /** Field validation of the step, e.g. `validateCertFields`. */
    validate: (values: EncryptionFormValues) => Record<string, string>;
    setErrors: Setter<Record<string, string>>;
    /** Schedules the debounced backend check for the step. */
    scheduleCheck: () => void;
    /** Drops the current step message, e.g. when a local error takes over. */
    clearMessage: () => void;
};

/**
 * Handlers of one PEM source pair, shared by wizard steps 1 and 2.
 * All values are read through accessors so the markup stays reactive.
 */
export const createPemFields = (config: PemStepConfig, deps: Deps) => {
    const { fields } = config;

    const sourceValue = () => deps.values[fields.source];
    const isContent = () => sourceValue() === ENCRYPTION_SOURCE.CONTENT;
    const contentValue = () => deps.values[fields.content] ?? '';
    const pathValue = () => deps.values[fields.path] ?? '';
    /** The value the current source points at. */
    const fieldValue = () => (isContent() ? contentValue() : pathValue());

    // Dropzone shown until PEM data is entered (Figma AGDNS-4286: the paste
    // field is 116px while the dropzone is visible, 200px once it is gone).
    const dropzoneVisible = () => !contentValue();

    const handleSourceChange = (v: string) => {
        deps.resetValidation(fields.content, fields.path);
        deps.setValues(fields.source, v);
    };

    const handleContentChange = (e: Event) => {
        deps.resetValidation(fields.content);
        deps.setValues(fields.content, (e.target as HTMLTextAreaElement).value);
    };

    const handlePathChange = (e: Event) => {
        deps.resetValidation(fields.path);
        deps.setValues(fields.path, (e.target as HTMLInputElement).value);
    };

    const handleFileSelect = (content: string) => {
        deps.resetValidation(fields.content, fields.path);
        deps.setValues(fields.content, content);
        deps.setValues(fields.source, ENCRYPTION_SOURCE.CONTENT);
    };

    const handleClear = () => {
        deps.resetValidation(fields.content);
        deps.setValues(fields.content, '');
    };

    const validateOnBlur = () => {
        // Blur may mean reaching for the dropzone / Browse, or the native file
        // dialog taking focus — an empty field is not an error yet.  The
        // required error belongs to Add.
        if (!String(fieldValue()).trim()) return;

        const errs = deps.validate(deps.values);
        deps.setErrors((prev) => ({ ...prev, ...errs }));
        if (Object.values(errs).some(Boolean)) {
            deps.clearMessage();
            return;
        }
        deps.scheduleCheck();
    };

    return {
        sourceValue,
        isContent,
        contentValue,
        pathValue,
        fieldValue,
        dropzoneVisible,
        handleSourceChange,
        handleContentChange,
        handlePathChange,
        handleFileSelect,
        handleClear,
        validateOnBlur,
    };
};

export type PemFields = ReturnType<typeof createPemFields>;
