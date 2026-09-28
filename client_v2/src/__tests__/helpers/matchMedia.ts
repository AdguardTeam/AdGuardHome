type Listener = (event: MediaQueryListEvent) => void;

/**
 * How a stubbed `window.matchMedia` should answer a query: a single boolean for
 * every query, or a predicate for per-query control.
 */
export type MatchMediaResult = boolean | ((query: string) => boolean);

export interface MatchMediaMock {
    /**
     * Replaces the current result and notifies every registered `change`
     * listener, which is how the reactive hooks observe a viewport change.
     */
    set: (next: MatchMediaResult) => void;
    /**
     * Number of live `change` listeners across the media queries whose text
     * contains `queryFragment` (all of them when it is omitted).  Use it to
     * assert that a hook unsubscribes on unmount.
     */
    listenerCount: (queryFragment?: string) => number;
}

/** Builds a predicate that emulates a viewport of `width` pixels wide. */
export const byViewportWidth =
    (width: number): MatchMediaResult =>
    (query) => {
        const minWidth = /\(min-width:\s*(\d+)px\)/.exec(query);

        return minWidth ? width >= Number(minWidth[1]) : false;
    };

/**
 * Installs a controllable `window.matchMedia` stub — jsdom does not implement
 * it.  Pass a boolean to make every query match (or not match), or a predicate
 * to answer per query, e.g. `byViewportWidth(900)`.
 *
 * The reactive hooks in `panel/hooks/useMediaQuery` read `matches` both when
 * they are created and again on mount, so the stub keeps the value live behind
 * a getter instead of copying it into each `MediaQueryList`.
 */
export const mockMatchMedia = (initial: MatchMediaResult = false): MatchMediaMock => {
    let current = initial;

    const created: { query: string; listeners: Set<Listener> }[] = [];

    const resolve = (query: string) =>
        typeof current === 'boolean' ? current : current(query);

    const createMediaQueryList = (query: string): MediaQueryList => {
        const record = { query, listeners: new Set<Listener>() };
        created.push(record);

        return {
            get matches() {
                return resolve(query);
            },
            media: query,
            onchange: null,
            addEventListener: (type: string, listener: Listener) => {
                if (type === 'change') record.listeners.add(listener);
            },
            removeEventListener: (type: string, listener: Listener) => {
                if (type === 'change') record.listeners.delete(listener);
            },
            addListener: (listener: Listener) => record.listeners.add(listener),
            removeListener: (listener: Listener) => record.listeners.delete(listener),
            dispatchEvent: () => false,
        } as unknown as MediaQueryList;
    };

    Object.defineProperty(window, 'matchMedia', {
        writable: true,
        configurable: true,
        value: createMediaQueryList,
    });

    return {
        set: (next) => {
            current = next;

            created.forEach(({ query, listeners }) => {
                const event = { matches: resolve(query), media: query } as MediaQueryListEvent;
                listeners.forEach((listener) => listener(event));
            });
        },
        listenerCount: (queryFragment) =>
            created
                .filter(({ query }) => !queryFragment || query.includes(queryFragment))
                .reduce((count, { listeners }) => count + listeners.size, 0),
    };
};
