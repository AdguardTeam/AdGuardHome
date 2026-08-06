export const FULL_DAY_END_MS = 86340000;

export const DAYS_OF_WEEK = ['mon', 'tue', 'wed', 'thu', 'fri', 'sat', 'sun'] as const;

export type DayKey = (typeof DAYS_OF_WEEK)[number];

export type TimeValue = {
    hours: number;
    minutes: number;
};

export type ScheduleTimeOption = {
    label: string;
    value: number;
    isDisabled?: boolean;
};

export type ScheduleDayData = {
    start: number;
    end: number;
};

export type ScheduleData = {
    time_zone: string;
    mon?: ScheduleDayData;
    tue?: ScheduleDayData;
    wed?: ScheduleDayData;
    thu?: ScheduleDayData;
    fri?: ScheduleDayData;
    sat?: ScheduleDayData;
    sun?: ScheduleDayData;
};

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

export const HOURS_OPTIONS: ScheduleTimeOption[] = Array.from({ length: 24 }, (_, i) => ({
    label: padTime(i),
    value: i,
}));

export const MINUTES_OPTIONS: ScheduleTimeOption[] = Array.from({ length: 60 }, (_, i) => ({
    label: padTime(i),
    value: i,
}));

export const formatTimePeriod = (startMs: number, endMs: number): string => {
    const start = msToTime(startMs);
    const end = msToTime(endMs);
    return `${padTime(start.hours)}:${padTime(start.minutes)} \u2013 ${padTime(end.hours)}:${padTime(end.minutes)}`;
};

export const getNextTimeValue = ({ hours, minutes }: TimeValue): TimeValue | null => {
    if (hours === 23 && minutes === 59) {
        return null;
    }

    if (minutes === 59) {
        return { hours: hours + 1, minutes: 0 };
    }

    return { hours, minutes: minutes + 1 };
};

export const getNormalizedEndTime = (start: TimeValue, end: TimeValue): TimeValue | null => {
    const startMs = timeToMs(start.hours, start.minutes);
    const endMs = timeToMs(end.hours, end.minutes);

    if (endMs > startMs) {
        return end;
    }

    return getNextTimeValue(start);
};

export const getEndTimeOptions = (
    start: TimeValue,
    selectedEndHour: number,
): {
    hours: ScheduleTimeOption[];
    minutes: ScheduleTimeOption[];
    hasAvailableEndTime: boolean;
} => {
    const hasAvailableEndTime = getNextTimeValue(start) !== null;

    return {
        hasAvailableEndTime,
        hours: HOURS_OPTIONS.map((option) => ({
            ...option,
            isDisabled: !hasAvailableEndTime || option.value < start.hours,
        })),
        minutes: MINUTES_OPTIONS.map((option) => ({
            ...option,
            isDisabled:
                !hasAvailableEndTime ||
                selectedEndHour < start.hours ||
                (selectedEndHour === start.hours && option.value <= start.minutes),
        })),
    };
};

export const isFullDay = (start: number, end: number): boolean => {
    return start === 0 && end === FULL_DAY_END_MS;
};

export const getLocalTimezone = (): string => {
    return Intl.DateTimeFormat().resolvedOptions().timeZone;
};
