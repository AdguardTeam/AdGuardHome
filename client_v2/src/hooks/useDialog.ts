import { createSignal, type Accessor } from 'solid-js';

/**
 * Return type of the useDialog hook.
 */
type UseDialogResult = {
    /** Reactive accessor for the dialog open state. */
    open: Accessor<boolean>;
    /** Opens the dialog. */
    openDialog: () => void;
    /** Closes the dialog. */
    closeDialog: () => void;
};

/**
 * Convenience hook for dialog open/close state.
 * Returns `open` signal, `openDialog`, and `closeDialog` helpers.
 */
export function useDialog(): UseDialogResult {
    const [open, setOpen] = createSignal(false);

    return {
        open,
        openDialog: () => setOpen(true),
        closeDialog: () => setOpen(false),
    };
}
