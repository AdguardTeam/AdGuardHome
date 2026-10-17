import { unwrap } from 'solid-js/store';

/**
 * Writes fields into a Solid store between test cases.
 *
 * Solid's store proxy ignores direct writes — `state.flag = false` only logs a
 * "Cannot mutate a Store directly" warning and leaves the value untouched — so
 * a reset must reach the underlying object through `unwrap`.  The write is not
 * reactive, which is what arranging a store before a test needs; the store's
 * own setter still notifies every later change.
 */
export const resetStore = <T extends object>(store: T, values: Partial<T>): void => {
    Object.assign(unwrap(store) as T, values);
};
