import { createSignal } from 'solid-js';

import { encryptionState, validateTlsConfig } from 'panel/stores/encryption';
import type { ServerSettingsValues } from '../validate';
import { getStepValidationValues, getStoreFormValues } from './helpers';
import { mapStepResult } from './SetupWizard/mapStepResult';

type Options = {
    /** Live form values of the settings dialog; read when a check runs. */
    values: ServerSettingsValues;
};

/**
 * Certificate/server-name check behind the "Encrypted DNS server settings"
 * dialog.
 *
 * The dialog edits the name the certificate is checked against, so only the
 * backend can judge it: a check sends the edited draft together with the saved
 * certificate and key, exactly like the wizard's config step, and keeps the
 * mismatch warning it reports.  The warning, the live name it belongs to and
 * the check bookkeeping all live here, so the dialog only wires field events
 * to it.
 */
export const createServerNameCheck = (opts: Options) => {
    /**
     * Warning plus the name it was reported for.  The live name moves on every
     * keystroke while the store only learns it on `change` (blur), so the pair
     * is what keeps the warning from outliving the value it describes.
     */
    const [warning, setWarning] = createSignal<{ name: string; text: string }>();
    const [draftName, setDraftName] = createSignal('');

    // Monotonic id used to drop superseded validation responses.
    let checkId = 0;
    // Name of the last completed check, so re-leaving the field without
    // editing the name does not fire another request.
    let lastCheckedName: string | undefined;

    /** The warning to render — only while the field still holds the checked name. */
    const warningText = () => (warning()?.name === draftName() ? warning()?.text : undefined);

    const hasCert = () =>
        !!(encryptionState.certificate_chain || encryptionState.certificate_path);
    const hasKey = () =>
        !!(
            encryptionState.private_key ||
            encryptionState.private_key_path ||
            encryptionState.private_key_saved
        );

    /**
     * Asks the backend whether the certificate covers the edited server name.
     *
     * `current` is false when a newer check or an edit made the response stale;
     * the caller must then leave the decision to that newer check.  `warning`
     * is the text to show, or undefined when there is nothing to warn about —
     * no certificate to check the name against, a name the certificate covers,
     * or a backend verdict other than the server-name mismatch (an unreadable
     * certificate, a busy port) that this dialog does not render.
     */
    const run = async (): Promise<{ current: boolean; warning?: string }> => {
        if (!hasCert() || !hasKey()) return { current: true };

        // Validate the edited draft against the saved certificate and key,
        // exactly as the wizard's config step does.
        const formValues = getStoreFormValues({ ...opts.values });
        const name = String(formValues.server_name ?? '');
        const payload = getStepValidationValues(3, formValues);

        const id = ++checkId;
        const res = await validateTlsConfig(payload, { persist: false });

        if (id !== checkId || name !== draftName()) {
            return { current: false };
        }

        // The wizard's config step owns this copy, so the same mapping keeps
        // the two surfaces saying the same thing.
        const message = mapStepResult(3, res, formValues);
        const text =
            message?.kind === 'warning' && message.field === 'server_name'
                ? message.message
                : undefined;
        setWarning(text ? { name, text } : undefined);
        lastCheckedName = name;

        return { current: true, warning: text };
    };

    /** Fills the check with the settings the dialog opened with. */
    const reset = (name: string) => {
        setWarning(undefined);
        setDraftName(name);
        lastCheckedName = undefined;
    };

    /** Keeps the live name in step with the field, so the warning can be dropped. */
    const onNameInput = (name: string) => setDraftName(name);

    /** Checks the name the field was left with, unless it was already checked. */
    const checkOnBlur = (name: string) => {
        if (name !== lastCheckedName) void run();
    };

    /**
     * Runs the check for the save and reports whether it may go through.
     *
     * A newly discovered warning has to be seen before it can be saved past: it
     * is put on screen and this attempt is held back, so the next one is
     * visibly a confirmation.  A warning already on screen does not ask again,
     * and a check superseded by a newer one leaves the decision to that check.
     */
    const confirmSave = async (): Promise<boolean> => {
        const shown = warningText();
        const { current, warning: text } = await run();
        if (!current) return false;

        return !text || text === shown;
    };

    return { warningText, reset, onNameInput, checkOnBlur, confirmSave };
};
