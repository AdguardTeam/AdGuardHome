import { type Accessor, createEffect, createSignal } from 'solid-js';

/**
 * Return type of the useSwitchState hook: the value to render and the setter
 * used to correct it.
 */
type UseSwitchStateResult = [Accessor<boolean>, (next: boolean) => void];

/**
 * Mirrors a store-backed switch into a local signal so the rendered state can
 * be corrected even when the store value did not change.
 *
 * A native checkbox flips itself the moment it is clicked, while the store only
 * changes after a successful save — so without this, a rejected toggle leaves
 * the switch looking enabled and every further click re-sends the same failing
 * value.  The signal is created with `equals: false` to force that correction
 * through, and it re-reads the store only while `processing` is false so the
 * switch does not flicker back mid-save.
 *
 * @param value  Getter for the store-backed value.
 * @param processing  Getter for the in-flight flag of the save request.
 */
export function useSwitchState(
    value: Accessor<boolean>,
    processing: Accessor<boolean>,
): UseSwitchStateResult {
    const [checked, setChecked] = createSignal(!!value(), { equals: false });

    createEffect(() => {
        if (!processing()) {
            setChecked(!!value());
        }
    });

    return [checked, setChecked];
}
