import { type Accessor, createEffect, createSignal, onCleanup } from 'solid-js';

/**
 * Slack for the scroll position comparison.  Layout values are fractional
 * (device pixel ratios, zoom), so an exact comparison would keep reporting
 * "there is more to scroll" when only a sub-pixel remainder is left.
 */
const EDGE_TOLERANCE = 1;

/**
 * Reports which directions a scroll container can still be scrolled in — that
 * is, whether content is currently hidden past its top or bottom edge.  Used to
 * show edge shadows only while there is something to reveal in that direction.
 *
 * Both values read `false` when the content fits, so a container that does not
 * scroll never gets an edge shadow.  The element must be the scroll container
 * itself (`overflow-y: auto`); an element inside a `display: none` subtree
 * reports all metrics as 0, which reads as "cannot scroll"; the observer
 * re-measures once the subtree becomes visible.
 */
export function useScrollEdges(
    getElement: Accessor<HTMLElement | undefined>,
): { canScrollUp: Accessor<boolean>; canScrollDown: Accessor<boolean> } {
    const [canScrollUp, setCanScrollUp] = createSignal(false);
    const [canScrollDown, setCanScrollDown] = createSignal(false);

    createEffect(() => {
        const element = getElement();
        if (!element) return;

        const measure = () => {
            const { scrollTop, scrollHeight, clientHeight } = element;

            setCanScrollUp(scrollTop > EDGE_TOLERANCE);
            setCanScrollDown(scrollTop + clientHeight < scrollHeight - EDGE_TOLERANCE);
        };

        // Measure once synchronously: the first value has to be right before
        // the next paint, and the observers only fire on later changes.
        measure();

        element.addEventListener('scroll', measure, { passive: true });

        // Content changes (a different entry, a section appearing, a late font
        // swap) alter the scroll range without resizing the container itself,
        // so the children are observed as well.
        const observer = new ResizeObserver(measure);
        const observeContent = () => {
            observer.disconnect();
            observer.observe(element);
            Array.from(element.children).forEach((child) => observer.observe(child));
        };
        observeContent();

        const mutations = new MutationObserver(observeContent);
        mutations.observe(element, { childList: true });

        onCleanup(() => {
            element.removeEventListener('scroll', measure);
            mutations.disconnect();
            observer.disconnect();
        });
    });

    return { canScrollUp, canScrollDown };
}
