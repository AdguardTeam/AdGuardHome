import { type Accessor, createEffect, createSignal, onCleanup } from 'solid-js';

type Options = {
    /**
     * While it returns false the hook neither measures nor observes — for
     * callers whose tooltip can never open, e.g. on touch devices.  The signal
     * resets to `false` so a stale value cannot leak into the disabled state.
     */
    enabled?: Accessor<boolean>;
    /**
     * Re-measures when this changes.  The observer only fires on box changes,
     * so text swapped inside an unchanged box would keep a stale value.
     */
    source?: Accessor<unknown>;
};

/**
 * Reports whether an element's content is clipped by its own box — that is,
 * whether the browser is currently rendering a `text-overflow: ellipsis`.
 *
 * The element must clip its own overflow (`overflow: hidden`) for the
 * measurement to mean anything.  An element inside a `display: none` subtree
 * reports both dimensions as 0, which reads as "not truncated"; the observer
 * re-measures once the subtree becomes visible.
 */
export function useIsTruncated(
    getElement: Accessor<HTMLElement | undefined>,
    options?: Options,
): Accessor<boolean> {
    const [isTruncated, setIsTruncated] = createSignal(false);

    const isEnabled = () => options?.enabled?.() ?? true;

    const measure = () => {
        const element = getElement();
        if (element) setIsTruncated(element.scrollWidth > element.clientWidth);
    };

    // The observer follows the element and the enabled flag only, so a source
    // change never re-creates it.
    createEffect(() => {
        if (!isEnabled()) {
            setIsTruncated(false);
            return;
        }

        const element = getElement();
        if (!element) return;

        const observer = new ResizeObserver(measure);
        observer.observe(element);

        onCleanup(() => observer.disconnect());
    });

    // Measured apart from the observer: a source change has to re-measure even
    // when the box keeps its size, which the observer alone never reports.
    createEffect(() => {
        // Read the accessor for its dependency, not for its value.
        options?.source?.();

        if (isEnabled()) measure();
    });

    return isTruncated;
}
