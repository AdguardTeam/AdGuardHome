import { describe, it, expect } from 'vitest';

import { formatHistoryLabel } from 'panel/helpers/lineUtils';
import { TIME_UNITS } from 'panel/helpers/constants';

// Use a local-time reference to make the tests timezone-independent.
const NOW = new Date(2026, 7, 24, 15, 0, 0); // Aug 24, 2026 15:00 local

describe('formatHistoryLabel', () => {
    it('formats hourly labels with date and hour', () => {
        // 24 hourly points: oldest is 23 hours ago.
        expect(formatHistoryLabel(0, 24, TIME_UNITS.HOURS, NOW)).toBe('23 Aug 16:00');
        // Newest point is the current hour.
        expect(formatHistoryLabel(23, 24, TIME_UNITS.HOURS, NOW)).toBe('24 Aug 15:00');
    });

    it('formats hourly labels for sub-day intervals', () => {
        // 6 hourly points (6h retention).
        expect(formatHistoryLabel(0, 6, TIME_UNITS.HOURS, NOW)).toBe('24 Aug 10:00');
        expect(formatHistoryLabel(5, 6, TIME_UNITS.HOURS, NOW)).toBe('24 Aug 15:00');
    });

    it('formats daily labels with date and year', () => {
        // 30 daily points: oldest is 29 days ago.
        expect(formatHistoryLabel(0, 30, TIME_UNITS.DAYS, NOW)).toBe('26 Jul 2026');
        // Newest point is today.
        expect(formatHistoryLabel(29, 30, TIME_UNITS.DAYS, NOW)).toBe('24 Aug 2026');
    });

    it('handles a single point', () => {
        expect(formatHistoryLabel(0, 1, TIME_UNITS.DAYS, NOW)).toBe('24 Aug 2026');
        expect(formatHistoryLabel(0, 1, TIME_UNITS.HOURS, NOW)).toBe('24 Aug 15:00');
    });
});
