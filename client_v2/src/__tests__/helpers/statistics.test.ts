import { describe, it, expect, beforeEach } from 'vitest';

import { DAY, HOUR } from 'panel/helpers/constants';
import { LocalStorageHelper, LOCAL_STORAGE_KEYS } from 'panel/helpers/localStorageHelper';
import {
    clampStatsPeriod,
    computePercent,
    formatPercent,
    getEffectiveStatsPeriod,
    getStoredStatsPeriod,
    resolveStatsPeriod,
    sumUpstreamResponses,
} from 'panel/helpers/statistics';

const storePeriod = (period: number) =>
    LocalStorageHelper.setItem(LOCAL_STORAGE_KEYS.STATS_PERIOD, period);

describe('clampStatsPeriod', () => {
    it('clamps a period above the max interval down to it', () => {
        expect(clampStatsPeriod(DAY * 30, DAY * 7)).toBe(DAY * 7);
    });

    it('keeps a period within the max interval as is', () => {
        expect(clampStatsPeriod(HOUR * 6, DAY * 7)).toBe(HOUR * 6);
    });

    it('falls back to DAY when the max interval is unexpectedly small', () => {
        expect(clampStatsPeriod(DAY * 30, 0)).toBe(DAY);
    });
});

describe('getStoredStatsPeriod', () => {
    beforeEach(() => localStorage.clear());

    it('defaults to DAY when nothing is stored', () => {
        expect(getStoredStatsPeriod()).toBe(DAY);
    });

    it('returns the stored period', () => {
        storePeriod(DAY * 30);
        expect(getStoredStatsPeriod()).toBe(DAY * 30);
    });

    it('ignores a non-positive stored value', () => {
        storePeriod(0);
        expect(getStoredStatsPeriod()).toBe(DAY);
    });
});

describe('getEffectiveStatsPeriod', () => {
    beforeEach(() => localStorage.clear());

    it('clamps the stored period by the max interval', () => {
        storePeriod(DAY * 90);
        expect(getEffectiveStatsPeriod(DAY * 7)).toBe(DAY * 7);
    });

    it('keeps the stored period when it is within the max interval', () => {
        storePeriod(DAY * 7);
        expect(getEffectiveStatsPeriod(DAY * 90)).toBe(DAY * 7);
    });
});

describe('resolveStatsPeriod', () => {
    beforeEach(() => localStorage.clear());

    it('prefers a valid URL period', () => {
        storePeriod(DAY);
        expect(resolveStatsPeriod({ period: String(HOUR * 6) }, DAY * 30)).toBe(HOUR * 6);
    });

    it('clamps a URL period above the max interval', () => {
        expect(resolveStatsPeriod({ period: String(DAY * 90) }, DAY * 30)).toBe(DAY * 30);
    });

    it('falls back to the stored period when the URL period is invalid', () => {
        storePeriod(DAY * 7);
        expect(resolveStatsPeriod({ period: 'nope' }, DAY * 30)).toBe(DAY * 7);
    });
});

describe('computePercent', () => {
    it('returns the part of the total as a percentage', () => {
        expect(computePercent(1, 4)).toBe(25);
    });

    it('returns 0 instead of NaN for an empty total', () => {
        expect(computePercent(5, 0)).toBe(0);
    });
});

describe('formatPercent', () => {
    it('prints whole numbers for zero and for large shares', () => {
        expect(formatPercent(0)).toBe('0');
        expect(formatPercent(10)).toBe('10');
        expect(formatPercent(99.6)).toBe('100');
        expect(formatPercent(100)).toBe('100');
    });

    it('keeps one decimal for small non-zero shares', () => {
        expect(formatPercent(8.5)).toBe('8.5');
        expect(formatPercent(9.99)).toBe('10.0');
    });
});

describe('sumUpstreamResponses', () => {
    it('adds the response counts', () => {
        expect(sumUpstreamResponses([{ count: 600 }, { count: 500 }, { count: 200 }])).toBe(
            1300,
        );
    });

    it('returns 0 for an empty list', () => {
        expect(sumUpstreamResponses([])).toBe(0);
    });
});
