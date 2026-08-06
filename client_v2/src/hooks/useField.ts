import { createSignal, createEffect, type Accessor, type Setter } from 'solid-js';

/**
 * Configuration for useField hook.
 */
type UseFieldConfig<T> = {
    /**
     * Optional validator function.
     * Should return an error message string on failure, or empty string on success.
     */
    validate?: (value: T) => string;
};

/**
 * Return type of the useField hook.
 */
type UseFieldResult<T> = {
    /** Current field value (reactive accessor). */
    value: Accessor<T>;
    /** Update the field value. */
    setValue: Setter<T>;
    /** Current validation error message (empty string when valid). */
    error: Accessor<string>;
    /** Manually override the error message. */
    setError: Setter<string>;
    /**
     * Run the validator, set the error signal, and return the error message.
     * Returns empty string if valid.
     */
    validate: () => string;
    /**
     * Validate then submit. Runs the validator; if no error, calls onSubmit
     * with the current value. Returns true if submitted, false if validation failed.
     */
    submitIfValid: (onSubmit: (value: T) => void) => boolean;
};

/**
 * Composable hook that encapsulates the 4-signal + sync-effect pattern
 * used by DNS settings dialogs.
 *
 * When `isOpen` returns true, the field value resets from `getStoreValue()`
 * and the error is cleared — mirroring the original createEffect behavior.
 *
 * @param isOpen  Getter that returns whether the dialog is open.
 *                Called inside createEffect for proper reactivity tracking.
 * @param getStoreValue  Lazily reads the store value on each effect re-run.
 * @param config  Optional configuration (validator).
 */
export function useField<T>(
    isOpen: () => boolean,
    getStoreValue: () => T,
    config?: UseFieldConfig<T>,
): UseFieldResult<T> {
    const [value, setValue] = createSignal<T>(getStoreValue());
    const [error, setError] = createSignal('');

    // Sync from store when dialog opens — mirrors the original pattern:
    //   createEffect(() => { if (open()) { setValue(storeValue); setError(''); } });
    createEffect(() => {
        if (isOpen()) {
            setValue(() => getStoreValue());
            setError('');
        }
    });

    const validate = (): string => {
        const err = config?.validate ? config.validate(value()) : '';
        setError(err);
        return err;
    };

    const submitIfValid = (onSubmit: (v: T) => void): boolean => {
        const err = validate();
        if (err) {
            return false;
        }
        onSubmit(value());
        return true;
    };

    return {
        value,
        setValue,
        error,
        setError,
        validate,
        submitIfValid,
    };
}
