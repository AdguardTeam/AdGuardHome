export const FULL_DAY_END_MS = 86340000;

export const DAYS_OF_WEEK = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const;

export type DayKey = typeof DAYS_OF_WEEK[number];

export interface TimeValue {
    hours: number;
    minutes: number;
}

export interface ScheduleDayData {
    start: number;
    end: number;
}

export interface ScheduleData {
    time_zone: string;
    mon?: ScheduleDayData;
    tue?: ScheduleDayData;
    wed?: ScheduleDayData;
    thu?: ScheduleDayData;
    fri?: ScheduleDayData;
    sat?: ScheduleDayData;
    sun?: ScheduleDayData;
}

export const msToTime = (ms: number): TimeValue => {
    const totalMinutes = Math.floor(ms / 60000);
    const hours = Math.floor(totalMinutes / 60);
    const minutes = totalMinutes % 60;
    return { hours, minutes };
};

export const timeToMs = (hours: number, minutes: number): number => {
    return (hours * 60 + minutes) * 60000;
};

export const padTime = (value: number): string => {
    return value.toString().padStart(2, '0');
};

export const formatTimePeriod = (startMs: number, endMs: number): string => {
    const start = msToTime(startMs);
    const end = msToTime(endMs);
    return `${padTime(start.hours)}:${padTime(start.minutes)} \u2013 ${padTime(end.hours)}:${padTime(end.minutes)}`;
};

export const isFullDay = (start: number, end: number): boolean => {
    return start === 0 && end === FULL_DAY_END_MS;
};

export const getLocalTimezone = (): string => {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
};
