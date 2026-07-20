import { createEffect, onCleanup } from 'solid-js';

/**
 * Calls `dismiss` when a `pointerdown` fires outside every one of the
 * given `refs` (the "safe zone").  Only activates when `isActive()` is true.
 *
 * Uses capture phase so it can intercept the event before
 * `stopPropagation` calls from descendants.
 */
export function useOutsideDismiss(
    isActive: () => boolean,
    dismiss: () => void,
    ...refs: Array<HTMLElement | undefined>
) {
    createEffect(() => {
        if (!isActive()) return;

        const onPointerDown = (e: PointerEvent) => {
            const target = e.target as Element;
            if (refs.some((r) => r?.contains(target))) return;
            dismiss();
        };

        document.addEventListener('pointerdown', onPointerDown, true);
        onCleanup(() => document.removeEventListener('pointerdown', onPointerDown, true));
    });
}
