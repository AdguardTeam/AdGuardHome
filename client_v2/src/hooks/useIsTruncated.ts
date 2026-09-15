import { type Accessor, createEffect, createSignal, onCleanup } from 'solid-js';

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
): Accessor<boolean> {
    const [isTruncated, setIsTruncated] = createSignal(false);

    createEffect(() => {
        const element = getElement();
        if (!element) return;

        const measure = () => setIsTruncated(element.scrollWidth > element.clientWidth);

        // Measure once synchronously: the first value has to be right before
        // the next paint, and the observer only fires on later box changes.
        measure();

        const observer = new ResizeObserver(measure);
        observer.observe(element);

        onCleanup(() => observer.disconnect());
    });

    return isTruncated;
}
