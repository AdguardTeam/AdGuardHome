import { createMemo } from 'solid-js';
import { statsState } from './stats';
import { dashboardState } from './dashboard';

/**
 * Derived state computations using createMemo.
 * These replace reselect-style selectors from the Redux setup.
 */

export const blockedPercentage = createMemo(() => {
    const { numDnsQueries, numBlockedFiltering } = statsState;
    if (!numDnsQueries) return 0;
    return Math.round((numBlockedFiltering / numDnsQueries) * 100);
});

export const parentalPercentage = createMemo(() => {
    const { numDnsQueries, numReplacedParental } = statsState;
    if (!numDnsQueries) return 0;
    return Math.round((numReplacedParental / numDnsQueries) * 100);
});

export const safebrowsingPercentage = createMemo(() => {
    const { numDnsQueries, numReplacedSafebrowsing } = statsState;
    if (!numDnsQueries) return 0;
    return Math.round((numReplacedSafebrowsing / numDnsQueries) * 100);
});

export const safesearchPercentage = createMemo(() => {
    const { numDnsQueries, numReplacedSafesearch } = statsState;
    if (!numDnsQueries) return 0;
    return Math.round((numReplacedSafesearch / numDnsQueries) * 100);
});

export const isProtectionActive = createMemo(() => {
    return dashboardState.protectionEnabled && dashboardState.isCoreRunning;
});
