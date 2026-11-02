/**
 * The days of the week in the order the schedule UI renders them.
 *
 * Shared by the schedule components and the client form store so the two
 * cannot drift apart: a day missing from either list is silently dropped when
 * a client is loaded or saved.
 */
export const DAYS_OF_WEEK = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const;

export type DayKey = (typeof DAYS_OF_WEEK)[number];
