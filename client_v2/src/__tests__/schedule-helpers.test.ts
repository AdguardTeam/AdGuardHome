import { describe, test, expect } from 'vitest';
import {
    msToTime,
    timeToMs,
    formatTimePeriod,
    isFullDay,
    FULL_DAY_END_MS,
    DAYS_OF_WEEK,
} from '../components/BlockedServices/InactivitySchedule/helpers';

describe('schedule helpers', () => {
    describe('msToTime', () => {
        test('converts 0ms to 00:00', () => {
            expect(msToTime(0)).toEqual({ hours: 0, minutes: 0 });
        });

        test('converts 3600000ms to 01:00', () => {
            expect(msToTime(3600000)).toEqual({ hours: 1, minutes: 0 });
        });

        test('converts 86340000ms to 23:59', () => {
            expect(msToTime(86340000)).toEqual({ hours: 23, minutes: 59 });
        });

        test('converts 45060000ms to 12:31', () => {
            expect(msToTime(45060000)).toEqual({ hours: 12, minutes: 31 });
        });
    });

    describe('timeToMs', () => {
        test('converts 00:00 to 0ms', () => {
            expect(timeToMs(0, 0)).toBe(0);
        });

        test('converts 01:00 to 3600000ms', () => {
            expect(timeToMs(1, 0)).toBe(3600000);
        });

        test('converts 23:59 to 86340000ms', () => {
            expect(timeToMs(23, 59)).toBe(86340000);
        });
    });

    describe('formatTimePeriod', () => {
        test('formats start and end as HH:MM – HH:MM', () => {
            expect(formatTimePeriod(3600000, 64800000)).toBe('01:00 \u2013 18:00');
        });

        test('pads single digits', () => {
            expect(formatTimePeriod(0, 3660000)).toBe('00:00 \u2013 01:01');
        });
    });

    describe('isFullDay', () => {
        test('returns true for start=0, end=86340000', () => {
            expect(isFullDay(0, 86340000)).toBe(true);
        });

        test('returns false for partial day', () => {
            expect(isFullDay(0, 3600000)).toBe(false);
        });
    });

    describe('FULL_DAY_END_MS', () => {
        test('equals 86340000', () => {
            expect(FULL_DAY_END_MS).toBe(86340000);
        });
    });

    describe('DAYS_OF_WEEK', () => {
        test('has 7 days in correct order', () => {
            expect(DAYS_OF_WEEK).toEqual(['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun']);
        });
    });
});
