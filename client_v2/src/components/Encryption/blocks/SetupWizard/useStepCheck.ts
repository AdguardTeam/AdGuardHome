import { createSignal, onCleanup, type Accessor } from 'solid-js';

import { DEBOUNCE_TIMEOUT } from 'panel/helpers/constants';
import { validateTlsConfig } from 'panel/stores/encryption';
import type { EncryptionFormValues } from '../../validate';
import { getStepValidationValues, type WizardStep } from '../helpers';
import { mapStepResult, STEP_FIELDS, type StepMessage } from './mapStepResult';

type Options = {
    /** Current step — decides which messages apply to what is on screen. */
    step: Accessor<WizardStep>;
    /** Live form values, read when a check runs. */
    values: EncryptionFormValues;
    /** Moves the wizard to another step, e.g. back to a broken field. */
    goToStep: (step: WizardStep) => void;
};

const STEP_ORDER: readonly WizardStep[] = [1, 2, 3];

/** The step whose form contains `field`, if any. */
const stepOfField = (field: string): WizardStep | undefined =>
    STEP_ORDER.find((step) => STEP_FIELDS[step].has(field));

/** True when both describe the same thing the user has already been shown. */
const sameMessage = (a?: StepMessage, b?: StepMessage) =>
    !!a && !!b && a.kind === b.kind && a.field === b.field && a.message === b.message;

/**
 * Per-step backend validation for the TLS setup wizard.
 *
 * The backend reports a single warning or error at a time, so the hook keeps
 * one inline message for the whole step plus the flags the dialog footer
 * reacts to.  Checks are debounced and superseded responses are dropped; see
 * `message` for which of them reach the screen.
 */
export const createStepCheck = (opts: Options) => {
    const [checked, setChecked] = createSignal<StepMessage>();
    const [validating, setValidating] = createSignal(false);

    // Debounce timer plus a monotonic id used to drop superseded responses.
    let timer: ReturnType<typeof setTimeout> | undefined;
    let checkId = 0;

    onCleanup(() => {
        if (timer) clearTimeout(timer);
    });

    /**
     * Drops the current message, the pending debounce and any in-flight check.
     * The `validating` reset matters: a superseded in-flight check returns
     * early without touching the flag, so leaving it set would keep Add/Enable
     * disabled forever.
     */
    const clear = () => {
        checkId += 1;
        if (timer) clearTimeout(timer);
        timer = undefined;
        setChecked(undefined);
        setValidating(false);
    };

    /**
     * The message to render right now, if any.
     *
     * An error always shows — the wizard takes the user to the step that owns
     * its field, so a certificate that regressed is reported where the
     * certificate is entered.  A warning shows only where it is actionable —
     * bound to a field of the current step.  Every step re-runs the certificate
     * checks on the backend, so without this the certificate diagnostics would
     * repeat on steps that cannot fix them.
     */
    const message = () => {
        const m = checked();
        if (!m) return undefined;
        if (m.kind === 'error') return m;

        return STEP_FIELDS[opts.step()].has(m.field ?? '') ? m : undefined;
    };

    /** Runs the backend check for `step`; returns the message it produced. */
    const run = async (step: WizardStep): Promise<StepMessage | undefined> => {
        const id = ++checkId;
        setValidating(true);
        const res = await validateTlsConfig(getStepValidationValues(step, opts.values), {
            persist: false,
        });
        if (id !== checkId) return undefined; // superseded by a newer check
        setValidating(false);
        const result = mapStepResult(step, res, opts.values);
        setChecked(result);

        // An error about a field on an earlier step asks the user to fix
        // something they cannot see: show it where it belongs instead of
        // blocking the current step with a hint.
        const owner = result?.kind === 'error' && result.field ? stepOfField(result.field) : undefined;
        if (owner !== undefined && owner < step) opts.goToStep(owner);

        return result;
    };

    /** Debounced `run`, used from blur handlers and the step-3 effect. */
    const schedule = (step: WizardStep) => {
        if (timer) clearTimeout(timer);
        timer = setTimeout(() => void run(step), DEBOUNCE_TIMEOUT);
    };

    /** Drops the pending debounce, keeping the current message. */
    const cancel = () => {
        if (timer) clearTimeout(timer);
        timer = undefined;
    };

    /**
     * Runs the check and reports whether the caller may go through.
     *
     * An error always holds back.  A warning does not block, but a *newly
     * discovered* one has to be seen first: if it is not on screen already,
     * this attempt only puts it there and the next one goes through.  Warnings
     * the step does not display are ignored, so the user is never asked for a
     * click that would show nothing.
     */
    const requestAdvance = async (step: WizardStep): Promise<boolean> => {
        const shown = message();
        const result = await run(step);

        if (!result) return true;
        if (result.kind === 'error') return false;
        if (!message()) return true; // produced, but not shown on this step

        return sameMessage(shown, result);
    };

    /** Blocks the step with a form-level error, e.g. a failed save. */
    const fail = (text: string) => setChecked({ kind: 'error', message: text });

    /** True while the step carries a blocking error. */
    const blocked = () => message()?.kind === 'error';

    /**
     * True while the step shows a warning the user may still go past.  The
     * footer flips its submit button into the warning state on this, so the
     * click that goes through is visibly a confirmation.
     */
    const hasWarning = () => message()?.kind === 'warning';

    /** Error text to render under `field` on the current step, if any. */
    const fieldError = (field: string) => {
        const m = message();
        return m?.field === field && m.kind === 'error' ? m.message : undefined;
    };

    /** Warning text to render under `field` on the current step, if any. */
    const fieldWarning = (field: string) => {
        const m = message();
        return m?.field === field && m.kind === 'warning' ? m.message : undefined;
    };

    /** Message that is not bound to a visible field on the current step. */
    const formMessage = () => {
        const m = message();
        return m && STEP_FIELDS[opts.step()].has(m.field ?? '') ? undefined : m;
    };

    return {
        validating,
        clear,
        schedule,
        cancel,
        requestAdvance,
        fail,
        blocked,
        hasWarning,
        fieldError,
        fieldWarning,
        formMessage,
    };
};
